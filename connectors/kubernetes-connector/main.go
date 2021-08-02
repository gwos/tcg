package main

import (
	"bytes"
	"context"
	"io/ioutil"
	"strings"
	"time"

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
	chksum            []byte
	connector         KubernetesConnector
	ctxCancel, cancel = context.WithCancel(context.Background())
	count             = 0
)

func main() {
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
		EndPoint:  defaultKubernetesClusterEndpoint,
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

	if tMonConn.Extensions.(*ExtConfig).AuthType == ConfigFile {
		if err := writeDataToFile([]byte(tMonConn.Extensions.(*ExtConfig).KubernetesConfigFile)); err != nil {
			log.Err(err).Msg("could not write to file")
		}
	}

	/* Update config with received values */
	tExt.Views[ViewNodes] = buildNodeMetricsMap(tMetProf.Metrics)
	tExt.Views[ViewPods] = buildPodMetricsMap(tMetProf.Metrics)

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

	extConfig, monitorConnection = tExt, tMonConn
	monitorConnection.Extensions = extConfig

	/* Process checksums */
	chk, err := connectors.Hashsum(extConfig)
	if err != nil || !bytes.Equal(chksum, chk) {
		// TODO: process inventory
	}
	if err == nil {
		chksum = chk
	}

	if err = connector.Initialize(*monitorConnection.Extensions.(*ExtConfig)); err != nil {
		log.Err(err).Msg("could not initialize connector")
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
		time.Sleep(3 * time.Second) // TODO: better way to assure sync completion?
		err := connectors.SendMetrics(context.Background(), monitored, &groups)
		log.Err(err).Msg("sending metrics")
	}
}

func buildNodeMetricsMap(metricsArray []transit.MetricDefinition) map[string]transit.MetricDefinition {
	metrics := make(map[string]transit.MetricDefinition)
	for _, metric := range metricsArray {
		if metric.ServiceType == string(ViewNodes) {
			metrics[metric.Name] = metric
		}
	}

	// TODO: storage is not supported yet
	return metrics
}

func buildPodMetricsMap(metricsArray []transit.MetricDefinition) map[string]transit.MetricDefinition {
	metrics := make(map[string]transit.MetricDefinition)
	for _, metric := range metricsArray {
		if metric.ServiceType == string(ViewPods) {
			metrics[metric.Name] = metric
		}
	}

	// TODO: storage is not supported yet
	return metrics
}

func writeDataToFile(data []byte) error {
	strPath := config.GetConfig().ConfigPath()
	strArray := strings.Split(strPath, "/")
	finalPath := ""
	for i := 0; i < len(strArray)-1; i++ {
		finalPath += strArray[i] + "/"
	}
	finalPath += "kubernetes_config.yaml"
	return ioutil.WriteFile(finalPath, data, 0644)
}
