package main

import (
	"bytes"
	"context"

	"github.com/gwos/tcg/config"
	"github.com/gwos/tcg/connectors"
	_ "github.com/gwos/tcg/docs"
	"github.com/gwos/tcg/sdk/transit"
	"github.com/gwos/tcg/services"
	"github.com/rs/zerolog/log"
)

var (
	extConfig         = &ExtConfig{}
	metricsProfile    = &transit.MetricsProfile{}
	monitorConnection = &transit.MonitorConnection{
		Extensions: extConfig,
	}
	chksum            []byte
	ctxCancel, cancel = context.WithCancel(context.Background())
)

// @title TCG API Documentation
// @version 1.0

// @host localhost:8099
// @BasePath /api/v1
func main() {
	go handleCache()

	services.GetController().RegisterEntrypoints(initializeEntrypoints())

	transitService := services.GetTransitService()
	transitService.RegisterConfigHandler(configHandler)
	transitService.RegisterExitHandler(cancel)

	log.Info().Msg("waiting for configuration to be delivered ...")
	if err := transitService.DemandConfig(); err != nil {
		log.Err(err).Msg("could not demand the configuration")
		return
	}

	if err := connectors.Start(); err != nil {
		log.Err(err).Msg("could not start connector")
		return
	}

	connectors.StartPeriodic(ctxCancel, connectors.CheckInterval, periodicHandler)

	/* return on quit signal */
	<-transitService.Quit()
}

func handleCache() {
	connectors.ProcessesCache.SetDefault("processes", collectProcesses())
}

func configHandler(data []byte) {
	log.Info().Msg("configuration received")
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
		log.Err(err).Msg("could not parse config")
		return
	}
	/* Update config with received values */
	gwConnections := config.GetConfig().GWConnections
	if len(gwConnections) > 0 {
		tExt.Ownership = transit.HostOwnershipType(gwConnections[0].DeferOwnership)
	}
	extConfig, metricsProfile, monitorConnection = tExt, tMetProf, tMonConn
	monitorConnection.Extensions = extConfig
	/* Process checksums */
	chk, err := connectors.Hashsum(
		config.GetConfig().Connector.AgentID,
		config.GetConfig().GWConnections,
		metricsProfile,
		extConfig,
	)
	if err != nil || !bytes.Equal(chksum, chk) {
		log.Info().Msg("sending inventory ...")
		resources := []transit.InventoryResource{*Synchronize(metricsProfile.Metrics)}
		groups := extConfig.Groups
		for i, group := range groups {
			groups[i] = connectors.FillGroupWithResources(group, resources)
		}
		_ = connectors.SendInventory(
			context.Background(),
			resources,
			groups,
			extConfig.Ownership,
		)
	}
	if err == nil {
		chksum = chk
	}
	/* Restart periodic loop */
	cancel()
	ctxCancel, cancel = context.WithCancel(context.Background())
	services.GetTransitService().RegisterExitHandler(cancel)
	connectors.StartPeriodic(ctxCancel, connectors.CheckInterval, periodicHandler)
}

func periodicHandler() {
	if len(metricsProfile.Metrics) > 0 {
		log.Info().Msg("monitoring resources ...")
		if err := connectors.SendMetrics(context.Background(), []transit.MonitoredResource{
			*CollectMetrics(metricsProfile.Metrics),
		}, nil); err != nil {
			log.Err(err).Msg("could not send metrics")
		}
	}
}
