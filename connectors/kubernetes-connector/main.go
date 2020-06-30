package main

import (
	"bytes"
	"fmt"
	"github.com/gwos/tcg/config"
	"github.com/gwos/tcg/connectors"
	"github.com/gwos/tcg/log"
	"github.com/gwos/tcg/services"
	"github.com/gwos/tcg/transit"
)

// Variables to control connector version and build time.
// Can be overridden during the build step.
// See README for details.
var (
	buildTime = "Build time not provided"
	buildTag  = "8.1.1"
)

func main() {
	connectors.ControlCHandler()

	var transitService = services.GetTransitService()
	var cfg KubernetesConnectorConfig
	var chksum []byte
	var connector KubernetesConnector

	log.Info(fmt.Sprintf("[Kubernetes Connector]: Version: %s   /   Build time: %s", buildTag, buildTime))

	transitService.ConfigHandler = func(data []byte) {
		log.Info("[Kubernetes Connector]: Configuration received")
		if monitorConn, profile, gwConnections, err := connectors.RetrieveCommonConnectorInfo(data); err == nil {
			c := InitConfig(monitorConn, profile, gwConnections)
			cfg = *c
			chk, err := connectors.Hashsum(
				config.GetConfig().Connector.AgentID,
				config.GetConfig().GWConnections,
				cfg,
			)
			if err != nil || !bytes.Equal(chksum, chk) {
				log.Info("[Kubernetes Connector]: Sending inventory ...")
				_ = connectors.SendInventory(
					//*GatherInventory(),
					nil, // TODO:
					cfg.Groups,
					cfg.Ownership,
				)
			}
			if err == nil {
				chksum = chk
			}
		} else {
			log.Error("[Kubernetes Connector]: Error during parsing config. Aborting ...")
			return
		}
	}

	log.Info("[Kubernetes Connector]: Waiting for configuration to be delivered ...")
	if err := transitService.DemandConfig(initializeEntrypoints()...); err != nil {
		log.Error(err)
		return
	}

	if err := connectors.Start(); err != nil {
		log.Error(err)
		return
	}

	log.Info("[Kubernetes1 Connector]: Starting metric connection ...")
	// TODO: this if block is temporary code - should be removed as impl matures
	if connector.kapi == nil {
		cfg = KubernetesConnectorConfig{
			EndPoint:  "gwos.bluesunrise.com:8001",
			Ownership: transit.Yield,
			Views:     nil,
			Groups:    nil,
		}
		err := connector.Initialize(cfg)
		if err != nil {
			panic(err)
		}
	}
	// fudge up some metrics
	cfg.Views = make(map[KubernetesView	]map[string]transit.MetricDefinition)
	cfg.Views[ViewNodes] = fudgeUpNodeMetricDefinitions()
	cfg.Views[ViewPods] =  make(map[string]transit.MetricDefinition) // TODO: pods snext
	inventory, monitored, groups := connector.Collect(&cfg)
	fmt.Println(len(inventory), len(monitored), len(groups))
	// TODO: connectors.SendInventory
}

// initializeEntrypoints - function for setting entrypoints,
// that will be available through the Server Connector API
func initializeEntrypoints() []services.Entrypoint {
	return make([]services.Entrypoint, 1)
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