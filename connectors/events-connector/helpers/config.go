package helpers

import (
	"context"

	"github.com/gwos/tcg/connectors"
	"github.com/gwos/tcg/sdk/mapping"
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

	_, cancel = context.WithCancel(context.Background())
)

type ExtConfig struct {
	MapHostgroup mapping.Mappings `json:"mapHostgroup"`
	MapHostname  mapping.Mappings `json:"mapHostname"`
	MapService   mapping.Mappings `json:"mapService"`
}

func GetExtConfig() *ExtConfig {
	return extConfig
}

func GetCancelFunc() *context.CancelFunc {
	return &cancel
}

func GetMonitorConnection() *transit.MetricsProfile {
	return metricsProfile
}

func GetMetricsProfile() *transit.MonitorConnection {
	return monitorConnection
}

func ConfigHandler(data []byte) {
	log.Info().Msg("configuration received")
	/* Init config with default values */
	tExt := &ExtConfig{}
	tMonConn := &transit.MonitorConnection{Extensions: tExt}
	tMetProf := &transit.MetricsProfile{}
	if err := connectors.UnmarshalConfig(data, tMetProf, tMonConn); err != nil {
		log.Err(err).Msg("failed to parse config")
		return
	}
	/* Update config with received values */
	extConfig, metricsProfile, monitorConnection = tExt, tMetProf, tMonConn
	if err := extConfig.MapHostgroup.Compile(); err != nil {
		log.Err(err).Msg("failed to compile host group mappings")
		return
	}
	if err := extConfig.MapHostname.Compile(); err != nil {
		log.Err(err).Msg("failed to compile host mappings")
		return
	}
	if err := extConfig.MapService.Compile(); err != nil {
		log.Err(err).Msg("failed to compile service mappings")
		return
	}
	monitorConnection.Extensions = extConfig
	/* Restart periodic loop */
	cancel()
	_, cancel = context.WithCancel(context.Background())
	services.GetTransitService().RegisterExitHandler(cancel)
}
