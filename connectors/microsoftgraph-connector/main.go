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
	"k8s.io/utils/env"
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
	// services.GetController().RegisterEntrypoints(initializeEntrypoints())

	transitService := services.GetTransitService()
	transitService.RegisterConfigHandler(configHandler)
	transitService.RegisterExitHandler(cancel)

	/////////////////////////////////////////////////////////////////////////////////////////////
	// TODO: get these from configuration
	tenantId := env.GetString("MICROSOFT_TENANT_ID", "NOT SET")
	clientId := env.GetString("MICROSOFT_CLIENT_ID", "NOT SET")
	clientSecret := env.GetString("MICROSOFT_CLIENT_SECRET", "NOT SET")
	// TODO: get these options from configuration (Views)
	enableOneDriveMetrics, _ := env.GetBool("ENABLE_ONEDRIVE_METRICS", false)
	enableLicensingMetrics, _ := env.GetBool("ENABLE_LICENSING_METRICS", false)
	enableSharePointMetrics, _ := env.GetBool("ENABLE_SHAREPOINT_METRICS", false)
	enableEmailMetrics, _ := env.GetBool("ENABLE_EMAIL_METRICS", false)
	enableSecurityMetrics, _ := env.GetBool("ENABLE_SECURITY_METRICS", false)
	sharePointSite := env.GetString("SHAREPOINT_SITE", "")
	sharePointSubSite := env.GetString("SHAREPOINT_SUBSITE", "")
	outlookEmailAddress := env.GetString("OUTLOOK_EMAIL_ADDRESS", "")
	connector.SetCredentials(tenantId, clientId, clientSecret)

	//enableOneDriveMetrics = true
	//enableLicensingMetrics = true
	//enableSharePointMetrics = true
	//sharePointSite = "gwosjoey.sharepoint.com"
	//sharePointSubSite = "GWOS"
	//enableEmailMetrics = true
	//outlookEmailAddress = "davidt@gwosjoey.onmicrosoft.com"
	//enableSecurityMetrics = true

	connector.SetOptions(enableOneDriveMetrics, enableLicensingMetrics, enableSharePointMetrics, enableEmailMetrics,
		enableSecurityMetrics, sharePointSite, sharePointSubSite, outlookEmailAddress)
	//////////////////////////////////////////////////////////////////////////////////////////////////

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
		//if tempslateMetricName == metric.Name || !metric.Monitored {
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
	/* Restart periodic loop */
	cancel()
	ctxCancel, cancel = context.WithCancel(context.Background())
	services.GetTransitService().RegisterExitHandler(cancel)
	connectors.StartPeriodic(ctxCancel, connectors.CheckInterval, periodicHandler)
}

func periodicHandler() {
	temporaryMetricsDefinitions() // TODO: remove this when you have provisioning ready
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
	time.Sleep(3 * time.Second) // TODO: better way to assure synch completion?
	err := connectors.SendMetrics(context.Background(), monitored, &groups)
	log.Err(err).Msg("sending metrics")
}

// TODO: remove this when you have provisioning ready
func temporaryMetricsDefinitions() {
	//bytes, _ := ioutil.ReadFile("/Users/dtaylor/gw8/tcg/connectors/microsoftgraph-connector/microsoftgraph-default.json")
	//configHandler(bytes)
}
