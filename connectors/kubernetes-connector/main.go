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
	connector         KubernetesConnector
	ctxCancel, cancel = context.WithCancel(context.Background())
	count             = 0
)

func main() {
	// services.GetController().RegisterEntrypoints(initializeEntrypoints())

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
		AppType:   config.GetConfig().Connector.AppType,
		AppName:   config.GetConfig().Connector.AppName,
		AgentID:   config.GetConfig().Connector.AgentID,
		EndPoint:  "gwos.bluesunrise.com:8001", // TODO: hardcoded
		Ownership: transit.Yield,
		Views:     make(map[KubernetesView]map[string]transit.MetricDefinition),
		Groups:    []transit.ResourceGroup{},
	}
	tMonConn := &transit.MonitorConnection{Extensions: tExt}
	tMetProf := &transit.MetricsProfile{}
	if err := connectors.UnmarshalConfig(data, tMetProf, tMonConn); err != nil {
		log.Err(err).Msg("could not parse config")
		return
	}
	/* Update config with received values */
	// TODO: fudge up some metrics - remove this once we hook in live metrics, apptype
	tExt.Views[ViewNodes] = fudgeUpNodeMetricDefinitions()
	tExt.Views[ViewPods] = fudgeUpPodMetricDefinitions()
	for _, metric := range tMetProf.Metrics {
		// temporary solution, will be removed
		// TODO: push down into connectors - metric.Monitored breaks synthetics
		//if templateMetricName == metric.Name || !metric.Monitored {
		//	continue
		//}
		if metrics, has := tExt.Views[KubernetesView(metric.ServiceType)]; has {
			metrics[metric.Name] = metric
			tExt.Views[KubernetesView(metric.ServiceType)] = metrics
		} else {
			metrics := make(map[string]transit.MetricDefinition)
			metrics[metric.Name] = metric
			if tExt.Views != nil {
				tExt.Views[KubernetesView(metric.ServiceType)] = metrics
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
	if connector.kapi != nil {
		inventory, monitored, groups := connector.Collect(extConfig)
		log.Debug().Msgf("collected %d:%d:%d", len(inventory), len(monitored), len(groups))

		if count == 0 {
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
}

// TODO: remove this
func fudgeUpNodeMetricDefinitions() map[string]transit.MetricDefinition {
	metrics := make(map[string]transit.MetricDefinition)
	metrics["cpu"] = transit.MetricDefinition{
		Name:              "cpu",
		Monitored:         true,
		Graphed:           true,
		ComputeType:       transit.Query,
		ServiceType:       "Node",
		WarningThreshold:  -1,
		CriticalThreshold: -1,
	}
	metrics["cpu.cores"] = transit.MetricDefinition{
		Name:              "cpu.cores",
		Monitored:         false,
		Graphed:           false,
		ComputeType:       transit.Informational,
		ServiceType:       "Node",
		WarningThreshold:  -1,
		CriticalThreshold: -1,
	}
	metrics["cpu.allocated"] = transit.MetricDefinition{
		Name:              "cpu.allocated",
		Monitored:         false,
		Graphed:           false,
		ComputeType:       transit.Informational,
		ServiceType:       "Node",
		WarningThreshold:  -1,
		CriticalThreshold: -1,
	}
	metrics["memory"] = transit.MetricDefinition{
		Name:              "memory",
		Monitored:         true,
		Graphed:           true,
		ComputeType:       transit.Query,
		ServiceType:       "Node",
		WarningThreshold:  -1,
		CriticalThreshold: -1,
	}
	metrics["memory.capacity"] = transit.MetricDefinition{
		Name:              "memory.capacity",
		Monitored:         false,
		Graphed:           false,
		ComputeType:       transit.Informational,
		ServiceType:       "Node",
		WarningThreshold:  -1,
		CriticalThreshold: -1,
	}
	metrics["memory.allocated"] = transit.MetricDefinition{
		Name:              "memory.allocated",
		Monitored:         false,
		Graphed:           false,
		ComputeType:       transit.Informational,
		ServiceType:       "Node",
		WarningThreshold:  -1,
		CriticalThreshold: -1,
	}

	metrics["pods"] = transit.MetricDefinition{
		Name:              "pods",
		Monitored:         true,
		Graphed:           true,
		ComputeType:       transit.Query,
		ServiceType:       "Node",
		WarningThreshold:  -1,
		CriticalThreshold: -1,
	}
	// TODO: storage is not supported yet
	return metrics
}

func fudgeUpPodMetricDefinitions() map[string]transit.MetricDefinition {
	metrics := make(map[string]transit.MetricDefinition)
	metrics["cpu"] = transit.MetricDefinition{
		Name:              "cpu",
		Monitored:         true,
		Graphed:           true,
		ComputeType:       transit.Query,
		ServiceType:       "Pod",
		WarningThreshold:  -1,
		CriticalThreshold: -1,
	}
	metrics["memory"] = transit.MetricDefinition{
		Name:              "memory",
		Monitored:         true,
		Graphed:           true,
		ComputeType:       transit.Query,
		ServiceType:       "Pod",
		WarningThreshold:  -1,
		CriticalThreshold: -1,
	}
	// TODO: storage is not supported yet
	return metrics
}
