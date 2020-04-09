package model

import (
	"github.com/gwos/tng/connectors"
	"github.com/gwos/tng/transit"
	"strconv"
	"strings"
)

// default extensions values
const (
	defaultProtocol = "http"

	defaultElasticServer = "http://localhost:9200"
	defaultKibanaServer  = "http://localhost:5601"

	defaultTimeFilterFrom           = "now-$interval"
	defaultTimeFilterTo             = "now"
	defaultAlwaysOverrideTimeFilter = false

	defaultHostNameLabel  = "container.name"
	defaultHostGroupLabel = "container.labels.com_docker_compose_project"

	// TODO move somewhere more global
	DefaultTimer = 120
)

// keys for extensions
const (
	exKeyKibanaServer       = "kibanaServer"
	exKeyTimeFilter         = "timefilter"
	exKeyFrom               = "from"
	exKeyTo                 = "to"
	exKeyOverride           = "override"
	exKeyHostNameLabelPath  = "hostNameLabelPath"
	exKeyHostGroupLabelPath = "hostGroupLabelPath"
	exKeyTimer              = "timer"
)

const (
	templateMetricName    = "$view_Template#"
	intervalTemplate      = "$interval"
	intervalPeriodSeconds = "s"
)

type ElasticConnectorConfig struct {
	Servers            []string
	KibanaServer       string
	Views              map[string]map[string]transit.MetricDefinition
	CustomTimeFilter   TimeFilter
	OverrideTimeFilter bool
	HostNameLabelPath  []string
	HostGroupLabelPath []string
	Timer              float64
	Ownership          transit.HostOwnershipType
}

// Builds elastic connector configuration based on monitor connection settings and default values
func InitConfig(connection *transit.MonitorConnection, profile *transit.MetricsProfile, ownership transit.HostOwnershipType) *ElasticConnectorConfig {
	var config ElasticConnectorConfig

	// Servers
	server := connection.Server
	if server == "" {
		server = defaultElasticServer
	}
	servers := strings.Split(server, ",")
	for i, server := range servers {
		if !strings.HasPrefix(server, defaultProtocol) {
			servers[i] = defaultProtocol + ":" + "//" + server
		}
	}
	config.Servers = servers

	// Kibana
	kibanaServer := defaultKibanaServer
	if value, has := connection.Extensions[exKeyKibanaServer]; has {
		kibanaServer = value.(string)
	}
	if !strings.HasSuffix(kibanaServer, "/") {
		kibanaServer = kibanaServer + "/"
	}
	if !strings.HasPrefix(kibanaServer, defaultProtocol) {
		kibanaServer = defaultProtocol + ":" + "//" + kibanaServer
	}
	config.KibanaServer = kibanaServer

	// Views
	config.Views = make(map[string]map[string]transit.MetricDefinition)
	for _, metric := range profile.Metrics {
		if templateMetricName == metric.Name {
			continue
		}
		if metrics, has := config.Views[metric.ServiceType]; has {
			metrics[metric.Name] = metric
			config.Views[metric.ServiceType] = metrics
		} else {
			metrics := make(map[string]transit.MetricDefinition)
			metrics[metric.Name] = metric
			config.Views[metric.ServiceType] = metrics
		}
	}

	// Time filter
	if timeFilterValue, has := connection.Extensions[exKeyTimeFilter]; has {
		from := defaultTimeFilterFrom
		to := defaultTimeFilterTo
		if value, has := timeFilterValue.(map[string]interface{})[exKeyFrom]; has {
			from = value.(string)
		}
		if value, has := timeFilterValue.(map[string]interface{})[exKeyTo]; has {
			to = value.(string)
		}
		customTimeFilter := TimeFilter{
			From: from,
			To:   to,
		}
		config.CustomTimeFilter = customTimeFilter
		if value, has := timeFilterValue.(map[string]interface{})[exKeyOverride]; has {
			config.OverrideTimeFilter = value.(bool)
		} else {
			config.OverrideTimeFilter = defaultAlwaysOverrideTimeFilter
		}
	} else {
		defaultTimeFilter := TimeFilter{
			From: defaultTimeFilterFrom,
			To:   defaultTimeFilterTo,
		}
		config.CustomTimeFilter = defaultTimeFilter
		config.OverrideTimeFilter = defaultAlwaysOverrideTimeFilter
	}
	config.CustomTimeFilter.replaceIntervalPattern()

	// host name and host group labels
	var hostNameLabels, hostGroupLabels string
	if value, has := connection.Extensions[exKeyHostNameLabelPath]; has {
		hostNameLabels = value.(string)
	} else {
		hostNameLabels = defaultHostNameLabel
	}
	config.HostNameLabelPath = strings.Split(hostNameLabels, ".")

	if value, has := connection.Extensions[exKeyHostGroupLabelPath]; has {
		hostGroupLabels = value.(string)
	} else {
		hostGroupLabels = defaultHostGroupLabel
	}
	config.HostGroupLabelPath = strings.Split(hostGroupLabels, ".")

	// Timer
	config.Timer = float64(DefaultTimer)
	if value, has := connection.Extensions[exKeyTimer]; has {
		config.Timer = value.(float64)
	}

	// Ownership
	config.Ownership = ownership

	return &config
}

func (timeFilter *TimeFilter) replaceIntervalPattern() {
	timeFilter.From = replaceIntervalPattern(timeFilter.From)
	timeFilter.To = replaceIntervalPattern(timeFilter.To)
}

func replaceIntervalPattern(timeValue string) string {
	interval := strconv.Itoa(connectors.Timer) + intervalPeriodSeconds
	if strings.Contains(timeValue, intervalTemplate) {
		timeValue = strings.ReplaceAll(timeValue, intervalTemplate, interval)
	}
	return timeValue
}
