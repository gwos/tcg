package main

import (
	"context"

	"github.com/gwos/tcg/config"
	"github.com/gwos/tcg/connectors"
	"github.com/gwos/tcg/services"
	"github.com/gwos/tcg/transit"
	"github.com/rs/zerolog/log"
)

var (
	extConfig         = &ExtConfig{}
	monitorConnection = &transit.MonitorConnection{
		Extensions: extConfig,
	}
	ctxCancel, cancel = context.WithCancel(context.Background())
)

func main() {
	services.GetController().RegisterEntrypoints(initializeEntrypoints())

	transitService := services.GetTransitService()
	transitService.RegisterConfigHandler(configHandler)
	transitService.RegisterExitHandler(cancel)

	log.Info().Msg("waiting for configuration to be delivered ...")
	if err := transitService.DemandConfig(); err != nil {
		log.Err(err).Msg("could not demand config")
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

func configHandler(data []byte) {
	log.Info().Msg("configuration received")
	/* Init config with default values */
	tExt := &ExtConfig{
		Groups:        []transit.ResourceGroup{},
		Resources:     []Resource{},
		Services:      []string{},
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
	extConfig, _, monitorConnection = tExt, tMetProf, tMonConn
	monitorConnection.Extensions = extConfig
	/* Restart periodic loop */
	cancel()
	ctxCancel, cancel = context.WithCancel(context.Background())
	services.GetTransitService().RegisterExitHandler(cancel)
	connectors.StartPeriodic(ctxCancel, connectors.CheckInterval, periodicHandler)
}

func periodicHandler() {
	pull(extConfig.Resources)
}
