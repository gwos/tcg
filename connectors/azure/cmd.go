package azure

import (
	"context"

	"github.com/gwos/tcg/config"
	"github.com/gwos/tcg/connectors"
	"github.com/gwos/tcg/connectors/azure/utils"
	"github.com/gwos/tcg/sdk/transit"
	"github.com/gwos/tcg/services"
	"github.com/rs/zerolog/log"
)

var (
	extConfig         = &ExtConfig{}
	monitorConnection = &transit.MonitorConnection{
		Extensions: extConfig,
	}
	_                 = &transit.MetricsProfile{}
	_                 = &transit.Mappings{}
	ctxCancel, cancel = context.WithCancel(context.Background())
)

func Run() {
	transitService := services.GetTransitService()
	transitService.RegisterConfigHandler(configHandler)
	transitService.RegisterExitHandler(cancel)

	log.Info().Msg("waiting for configuration to be delivered")
	if err := transitService.DemandConfig(); err != nil {
		log.Err(err).Msg("failed to demand config")
		return
	}

	if err := connectors.Start(); err != nil {
		log.Err(err).Msg("failed to start connector")
		return
	}

	connectors.StartPeriodic(ctxCancel, connectors.CheckInterval, collectMetrics)

	/* return on quit signal */
	<-transitService.Quit()
}

func configHandler(data []byte) {
	log.Info().Msg("configuration received")

	tExt := &ExtConfig{
		CheckInterval: connectors.DefaultCheckInterval,
	}
	tMonConn := &transit.MonitorConnection{Extensions: tExt}
	tMetProf := &transit.MetricsProfile{}
	if err := connectors.UnmarshalConfig(data, tMetProf, tMonConn); err != nil {
		log.Err(err).Msg("failed to parse config")
		return
	}
	/* Update config with received values */
	gwConnections := config.GetConfig().GWConnections
	if len(gwConnections) > 0 {
		tExt.Ownership = transit.HostOwnershipType(gwConnections[0].DeferOwnership)
	}
	extConfig, _, monitorConnection = tExt, tMetProf, tMonConn
	monitorConnection.Extensions = extConfig
	var err error
	_, err = utils.UnmarshalMappings(data)
	if err != nil {
		log.Err(err).Msg("failed to parse config")
		return
	}
	/* Restart periodic loop */
	cancel()
	ctxCancel, cancel = context.WithCancel(context.Background())
	services.GetTransitService().RegisterExitHandler(cancel)
	connectors.StartPeriodic(ctxCancel, connectors.CheckInterval, collectMetrics)
}
