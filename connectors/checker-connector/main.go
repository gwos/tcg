package main

import (
	"bytes"
	"context"
	"os/exec"

	"github.com/gwos/tcg/connectors"
	"github.com/gwos/tcg/connectors/nsca-connector/parser"
	"github.com/gwos/tcg/sdk/transit"
	"github.com/gwos/tcg/services"
	"github.com/gwos/tcg/tracing"
	"github.com/robfig/cron/v3"
	"github.com/rs/zerolog/log"
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
	transitService := services.GetTransitService()
	transitService.RegisterConfigHandler(configHandler)
	transitService.RegisterExitHandler(func() {
		if sch != nil {
			sch.Stop()
		}
	})

	log.Info().Msg("waiting for configuration to be delivered ...")
	if err := transitService.DemandConfig(); err != nil {
		log.Err(err).Msg("could not demand config")
		return
	}
	if err := connectors.Start(); err != nil {
		log.Err(err).Msg("could not start connector")
		return
	}

	/* return on quit signal */
	<-transitService.Quit()
}

func configHandler(data []byte) {
	log.Info().Msg("configuration received")
	tExt, tMetProf := &ExtConfig{}, &transit.MetricsProfile{}
	tMonConn := &transit.MonitorConnection{Extensions: tExt}
	if err := connectors.UnmarshalConfig(data, tMetProf, tMonConn); err != nil {
		log.Err(err).Msg("could not parse config")
		return
	}
	if err := tExt.Validate(); err != nil {
		log.Err(err).Msg("could not validate config")
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
			handler func() ([]byte, error)
			err     error
			res     []byte

			monitoredResources *[]transit.DynamicMonitoredResource
		)
		if task.CombinedOutput {
			handler = cmd.CombinedOutput
		} else {
			handler = cmd.Output
		}

		ctx, span := tracing.StartTraceSpan(context.Background(), "connectors", "taskHandler")
		defer func() {
			tracing.EndTraceSpan(span,
				tracing.TraceAttrError(err),
				tracing.TraceAttrPayloadLen(res),
				tracing.TraceAttrStr("task", task.String()),
			)
		}()

		_, span2 := tracing.StartTraceSpan(ctx, "connectors", "command")
		res, err = handler()

		tracing.EndTraceSpan(span2,
			tracing.TraceAttrError(err),
			tracing.TraceAttrPayloadLen(res),
			tracing.TraceAttrStrs("command", task.Command),
		)

		if err != nil {
			log.Warn().Err(err).
				Interface("task", task).
				Bytes("res", res).
				Msg("task failed")
			return
		}
		log.Debug().
			Interface("task", task).
			Bytes("res", res).
			Msg("task done")

		_, span3 := tracing.StartTraceSpan(ctx, "connectors", "parse")
		monitoredResources, err = parser.Parse(res, task.DataFormat)

		tracing.EndTraceSpan(span3,
			tracing.TraceAttrError(err),
			tracing.TraceAttrPayloadLen(res),
		)

		if err != nil {
			log.Warn().Err(err).
				Interface("task", task).
				Bytes("res", res).
				Msg("could not parse metrics")
			return
		}
		if err = connectors.SendMetrics(ctx, *monitoredResources, nil); err != nil {
			log.Warn().Err(err).
				Interface("task", task).
				Bytes("res", res).
				Msg("could not send metrics")
		}
	}
}
