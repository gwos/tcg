package main

import (
	"github.com/gwos/tcg/config"
	"github.com/gwos/tcg/log"
	"github.com/gwos/tcg/transit"
)

const (
	defaultHostGroupName = "Servers"
	extensionsKeyGroups  = "groups"
	extensionsKeyName    = "name"
	extensionsKeyType    = "type"
)

type PrometheusConnectorConfig struct {
	Services       []string
	Groups         []transit.ResourceGroup
	MetricsProfile transit.MetricsProfile
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
	}

	// Update config with received values if presented
	if monitorConnection != nil && monitorConnection.Extensions != nil {
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
	}

	if metricsProfile != nil {
		connectorConfig.MetricsProfile = *metricsProfile
	}

	if len(gwConnections) > 0 {
		connectorConfig.Ownership = transit.HostOwnershipType(gwConnections[0].DeferOwnership)
	}

	return &connectorConfig
}
