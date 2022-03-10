package main

import (
	"bytes"
	"context"
	"gopkg.in/yaml.v3"
	"io/ioutil"
	"strings"
	"time"

	"github.com/gwos/tcg/config"
	"github.com/gwos/tcg/connectors"
	"github.com/gwos/tcg/sdk/transit"
	"github.com/gwos/tcg/services"
	"github.com/rs/zerolog/log"
)

var (
	connector         KubernetesConnector
	chksum            []byte
	fresh             = true
	extConfig         = &ExtConfig{}
	ctxCancel, cancel = context.WithCancel(context.Background())
	monitorConnection = &transit.MonitorConnection{
		Extensions: extConfig,
	}
)

func main() {
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
	log.Info().Msg("Configuration received")
	/* Init config with default values */
	tExt := &ExtConfig{
		EndPoint:  defaultKubernetesClusterEndpoint,
		Ownership: transit.Yield,
		Views:     make(map[KubernetesView]map[string]transit.MetricDefinition),
		Groups:    []transit.ResourceGroup{},
	}
	tMonConn := &transit.MonitorConnection{Extensions: tExt}
	tMetProf := &transit.MetricsProfile{}

	err := connectors.UnmarshalConfig(data, tMetProf, tMonConn)
	if err != nil {
		log.Err(err).Msg("Could not parse config")
		return
	}

	var yamlData []byte
	switch tMonConn.Extensions.(*ExtConfig).AuthType {
	case ConfigFile:
		yamlData = []byte(tMonConn.Extensions.(*ExtConfig).KubernetesConfigFile)
	default:
		yamlData, err = yaml.Marshal(struct {
			Auth                   AuthType `yaml:"auth"`
			EndPoint               string   `yaml:"host"`
			KubernetesBearerToken  string   `yaml:"token,omitempty"`
			KubernetesUserName     string   `yaml:"username,omitempty"`
			KubernetesUserPassword string   `yaml:"password,omitempty"`
		}{
			tMonConn.Extensions.(*ExtConfig).AuthType,
			tMonConn.Extensions.(*ExtConfig).EndPoint,
			tMonConn.Extensions.(*ExtConfig).KubernetesUserName,
			tMonConn.Extensions.(*ExtConfig).KubernetesUserPassword,
			tMonConn.Extensions.(*ExtConfig).KubernetesBearerToken,
		})
		if err != nil {
			log.Err(err).Msg("Could not marshal struct to yaml")
		}
	}
	if err := writeDataToFile(yamlData); err != nil {
		log.Err(err).Msg("Could not write to file")
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
	chk, err := connectors.Hashsum(extConfig, tMetProf, tMonConn)
	if err != nil || !bytes.Equal(chksum, chk) {
		fresh = true
	}
	if err == nil {
		chksum = chk
	}

	if monitorConnection.ConnectorID != 0 {
		if err = connector.Initialize(*monitorConnection.Extensions.(*ExtConfig)); err != nil {
			connector.Shutdown()
			log.Err(err).Msg("Could not initialize connector")
		}
	} else {
		connector.Shutdown()
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
		log.Debug().Msgf("Collected %d:%d:%d", len(inventory), len(monitored), len(groups))

		if fresh {
			err := connectors.SendInventory(
				context.Background(),
				inventory,
				groups,
				extConfig.Ownership,
			)
			// TODO: better way to assure sync completion?
			log.Err(err).Msg("Sending inventory")
			time.Sleep(3 * time.Second)
		}
		err := connectors.SendMetrics(context.Background(), monitored, &groups)
		log.Err(err).Msg("Sending metrics")
	}
}

func buildNodeMetricsMap(metricsArray []transit.MetricDefinition) map[string]transit.MetricDefinition {
	metrics := make(map[string]transit.MetricDefinition)
	if metricsArray != nil {
		for _, metric := range metricsArray {
			if metric.ServiceType == string(ViewNodes) {
				metrics[metric.Name] = metric
			}
		}
	}

	// TODO: storage is not supported yet
	return metrics
}

func buildPodMetricsMap(metricsArray []transit.MetricDefinition) map[string]transit.MetricDefinition {
	metrics := make(map[string]transit.MetricDefinition)
	if metricsArray != nil {
		for _, metric := range metricsArray {
			if metric.ServiceType == string(ViewPods) {
				metrics[metric.Name] = metric
			}
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
