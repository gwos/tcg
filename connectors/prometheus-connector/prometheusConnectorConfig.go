package main

import (
	"github.com/gwos/tcg/config"
	"github.com/gwos/tcg/connectors"
	"github.com/gwos/tcg/log"
	"github.com/gwos/tcg/transit"
	"time"
)

const (
	defaultHostGroupName   = "Servers"
	extensionsKeyGroups    = "groups"
	extensionsKeyName      = "name"
	extensionsKeyType      = "type"
	extensionsKeyResources = "resources"
	extensionsKeyUrl       = "url"
	extensionsKeyHeaders   = "headers"
)

type PrometheusConnectorConfig struct {
	Services       []string
	Groups         []transit.ResourceGroup
	Resources      []Resource
	MetricsProfile transit.MetricsProfile
	Timer          time.Duration
	Ownership      transit.HostOwnershipType
}

func InitConfig(monitorConnection *transit.MonitorConnection, metricsProfile *transit.MetricsProfile,
	gwConnections config.GWConnections) *PrometheusConnectorConfig {

	// Init config with default values
	connectorConfig := PrometheusConnectorConfig{
		Services: []string{},
		Groups: []transit.ResourceGroup{{
			GroupName: defaultHostGroupName,
			Type:      transit.HostGroup,
		}},
		MetricsProfile: transit.MetricsProfile{},
		Ownership:      transit.Yield,
		Timer:          connectors.DefaultTimer,
	}

	// Update config with received values if presented
	if monitorConnection != nil && monitorConnection.Extensions != nil {
		if value, present := monitorConnection.Extensions[connectors.ExtensionsKeyTimer]; present {
			connectorConfig.Timer = time.Duration(int64(value.(float64))) * time.Minute
			connectors.Timer = connectorConfig.Timer
		}

		if value, present := monitorConnection.Extensions[extensionsKeyGroups]; present {
			connectorConfig.Groups = []transit.ResourceGroup{}
			groups := value.([]interface{})
			for _, group := range groups {
				groupMap := group.(map[string]interface{})
				if _, present := groupMap[extensionsKeyName]; !present {
					log.Warn("[Prometheus Connector Config]: Group name required.")
					continue
				}
				if _, present := groupMap[extensionsKeyType]; !present {
					log.Warn("[Prometheus Connector Config]: Group type required.")
					continue
				}
				connectorConfig.Groups = append(connectorConfig.Groups,
					transit.ResourceGroup{
						GroupName: groupMap[extensionsKeyName].(string),
						Type:      transit.GroupType(groupMap[extensionsKeyType].(string)),
					},
				)
			}
		}
		if value, present := monitorConnection.Extensions[extensionsKeyResources]; present {
			connectorConfig.Resources = []Resource{}
			resources := value.([]interface{})
			for _, resource := range resources {
				resourceMap := resource.(map[string]interface{})
				if _, present := resourceMap[extensionsKeyUrl]; !present {
					log.Warn("[Prometheus Connector Config]: Resource url required.")
					continue
				}
				if _, present := resourceMap[extensionsKeyHeaders]; !present {
					log.Warn("[Prometheus Connector Config]: Resource headers required.")
					continue
				}
				headers := make(map[string]string)
				for _, header := range resourceMap[extensionsKeyHeaders].([]interface{}) {
					headers[header.(map[string]interface{})["key"].(string)] =
						header.(map[string]interface{})["value"].(string)
				}
				connectorConfig.Resources = append(connectorConfig.Resources,
					Resource{
						url:     resourceMap[extensionsKeyUrl].(string),
						headers: headers,
					},
				)
			}
		}
	}

	if metricsProfile != nil {
		connectorConfig.MetricsProfile = *metricsProfile
	}

	if len(gwConnections) > 0 {
		connectorConfig.Ownership = transit.HostOwnershipType(gwConnections[0].DeferOwnership)
	}

	return &connectorConfig
}

type Resource struct {
	url     string
	headers map[string]string
}
