package main

import (
	"bytes"
	"context"
	"fmt"
	"github.com/gwos/tcg/config"
	"github.com/gwos/tcg/connectors"
	"github.com/gwos/tcg/log"
	"github.com/gwos/tcg/services"
	"github.com/gwos/tcg/transit"
	"k8s.io/utils/env"
	"time"
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
	tenantId := env.GetString("MICROSOFT_TENANT_ID", "NOT SET")
	clientId := env.GetString("MICROSOFT_CLIENT_ID", "NOT SET")
	clientSecret := env.GetString("MICROSOFT_CLIENT_SECRET", "NOT SET")
	connector.SetCredentials(tenantId, clientId, clientSecret)
	log.Info("[MsGraph Connector]: Waiting for configuration to be delivered ...")
	if err := transitService.DemandConfig(); err != nil {
		log.Error("[MsGraph Connector]: ", err)
		return
	}

	if err := connectors.Start(); err != nil {
		log.Error(err)
		return
	}

	log.Info("[MsGraph Connector]: Starting metric connection ...")
	connectors.StartPeriodic(ctxCancel, connectors.CheckInterval, periodicHandler)

	/* return on quit signal */
	<-transitService.Quit()
}

func configHandler(data []byte) {
	log.Info("[K8 Connector]: Configuration received")
	/* Init config with default values */
	tExt := &ExtConfig{
		AppType:   config.GetConfig().Connector.AppType,
		AppName:   config.GetConfig().Connector.AppName,
		AgentID:   config.GetConfig().Connector.AgentID,
		EndPoint:  "gwos.bluesunrise.com:8001", // TODO: hardcoded
		Ownership: transit.Yield,
		Views:     make(map[MicrosoftGraphView]map[string]transit.MetricDefinition),
		Groups:    []transit.ResourceGroup{},
	}
	tMonConn := &transit.MonitorConnection{Extensions: tExt}
	tMetProf := &transit.MetricsProfile{}
	if err := connectors.UnmarshalConfig(data, tMetProf, tMonConn); err != nil {
		log.Error("[K8 Connector]: Error during parsing config.", err.Error())
		return
	}
	/* Update config with received values */
	// TODO: fudge up some metrics - remove this once we hook in live metrics, apptype
	tExt.Views[ViewServices] = temporaryMetricsDefinitions()
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
	monitorConnection.Extensions = extConfig
	/* Process checksums */
	chk, err := connectors.Hashsum(extConfig)
	if err != nil || !bytes.Equal(chksum, chk) {
		// TODO: process inventory
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
	inventory, monitored, groups := connector.Collect(extConfig)
	log.Debug("[MsGraph Connector]: ", fmt.Sprintf("%d:%d:%d", len(inventory), len(monitored), len(groups)))

	if count < 2 {
		if err := connectors.SendInventory(
			context.Background(),
			inventory,
			groups,
			extConfig.Ownership,
		); err != nil {
			log.Error("[K8 Connector]: Error during sending inventory.", err)
		}
		count = count + 1
	}
	time.Sleep(3 * time.Second) // TODO: better way to assure synch completion?
	if err := connectors.SendMetrics(context.Background(), monitored, &groups); err != nil {
		log.Error("[MsGraph Connector]: Error during sending metrics.", err)
	}
}

// TODO: remove this
func temporaryMetricsDefinitions() map[string]transit.MetricDefinition {
	metrics := make(map[string]transit.MetricDefinition)
	return metrics
}


