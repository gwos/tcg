package main

import (
	"bytes"
	"fmt"
	"github.com/gwos/tcg/config"
	"github.com/gwos/tcg/connectors"
	"github.com/gwos/tcg/log"
	"github.com/gwos/tcg/services"
	"github.com/gwos/tcg/transit"
	"github.com/robfig/cron/v3"
	"os/exec"
)

// Variables to control connector version and build time.
// Can be overridden during the build step.
// See README for details.
var (
	buildTime = "Build time not provided"
	buildTag  = "8.x.x"

	extConfig         = &ExtConfig{}
	metricsProfile    = &transit.MetricsProfile{}
	monitorConnection = &connectors.MonitorConnection{
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
	config.Version.Tag = buildTag
	config.Version.Time = buildTime
	log.Info(fmt.Sprintf("[Checker Connector]: Version: %s   /   Build time: %s", config.Version.Tag, config.Version.Time))

	connectors.SigTermHandler(func() {
		if sch != nil {
			sch.Stop()
		}
	})

	transitService := services.GetTransitService()
	transitService.ConfigHandler = configHandler

	log.Info("[Checker Connector]: Waiting for configuration to be delivered ...")
	if err := transitService.DemandConfig(initializeEntrypoints()...); err != nil {
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
	tMonConn := &connectors.MonitorConnection{Extensions: tExt}
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
		sch.AddFunc(task.Cron, taskHandler(task))
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
			res []byte
			err error
		)
		if task.CombinedOutput {
			res, err = cmd.CombinedOutput()
		} else {
			res, err = cmd.Output()
		}
		// TODO: parse output, check inventory changes
		fmt.Printf("###\n%v\n%s\n%v\n", cmd, res, err)
	}
}
