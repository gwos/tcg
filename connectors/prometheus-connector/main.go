package main

import (
	"bytes"
	"fmt"
	"github.com/gwos/tcg/config"
	"github.com/gwos/tcg/connectors"
	"github.com/gwos/tcg/log"
	"github.com/gwos/tcg/services"
	"time"
)

// Variables to control connector version and build time.
// Can be overridden during the build step.
// See README for details.
var (
	buildTime = "Build time not provided"
	buildTag  = "8.1.0"
)

func main() {
	connectors.ControlCHandler()

	var transitService = services.GetTransitService()
	var cfg PrometheusConnectorConfig
	var chksum []byte

	config.Version.Tag = buildTag
	config.Version.Time = buildTime

	log.Info(fmt.Sprintf("[Prometheus Connector]: Version: %s   /   Build time: %s", config.Version.Tag, config.Version.Time))

	transitService.ConfigHandler = func(data []byte) {
		log.Info("[Prometheus Connector]: Configuration received")
		if monitorConn, profile, gwConnections, err := connectors.RetrieveCommonConnectorInfo(data); err == nil {
			c := InitConfig(monitorConn, profile, gwConnections)
			cfg = *c
			chk, err := connectors.Hashsum(
				config.GetConfig().Connector.AgentID,
				config.GetConfig().GWConnections,
				cfg,
			)
			if err != nil || !bytes.Equal(chksum, chk) {
				log.Info("[Prometheus Connector]: Sending inventory ...")
				_ = connectors.SendInventory(
					*Synchronize(),
					cfg.Groups,
					cfg.Ownership,
				)
			}
			if err == nil {
				chksum = chk
			}
		} else {
			log.Error("[Prometheus Connector]: Error during parsing config. Aborting ...")
			return
		}
	}

	log.Info("[Prometheus Connector]: Waiting for configuration to be delivered ...")
	if err := transitService.DemandConfig(initializeEntrypoints()...); err != nil {
		log.Error(err)
		return
	}

	if err := connectors.Start(); err != nil {
		log.Error(err)
		return
	}

	log.Info("[Prometheus Connector]: Waiting for metrics ...")
	for {
		time.Sleep(1 * time.Minute)
	}
}
