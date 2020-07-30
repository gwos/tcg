package main

import (
	"bytes"
	"fmt"
	"github.com/gwos/tcg/cache"
	"github.com/gwos/tcg/config"
	"github.com/gwos/tcg/connectors"
	_ "github.com/gwos/tcg/docs"
	"github.com/gwos/tcg/log"
	"github.com/gwos/tcg/services"
	"github.com/gwos/tcg/transit"
)

// Variables to control connector version and build time.
// Can be overridden during the build step.
// See README for details.
var (
	buildTime = "Build time not provided"
	buildTag  = "8.1.0"
)

// @title TCG API Documentation
// @version 1.0

// @host localhost:8099
// @BasePath /api/v1
func main() {
	connectors.SigTermHandler()
	go handleCache()

	var transitService = services.GetTransitService()
	var cfg ServerConnectorConfig
	var chksum []byte

	config.Version.Tag = buildTag
	config.Version.Time = buildTime

	log.Info(fmt.Sprintf("[Server Connector]: Version: %s   /   Build time: %s", config.Version.Tag, config.Version.Time))

	transitService.ConfigHandler = func(data []byte) {
		log.Info("[Server Connector]: Configuration received")
		if monitorConn, profile, gwConnections, err := connectors.RetrieveCommonConnectorInfo(data); err == nil {
			c := InitConfig(monitorConn, profile, gwConnections)
			cfg = *c
			chk, err := connectors.Hashsum(
				config.GetConfig().Connector.AgentID,
				config.GetConfig().GWConnections,
				cfg,
			)
			if err != nil || !bytes.Equal(chksum, chk) {
				log.Info("[Server Connector]: Sending inventory ...")
				resources := []transit.InventoryResource{*Synchronize(cfg.MetricsProfile.Metrics)}
				groups := cfg.Groups
				for i, group := range groups {
					groups[i] = connectors.FillGroupWithResources(group, resources)
				}
				_ = connectors.SendInventory(
					resources,
					groups,
					cfg.Ownership,
				)
			}
			if err == nil {
				chksum = chk
			}
		} else {
			log.Error("[Server Connector]: Error during parsing config. Aborting ...")
			return
		}
	}

	log.Info("[Server Connector]: Waiting for configuration to be delivered ...")
	if err := transitService.DemandConfig(initializeEntrypoints()...); err != nil {
		log.Error("[Server Connector]: ", err)
		return
	}

	if err := connectors.Start(); err != nil {
		log.Error("[Server Connector]: ", err)
		return
	}

	connectors.StartPeriodic(nil, cfg.Timer, func() {
		if len(cfg.MetricsProfile.Metrics) > 0 {
			log.Info("[Server Connector]: Monitoring resources ...")
			if err := connectors.SendMetrics([]transit.MonitoredResource{
				*CollectMetrics(cfg.MetricsProfile.Metrics),
			}); err != nil {
				log.Error("[Server Connector]: ", err)
			}
		}
	})
}

func handleCache() {
	cache.ProcessesCache.SetDefault("processes", collectProcesses())
}
