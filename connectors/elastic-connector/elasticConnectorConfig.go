package main

import (
	"github.com/gwos/tcg/config"
	"github.com/gwos/tcg/connectors"
	"github.com/gwos/tcg/connectors/elastic-connector/clients"
	"github.com/gwos/tcg/transit"
	"strconv"
	"strings"
)

const (
	// default config values
	defaultProtocol                 = "http"
	defaultElasticServer            = "http://localhost:9200"
	defaultKibanaServerName         = "http://localhost:5601"
	defaultKibanaUsername           = ""
	defaultKibanaPassword           = ""
	defaultTimeFilterFrom           = "now-$interval"
	defaultTimeFilterTo             = "now"
	defaultAlwaysOverrideTimeFilter = false
	defaultHostNameLabel            = "container.name.keyword"
	defaultHostGroupLabel           = "container.labels.com_docker_compose_project.keyword"
	defaultHostGroupName            = "Elastic Search"
	defaultGroupNameByUser          = false

	// keys for extensions
	extensionsKeyKibana             = "kibana"
	extensionsKeyServerName         = "serverName"
	extensionsKeyUsername           = "userName"
	extensionsKeyPassword           = "password"
	extensionsKeyTimeFilter         = "timefilter"
	extensionsKeyFrom               = "from"
	extensionsKeyTo                 = "to"
	extensionsKeyOverride           = "override"
	extensionsKeyHostNameLabelPath  = "hostNameLabelPath"
	extensionsKeyHostGroupLabelPath = "hostGroupLabelPath"
	extensionsKeyGroupNameByUser    = "hostGroupNameByUser"

	intervalTemplate      = "$interval"
	intervalPeriodSeconds = "s"

	// temporary solution, will be removed
	templateMetricName = "$view_Template#"
)

type ElasticConnectorConfig struct {
	AppType            string
	AgentId            string
	Servers            []string
	Kibana             Kibana
	Views              map[string]map[string]transit.MetricDefinition
	CustomTimeFilter   clients.TimeFilter
	OverrideTimeFilter bool
	HostNameField      string
	HostGroupField     string
	GroupNameByUser    bool
	Timer              int64
	Ownership          transit.HostOwnershipType
	GWConnections      config.GWConnections
}

type Kibana struct {
	ServerName string
	Username   string
	Password   string
}

// Builds elastic connector configuration based on monitor connection settings and default values
func InitConfig(appType string, agentId string, monitorConnection *transit.MonitorConnection, metricsProfile *transit.MetricsProfile,
	gwConnections config.GWConnections) *ElasticConnectorConfig {

	// init config with default values
	connectorConfig := ElasticConnectorConfig{
		AppType: appType,
		AgentId: agentId,
		Servers: []string{defaultElasticServer},
		Kibana: Kibana{
			ServerName: defaultKibanaServerName,
			Username:   defaultKibanaUsername,
			Password:   defaultKibanaPassword,
		},
		Views: make(map[string]map[string]transit.MetricDefinition),
		CustomTimeFilter: clients.TimeFilter{
			From: defaultTimeFilterFrom,
			To:   defaultTimeFilterTo,
		},
		OverrideTimeFilter: defaultAlwaysOverrideTimeFilter,
		HostNameField:      defaultHostNameLabel,
		HostGroupField:     defaultHostGroupLabel,
		GroupNameByUser:    defaultGroupNameByUser,
		Timer:              connectors.DefaultTimer,
		Ownership:          transit.Yield,
		GWConnections:      gwConnections,
	}

	// update config with received values if presented
	if monitorConnection != nil {
		// Servers
		if monitorConnection.Server != "" {
			servers := strings.Split(monitorConnection.Server, ",")
			for i, server := range servers {
				if !strings.HasPrefix(server, defaultProtocol) {
					servers[i] = defaultProtocol + ":" + "//" + server
				}
			}
			connectorConfig.Servers = servers
		}

		if monitorConnection.Extensions != nil {
			// Kibana
			if monitorConnection.Extensions[extensionsKeyKibana] != nil {
				kibana := monitorConnection.Extensions[extensionsKeyKibana].(map[string]interface{})
				if kibana[extensionsKeyServerName] != nil {
					kibanaServer := kibana[extensionsKeyServerName].(string)
					if !strings.HasSuffix(kibanaServer, "/") {
						kibanaServer = kibanaServer + "/"
					}
					if !strings.HasPrefix(kibanaServer, defaultProtocol) {
						kibanaServer = defaultProtocol + ":" + "//" + kibanaServer
					}
					connectorConfig.Kibana.ServerName = kibanaServer
				}
				if kibana[extensionsKeyUsername] != nil {
					connectorConfig.Kibana.Username = kibana[extensionsKeyUsername].(string)
				}
				if kibana[extensionsKeyPassword] != nil {
					connectorConfig.Kibana.Password = kibana[extensionsKeyPassword].(string)
				}
			}

			// Time filter
			if monitorConnection.Extensions[extensionsKeyTimeFilter] != nil {
				timeFilterValue := monitorConnection.Extensions[extensionsKeyTimeFilter].(map[string]interface{})
				if timeFilterValue[extensionsKeyFrom] != nil {
					connectorConfig.CustomTimeFilter.From = timeFilterValue[extensionsKeyFrom].(string)
				}
				if timeFilterValue[extensionsKeyTo] != nil {
					connectorConfig.CustomTimeFilter.To = timeFilterValue[extensionsKeyTo].(string)
				}
				if timeFilterValue[extensionsKeyOverride] != nil {
					connectorConfig.OverrideTimeFilter = timeFilterValue[extensionsKeyOverride].(bool)
				}
			}

			// host name labels
			if monitorConnection.Extensions[extensionsKeyHostNameLabelPath] != nil {
				connectorConfig.HostNameField = monitorConnection.Extensions[extensionsKeyHostNameLabelPath].(string)
			}

			// host group name by user
			if monitorConnection.Extensions[extensionsKeyGroupNameByUser] != nil {
				connectorConfig.GroupNameByUser = monitorConnection.Extensions[extensionsKeyGroupNameByUser].(bool)
			}

			// host group labels
			// first update default if group name by user was changed
			if connectorConfig.GroupNameByUser {
				connectorConfig.HostGroupField = defaultHostGroupName
			} else {
				connectorConfig.HostGroupField = defaultHostGroupLabel
			}
			// update with user's value if specified
			if monitorConnection.Extensions[extensionsKeyHostGroupLabelPath] != nil {
				connectorConfig.HostGroupField = monitorConnection.Extensions[extensionsKeyHostGroupLabelPath].(string)
			}

			// Timer
			if monitorConnection.Extensions[connectors.ExtensionsKeyTimer] != nil {
				connectorConfig.Timer = int64(monitorConnection.Extensions[connectors.ExtensionsKeyTimer].(float64) * 60)
			}
		}
	}

	if metricsProfile != nil {
		// Views
		for _, metric := range metricsProfile.Metrics {
			// temporary solution, will be removed
			if templateMetricName == metric.Name || !metric.Monitored {
				continue
			}
			if connectorConfig.Views[metric.ServiceType] != nil {
				connectorConfig.Views[metric.ServiceType][metric.Name] = metric
				connectorConfig.Views[metric.ServiceType] = connectorConfig.Views[metric.ServiceType]
			} else {
				metrics := make(map[string]transit.MetricDefinition)
				metrics[metric.Name] = metric
				connectorConfig.Views[metric.ServiceType] = metrics
			}
		}
	}

	if len(gwConnections) > 0 {
		connectorConfig.Ownership = transit.HostOwnershipType(gwConnections[0].DeferOwnership)
	}

	// replace '$interval' with actual value in time filter
	connectorConfig.replaceIntervalTemplates()

	return &connectorConfig
}

func (connectorConfig *ElasticConnectorConfig) replaceIntervalTemplates() {
	connectorConfig.CustomTimeFilter.From = replaceIntervalTemplate(connectorConfig.CustomTimeFilter.From,
		connectorConfig.Timer)
	connectorConfig.CustomTimeFilter.To = replaceIntervalTemplate(connectorConfig.CustomTimeFilter.To,
		connectorConfig.Timer)
}

func replaceIntervalTemplate(templateString string, intervalValue int64) string {
	interval := strconv.Itoa(int(intervalValue)) + intervalPeriodSeconds
	if strings.Contains(templateString, intervalTemplate) {
		templateString = strings.ReplaceAll(templateString, intervalTemplate, interval)
	}
	return templateString
}
