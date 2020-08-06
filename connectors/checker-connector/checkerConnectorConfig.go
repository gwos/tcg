package main

import (
	"github.com/gwos/tcg/config"
	"github.com/gwos/tcg/transit"
	"time"
)

type CheckerConnectorConfig struct {
	Timer time.Duration
}

func InitConfig(monitorConnection *transit.MonitorConnection, metricsProfile *transit.MetricsProfile,
	gwConnections config.GWConnections) *CheckerConnectorConfig {

	return nil
}
