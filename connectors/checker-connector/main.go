package main

import (
	"bytes"
	"github.com/gwos/tcg/connectors"
	"github.com/gwos/tcg/log"
	"github.com/gwos/tcg/services"
	"github.com/gwos/tcg/transit"
	"github.com/robfig/cron/v3"
	"os/exec"
)

var (
	extConfig         = &ExtConfig{}
	metricsProfile    = &transit.MetricsProfile{}
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

	log.Info("[Checker Connector]: Waiting for configuration to be delivered ...")
	if err := transitService.DemandConfig(); err != nil {
		log.Error(err)
		return
	}
	if err := connectors.Start(); err != nil {
		log.Error(err)
		return
	}

	/* prevent return */
	<-make(chan bool, 1)
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
	extConfig, metricsProfile, monitorConnection = tExt, tMetProf, tMonConn
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
		)
		if task.CombinedOutput {
			handler = cmd.CombinedOutput
		} else {
			handler = cmd.Output
		}
		res, err = handler()
		logEntry := log.With(log.Fields{"task": task, "res": string(res)})
		if err != nil {
			logEntry.Warn("[Checker Connector]: Error running command:", err.Error())
			return
		}
		logEntry.Debug("[Checker Connector]: Success in command execution")
		if err = processMetrics(res, NSCA); err != nil {
			log.Warn("[Checker Connector]: Error processing metrics:", err.Error())
		}
	}
}
