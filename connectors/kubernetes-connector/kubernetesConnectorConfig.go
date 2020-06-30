package main

import (
	"github.com/gwos/tcg/config"
	"github.com/gwos/tcg/transit"
)

// TODO: push down to Connectors
const (
	defaultHostGroupName = "Servers"
	extensionsKeyGroups    = "groups"
	extensionsKeyName      = "name"
	extensionsKeyType      = "type"
)

type KubernetesConnectorConfig struct {
	EndPoint 	   string
	Ownership      transit.HostOwnershipType
	Views          map[KubernetesView]map[string]transit.MetricDefinition
	Groups         []transit.ResourceGroup
	// TODO: complete implementation
}

func InitConfig(monitorConnection *transit.MonitorConnection, metricsProfile *transit.MetricsProfile,
	gwConnections config.GWConnections) *KubernetesConnectorConfig {

	// Init config with default values
	connectorConfig := KubernetesConnectorConfig{
		EndPoint: monitorConnection.Server,
		Ownership:      transit.Yield,
	}

	// Update config with received values if presented
	if monitorConnection != nil && monitorConnection.Extensions != nil {
		if _, present := monitorConnection.Extensions[extensionsKeyGroups]; present {
			// TODO: complete implementation
		}
	}

	if metricsProfile != nil {
		for _, metric := range metricsProfile.Metrics {
			// temporary solution, will be removed
			// TODO: push down into connectors - metric.Monitored breaks synthetics
			//if templateMetricName == metric.Name || !metric.Monitored {
			//	continue
			//}
			if metrics, has := connectorConfig.Views[KubernetesView(metric.ServiceType)]; has {
				metrics[metric.Name] = metric
				connectorConfig.Views[KubernetesView(metric.ServiceType)] = metrics
			} else {
				metrics := make(map[string]transit.MetricDefinition)
				metrics[metric.Name] = metric
				connectorConfig.Views[KubernetesView(metric.ServiceType)] = metrics
			}
		}
	}

	if len(gwConnections) > 0 {
		connectorConfig.Ownership = transit.HostOwnershipType(gwConnections[0].DeferOwnership)
	}

	return &connectorConfig
}
