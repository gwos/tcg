package main

import (
	"github.com/gwos/tcg/config"
	"github.com/gwos/tcg/connectors"
	"github.com/gwos/tcg/log"
	"github.com/gwos/tcg/transit"
)

// keys for extensions
const (
	defaultHostGroupName = "Servers"

	extensionsKeyProcesses = "processes"
	extensionsKeyGroups    = "groups"
	extensionsKeyName      = "name"
	extensionsKeyType      = "type"
)

type ServerConnectorConfig struct {
	Processes      []string
	Groups         []transit.ResourceGroup
	MetricsProfile transit.MetricsProfile
	Timer          int64
	Ownership      transit.HostOwnershipType
}

func InitConfig(monitorConnection *transit.MonitorConnection, metricsProfile *transit.MetricsProfile,
	gwConnections config.GWConnections) *ServerConnectorConfig {

	// init config with default values
	connectorConfig := ServerConnectorConfig{
		Processes: []string{},
		Groups: []transit.ResourceGroup{{
			GroupName: defaultHostGroupName,
			Type:      transit.HostGroup,
		}},
		MetricsProfile: transit.MetricsProfile{},
		Timer:          connectors.DefaultTimer,
		Ownership:      transit.Yield,
	}

	// update config with received values if presented
	if monitorConnection != nil && monitorConnection.Extensions != nil {
		if value, present := monitorConnection.Extensions[connectors.ExtensionsKeyTimer]; present {
			if value.(float64) >= 1 {
				connectorConfig.Timer = int64(value.(float64) * 60)
			}
		}

		if value, present := monitorConnection.Extensions[extensionsKeyProcesses]; present {
			processesInterface := value.([]interface{})
			for _, process := range processesInterface {
				connectorConfig.Processes = append(connectorConfig.Processes, process.(string))
			}
		}

		if value, present := monitorConnection.Extensions[extensionsKeyGroups]; present {
			connectorConfig.Groups = []transit.ResourceGroup{}
			groups := value.([]interface{})
			for _, group := range groups {
				groupMap := group.(map[string]interface{})
				if _, present := groupMap[extensionsKeyName]; !present {
					log.Warn("[Server Connector Config]: Group name required.")
					continue
				}
				if _, present := groupMap[extensionsKeyType]; !present {
					log.Warn("[Server Connector Config]: Group type required.")
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
