package databricks

import (
	"context"
	"time"

	"github.com/gwos/tcg/config"
	"github.com/gwos/tcg/connectors"
	"github.com/gwos/tcg/sdk/transit"
	"github.com/gwos/tcg/services"
	"github.com/rs/zerolog/log"
)

const defaultCheckInterval = time.Minute

var (
	extConfig         = &ExtConfig{}
	monitorConnection = &transit.MonitorConnection{
		Extensions: extConfig,
	}
	ctxCancel, cancel = context.WithCancel(context.Background())

	lastRunTimeTo  = time.Now().Add(-time.Hour * 744)
	activeJobsRuns = make(map[int64]int64) // nolint:unused
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

	connectors.StartPeriodic(ctxCancel, defaultCheckInterval, collectMetrics)

	/* return on quit signal */
	<-transitService.Quit()
}

func configHandler(data []byte) {
	log.Info().Msg("configuration received")

	tExt := &ExtConfig{
		CheckInterval: defaultCheckInterval,
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

	tExt.GWMapping.Prepare()

	/* Restart periodic loop */
	cancel()
	ctxCancel, cancel = context.WithCancel(context.Background())
	services.GetTransitService().RegisterExitHandler(cancel)
	connectors.StartPeriodic(ctxCancel, extConfig.CheckInterval, collectMetrics)
}
