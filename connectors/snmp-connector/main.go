package main

import (
	"bytes"
	"context"

	"github.com/gwos/tcg/config"
	"github.com/gwos/tcg/connectors"
	"github.com/gwos/tcg/sdk/transit"
	"github.com/gwos/tcg/services"
	"github.com/rs/zerolog/log"
)

var (
	extConfig         = &ExtConfig{}
	monitorConnection = &transit.MonitorConnection{
		Extensions: extConfig,
	}
	cfgChksum         []byte
	invChksum         []byte
	connector         SnmpConnector
	ctxCancel, cancel = context.WithCancel(context.Background())
)

// temporary solution, will be removed
const templateMetricName = "$view_Template#"

func main() {
	services.GetController().RegisterEntrypoints(initializeEntryPoints())

	transitService := services.GetTransitService()
	transitService.RegisterConfigHandler(configHandler)
	services.GetTransitService().RegisterExitHandler(cancel)

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
		NediServer:    defaultNediServer,
		CheckInterval: connectors.DefaultCheckInterval,
		AppType:       config.GetConfig().Connector.AppType,
		AgentID:       config.GetConfig().Connector.AgentID,
		GWConnections: config.GetConfig().GWConnections,
		Ownership:     transit.Yield,
		Views:         make(map[string]map[string]transit.MetricDefinition),
	}
	tMonConn := &transit.MonitorConnection{Extensions: tExt}
	tMetProf := &transit.MetricsProfile{}
	if err := connectors.UnmarshalConfig(data, tMetProf, tMonConn); err != nil {
		log.Err(err).Msg("could not parse config")
		return
	}

	/* Update config with received values */
	if tMonConn.Server != "" {
		tExt.NediServer = tMonConn.Server
	}

	for _, metric := range tMetProf.Metrics {
		// temporary solution, will be removed
		if templateMetricName == metric.Name || !metric.Monitored {
			continue
		}
		if tExt.Views[metric.ServiceType] != nil {
			tExt.Views[metric.ServiceType][metric.Name] = metric
		} else {
			metrics := make(map[string]transit.MetricDefinition)
			metrics[metric.Name] = metric
			tExt.Views[metric.ServiceType] = metrics
		}
	}
	if len(tExt.GWConnections) > 0 {
		for _, conn := range tExt.GWConnections {
			if conn.DeferOwnership != "" {
				ownership := transit.HostOwnershipType(tExt.GWConnections[0].DeferOwnership)
				if ownership != "" {
					tExt.Ownership = ownership
					break
				}
			}
		}
	}

	extConfig, _, monitorConnection = tExt, tMetProf, tMonConn
	monitorConnection.Extensions = extConfig

	/* Process checksums */
	chk, err := connectors.Hashsum(extConfig)

	if err != nil || !bytes.Equal(cfgChksum, chk) {
		if err := connector.LoadConfig(*extConfig); err != nil {
			log.Err(err).Msg("could not reload SnmpConnector config")
		}
	}
	if err == nil {
		cfgChksum = chk
	}
	/* Restart periodic loop */
	cancel()
	ctxCancel, cancel = context.WithCancel(context.Background())
	services.GetTransitService().RegisterExitHandler(cancel)
	connectors.StartPeriodic(ctxCancel, connectors.CheckInterval, periodicHandler)
}

func periodicHandler() {
	hasMetrics := false
	if len(connector.config.Views) > 0 {
		for _, v := range connector.config.Views {
			if len(v) > 0 {
				hasMetrics = true
				break
			}
		}
	}

	if hasMetrics {
		metrics, inventory, groups, err := connector.CollectMetrics()
		log.Err(err).Msg("collect metrics")
		if err == nil {
			chk, chkErr := connector.getInventoryHashSum()
			if chkErr != nil || !bytes.Equal(invChksum, chk) {
				err := connectors.SendInventory(context.Background(), inventory, groups, connector.config.Ownership)
				log.Err(err).Msg("inventory changed: sending inventory")
			}
			if chkErr == nil {
				invChksum = chk
			}

			err = connectors.SendMetrics(context.Background(), metrics, nil)
			log.Err(err).Msg("sending metrics")
		}
	}
}
