package main

import (
	"bytes"
	"context"
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
	extConfig         = &ExtConfig{}
	metricsProfile    = &transit.MetricsProfile{}
	monitorConnection = &transit.MonitorConnection{
		Extensions: extConfig,
	}
	chksum []byte
)

// @title TCG API Documentation
// @version 1.0

// @host localhost:8099
// @BasePath /api/v1
func main() {
	go handleCache()

	services.GetController().RegisterEntrypoints(initializeEntrypoints())

	ctxExit, exitHandler := context.WithCancel(context.Background())
	transitService := services.GetTransitService()
	transitService.RegisterConfigHandler(configHandler)
	transitService.RegisterExitHandler(exitHandler)

	log.Info("[Server Connector]: Waiting for configuration to be delivered ...")
	if err := transitService.DemandConfig(); err != nil {
		log.Error("[Server Connector]: ", err)
		return
	}

	if err := connectors.Start(); err != nil {
		log.Error("[Server Connector]: ", err)
		return
	}

	connectors.StartPeriodic(ctxExit, extConfig.CheckInterval, func() {
		if len(metricsProfile.Metrics) > 0 {
			log.Info("[Server Connector]: Monitoring resources ...")
			if err := connectors.SendMetrics([]transit.MonitoredResource{
				*CollectMetrics(metricsProfile.Metrics),
			}); err != nil {
				log.Error("[Server Connector]: ", err)
			}
		}
	})
}

func handleCache() {
	cache.ProcessesCache.SetDefault("processes", collectProcesses())
}

func configHandler(data []byte) {
	log.Info("[Server Connector]: Configuration received")
	/* Init config with default values */
	tExt := &ExtConfig{
		Groups: []transit.ResourceGroup{{
			GroupName: defaultHostGroupName,
			Type:      transit.HostGroup,
		}},
		Processes:     []string{},
		CheckInterval: connectors.DefaultCheckInterval,
		Ownership:     transit.Yield,
	}
	tMonConn := &transit.MonitorConnection{Extensions: tExt}
	tMetProf := &transit.MetricsProfile{}
	if err := connectors.UnmarshalConfig(data, tMetProf, tMonConn); err != nil {
		log.Error("[Server Connector]: Error during parsing config.", err.Error())
		return
	}
	/* Update config with received values */
	gwConnections := config.GetConfig().GWConnections
	if len(gwConnections) > 0 {
		tExt.Ownership = transit.HostOwnershipType(gwConnections[0].DeferOwnership)
	}
	extConfig, metricsProfile, monitorConnection = tExt, tMetProf, tMonConn
	monitorConnection.Extensions = extConfig

	chk, err := connectors.Hashsum(
		config.GetConfig().Connector.AgentID,
		config.GetConfig().GWConnections,
		extConfig,
	)
	if err != nil || !bytes.Equal(chksum, chk) {
		log.Info("[Server Connector]: Sending inventory ...")
		resources := []transit.InventoryResource{*Synchronize(metricsProfile.Metrics)}
		groups := extConfig.Groups
		for i, group := range groups {
			groups[i] = connectors.FillGroupWithResources(group, resources)
		}
		_ = connectors.SendInventory(
			resources,
			groups,
			extConfig.Ownership,
		)
	}
	if err == nil {
		chksum = chk
	}
}
