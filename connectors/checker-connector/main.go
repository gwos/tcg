package main

import (
	"bytes"
	"context"
	"os/exec"

	"github.com/gwos/tcg/connectors"
	"github.com/gwos/tcg/log"
	"github.com/gwos/tcg/services"
	"github.com/gwos/tcg/transit"
	"github.com/robfig/cron/v3"
)

var (
	extConfig         = &ExtConfig{}
	monitorConnection = &transit.MonitorConnection{
		Extensions: extConfig,
	}
	chksum []byte

	sch = cron.New(
		cron.WithSeconds(),
		cron.WithChain(
			cron.Recover(cron.DefaultLogger),
			cron.SkipIfStillRunning(cron.DefaultLogger),
		),
	)
)

// @title TCG API Documentation
// @version 1.0

// @host localhost:8099
// @BasePath /api/v1
func main() {
	services.GetController().RegisterEntrypoints(initializeEntrypoints())

	transitService := services.GetTransitService()
	transitService.RegisterConfigHandler(configHandler)
	transitService.RegisterExitHandler(func() {
		if sch != nil {
			sch.Stop()
		}
	})

	log.Info("[Checker Connector]: Waiting for configuration to be delivered ...")
	if err := transitService.DemandConfig(); err != nil {
		log.Error(err)
		return
	}
	if err := connectors.Start(); err != nil {
		log.Error(err)
		return
	}

	/* return on quit signal */
	<-transitService.Quit()
}

func configHandler(data []byte) {
	log.Info("[Checker Connector]: Configuration received")
	tExt, tMetProf := &ExtConfig{}, &transit.MetricsProfile{}
	tMonConn := &transit.MonitorConnection{Extensions: tExt}
	if err := connectors.UnmarshalConfig(data, tMetProf, tMonConn); err != nil {
		log.Error("[Checker Connector]: Error during parsing config.", err.Error())
		return
	}
	if err := tExt.Validate(); err != nil {
		log.Error("[Checker Connector]: Error during parsing config.", err.Error())
		return
	}
	extConfig, _, monitorConnection = tExt, tMetProf, tMonConn
	monitorConnection.Extensions = extConfig

	chk, err := connectors.Hashsum(extConfig)
	if err != nil || !bytes.Equal(chksum, chk) {
		restartScheduler(sch, extConfig.Schedule)
	}
	if err == nil {
		chksum = chk
	}
}

func restartScheduler(sch *cron.Cron, tasks []ScheduleTask) {
	for _, entry := range sch.Entries() {
		sch.Remove(entry.ID)
	}
	for _, task := range tasks {
		_, _ = sch.AddFunc(task.Cron, taskHandler(task))
	}
	if len(sch.Entries()) > 0 {
		sch.Start()
	}
}

func taskHandler(task ScheduleTask) func() {
	return func() {
		cmd := exec.Command(task.Command[0], task.Command[1:]...)
		cmd.Env = task.Environment
		var (
			handler     func() ([]byte, error)
			err         error
			res         []byte
			span, spanN services.TraceSpan
			ctx, ctxN   context.Context
		)
		if task.CombinedOutput {
			handler = cmd.CombinedOutput
		} else {
			handler = cmd.Output
		}

		ctx, span = services.StartTraceSpan(context.Background(), "connectors", "taskHandler")
		defer func() {
			services.EndTraceSpan(span,
				services.TraceAttrError(err),
				services.TraceAttrPayloadLen(res),
				services.TraceAttrString("task", task.String()),
			)
		}()
		_, spanN = services.StartTraceSpan(ctx, "connectors", "command")

		res, err = handler()

		services.EndTraceSpan(spanN,
			services.TraceAttrError(err),
			services.TraceAttrPayloadLen(res),
			services.TraceAttrArray("command", task.Command),
		)

		logEntry := log.With(log.Fields{"task": task, "res": string(res)})
		if err != nil {
			logEntry.Warn("[Checker Connector]: Error running command:", err.Error())
			return
		}
		logEntry.Debug("[Checker Connector]: Success in command execution")

		ctxN, spanN = services.StartTraceSpan(ctx, "connectors", "processMetrics")

		if _, err = processMetrics(ctxN, res, task.DataFormat); err != nil {
			log.Warn("[Checker Connector]: Error processing metrics:", err.Error())
		}

		services.EndTraceSpan(spanN,
			services.TraceAttrError(err),
			services.TraceAttrPayloadLen(res),
		)
	}
}
