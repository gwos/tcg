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
			if kibana, has := monitorConnection.Extensions[extensionsKeyKibana]; has {
				if serverName, has := kibana.(map[string]interface{})[extensionsKeyServerName]; has {
					kibanaServer := serverName.(string)
					if !strings.HasSuffix(kibanaServer, "/") {
						kibanaServer = kibanaServer + "/"
					}
					if !strings.HasPrefix(kibanaServer, defaultProtocol) {
						kibanaServer = defaultProtocol + ":" + "//" + kibanaServer
					}
					connectorConfig.Kibana.ServerName = kibanaServer
				}
				if username, has := kibana.(map[string]interface{})[extensionsKeyUsername]; has {
					connectorConfig.Kibana.Username = username.(string)
				}
				if password, has := kibana.(map[string]interface{})[extensionsKeyPassword]; has {
					connectorConfig.Kibana.Password = password.(string)
				}
			}

			// Time filter
			if timeFilterValue, has := monitorConnection.Extensions[extensionsKeyTimeFilter]; has {
				if value, has := timeFilterValue.(map[string]interface{})[extensionsKeyFrom]; has {
					connectorConfig.CustomTimeFilter.From = value.(string)
				}
				if value, has := timeFilterValue.(map[string]interface{})[extensionsKeyTo]; has {
					connectorConfig.CustomTimeFilter.To = value.(string)
				}
				if value, has := timeFilterValue.(map[string]interface{})[extensionsKeyOverride]; has {
					connectorConfig.OverrideTimeFilter = value.(bool)
				}
			}

			// host name labels
			if value, has := monitorConnection.Extensions[extensionsKeyHostNameLabelPath]; has {
				connectorConfig.HostNameField = value.(string)
			}

			// host group labels
			if value, has := monitorConnection.Extensions[extensionsKeyHostGroupLabelPath]; has {
				connectorConfig.HostGroupField = value.(string)
			}

			// Timer
			if value, has := monitorConnection.Extensions[connectors.ExtensionsKeyTimer]; has {
				connectorConfig.Timer = int64(value.(float64) * 60)
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
			if metrics, has := connectorConfig.Views[metric.ServiceType]; has {
				metrics[metric.Name] = metric
				connectorConfig.Views[metric.ServiceType] = metrics
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
