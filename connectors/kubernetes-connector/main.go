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
	"io/ioutil"
	"strings"
	"time"
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

	log.Info("[K8 Connector]: Waiting for configuration to be delivered ...")
	if err := transitService.DemandConfig(); err != nil {
		log.Error("[K8 Connector]: ", err)
		return
	}

	if err := connectors.Start(); err != nil {
		log.Error(err)
		return
	}

	log.Info("[K8 Connector]: Starting metric connection ...")
	connectors.StartPeriodic(ctxCancel, connectors.CheckInterval, periodicHandler)

	/* prevent return */
	<-make(chan bool, 1)
}

func configHandler(data []byte) {
	log.Info("[K8 Connector]: Configuration received")

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
		log.Error("[K8 Connector]: Error during parsing config.", err.Error())
		return
	}

	if tMonConn.Extensions.(*ExtConfig).AuthType == ConfigFile {
		if err := writeDataToFile([]byte(tMonConn.Extensions.(*ExtConfig).KubernetesConfigFile)); err != nil {
			log.Error("[K8 Connector]: Error writing to file, reason: " + err.Error())
		}
	}

	/* Update config with received values */
	// TODO: fudge up some metrics - remove this once we hook in live metrics, appType
	tExt.Views[ViewNodes] = fudgeUpNodeMetricDefinitions()
	tExt.Views[ViewPods] = fudgeUpPodMetricDefinitions()
	for _, metric := range tMetProf.Metrics {
		// temporary solution, will be removed
		// TODO: push down into connectors - metric.Monitored breaks synthetics
		// if templateMetricName == metric.Name || !metric.Monitored {
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

	/* Restart periodic loop */
	cancel()
	ctxCancel, cancel = context.WithCancel(context.Background())
	services.GetTransitService().RegisterExitHandler(cancel)
	connectors.StartPeriodic(ctxCancel, connectors.CheckInterval, periodicHandler)
}

func periodicHandler() {
	if connector.kapi != nil {
		inventory, monitored, groups := connector.Collect(extConfig)
		log.Debug("[K8 Connector]: ", fmt.Sprintf("%d:%d:%d", len(inventory), len(monitored), len(groups)))

		if count == 0 {
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
		time.Sleep(3 * time.Second) // TODO: better way to assure sync completion?
		if err := connectors.SendMetrics(context.Background(), monitored, &groups); err != nil {
			log.Error("[K8 Connector]: Error during sending metrics.", err)
		}
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
