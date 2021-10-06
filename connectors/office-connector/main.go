package main

import (
	"bytes"
	"context"
	"time"

	"github.com/gwos/tcg/config"
	"github.com/gwos/tcg/connectors"
	"github.com/gwos/tcg/services"
	"github.com/gwos/tcg/transit"
	"github.com/rs/zerolog/log"
)

var (
	extConfig         = &ExtConfig{}
	metricsProfile    = &transit.MetricsProfile{}
	monitorConnection = &transit.MonitorConnection{
		Extensions: extConfig,
	}
	chksum            []byte
	connector         MicrosoftGraphConnector
	ctxCancel, cancel = context.WithCancel(context.Background())
	count             = 0
)

func main() {
	services.GetController().RegisterEntrypoints(initializeEntrypoints())

	transitService := services.GetTransitService()
	transitService.RegisterConfigHandler(configHandler)
	transitService.RegisterExitHandler(cancel)

	log.Info().Msg("Waiting for configuration to be delivered ...")
	if err := transitService.DemandConfig(); err != nil {
		log.Err(err).Msg("Could not demand config")
		return
	}

	if err := connectors.Start(); err != nil {
		log.Err(err).Msg("Could not start connector")
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
		Ownership: transit.Yield,
		Groups:    []transit.ResourceGroup{},
		Views:     make(map[MicrosoftGraphView]map[string]transit.MetricDefinition),
	}
	tMonConn := &transit.MonitorConnection{Extensions: tExt}
	tMetProf := &transit.MetricsProfile{}
	if err := connectors.UnmarshalConfig(data, tMetProf, tMonConn); err != nil {
		log.Err(err).Msg("could not parse config")
		return
	}
	/* Update config with received values */
	// tExt.Views[ViewServices] = temporaryMetricsDefinitions()
	for _, metric := range tMetProf.Metrics {
		// temporary solution, will be removed
		// TODO: push down into connectors - metric.Monitored breaks synthetics
		//if templateMetricName == metric.Name || !metric.Monitored {
		//	continue
		//}
		if metrics, has := tExt.Views[MicrosoftGraphView(metric.ServiceType)]; has {
			metrics[metric.Name] = metric
			tExt.Views[MicrosoftGraphView(metric.ServiceType)] = metrics
		} else {
			metrics := make(map[string]transit.MetricDefinition)
			metrics[metric.Name] = metric
			if tExt.Views != nil {
				tExt.Views[MicrosoftGraphView(metric.ServiceType)] = metrics
			}
		}
	}
	gwConnections := config.GetConfig().GWConnections
	if len(gwConnections) > 0 {
		for _, conn := range gwConnections {
			if conn.DeferOwnership != "" {
				ownership := transit.HostOwnershipType(gwConnections[0].DeferOwnership)
				if ownership != "" {
					tExt.Ownership = ownership
					break
				}
			}
		}
	}
	extConfig, metricsProfile, monitorConnection = tExt, tMetProf, tMonConn

	for k, _ := range viewStateMap {
		viewStateMap[k] = containsView(metricsProfile.Metrics, k)
	}

	// monitorConnection.Extensions = extConfig
	/* Process checksums */
	chk, err := connectors.Hashsum(extConfig)
	if err != nil || !bytes.Equal(chksum, chk) {
		// TODO: process inventory
	}
	if err == nil {
		chksum = chk
	}

	connector.SetCredentials(extConfig.TenantId, extConfig.ClientId, extConfig.ClientSecret)
	connector.SetOptions(extConfig.SharePointSite, extConfig.SharePointSubsite, extConfig.OutlookEmail)
	/* Restart periodic loop */
	cancel()
	ctxCancel, cancel = context.WithCancel(context.Background())
	services.GetTransitService().RegisterExitHandler(cancel)
	connectors.StartPeriodic(ctxCancel, connectors.CheckInterval, periodicHandler)
}

func periodicHandler() {
	inventory, monitored, groups := connector.Collect(extConfig)
	log.Debug().Msgf("collected %d:%d:%d", len(inventory), len(monitored), len(groups))

	if count > -1 {
		err := connectors.SendInventory(
			context.Background(),
			inventory,
			groups,
			extConfig.Ownership,
		)
		log.Err(err).Msg("sending inventory")
		count = count + 1
	}
	time.Sleep(3 * time.Second) // TODO: better way to assure sync completion?
	err := connectors.SendMetrics(context.Background(), monitored, &groups)
	log.Err(err).Msg("sending metrics")
}
