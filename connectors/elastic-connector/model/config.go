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

	defaultElasticServer    = "http://localhost:9200"
	defaultKibanaServerName = "http://localhost:5601"
	defaultKibanaUsername   = ""
	defaultKibanaPassword   = ""

	defaultTimeFilterFrom           = "now-$interval"
	defaultTimeFilterTo             = "now"
	defaultAlwaysOverrideTimeFilter = false

	defaultHostNamePrefix = "gw8-"
	defaultHostNameLabel  = "container.name"
	defaultHostGroupLabel = "container.labels.com_docker_compose_project"

	// TODO move somewhere more global
	DefaultInterval = 300
	DefaultTimeout  = 600
)

// keys for extensions
const (
	exKeyKibana             = "kibana"
	exKeyServerName         = "serverName"
	exKeyUsername           = "userName"
	exKeyPassword           = "password"
	exKeyTimeFilter         = "timefilter"
	exKeyFrom               = "from"
	exKeyTo                 = "to"
	exKeyOverride           = "override"
	exKeyHostamePrefix      = "hostNamePrefix"
	exKeyHostNameLabelPath  = "hostNameLabelPath"
	exKeyHostGroupLabelPath = "hostGroupLabelPath"
	exKeyInterval           = "checkIntervalMinutes"
	exKeyTimeout            = "checkTimeoutSeconds"
)

const (
	templateMetricName    = "$view_Template#"
	intervalTemplate      = "$interval"
	intervalPeriodSeconds = "s"
)

type Kibana struct {
	ServerName string
	Username   string
	Password   string
}

type ElasticConnectorConfig struct {
	Servers            []string
	Kibana             Kibana
	Views              map[string]map[string]transit.MetricDefinition
	CustomTimeFilter   TimeFilter
	OverrideTimeFilter bool
	HostNamePrefix     string
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
	kibanaServerName := defaultKibanaServerName
	kibanaUsername := defaultKibanaUsername
	kibanaPassword := defaultKibanaPassword
	if kibana, has := connection.Extensions[exKeyKibana]; has {
		if serverName, has := kibana.(map[string]interface{})[exKeyServerName]; has {
			kibanaServerName = serverName.(string)
		}
		if username, has := kibana.(map[string]interface{})[exKeyUsername]; has {
			kibanaUsername = username.(string)
		}
		if password, has := kibana.(map[string]interface{})[exKeyPassword]; has {
			kibanaPassword = password.(string)
		}
	}
	if !strings.HasSuffix(kibanaServerName, "/") {
		kibanaServerName = kibanaServerName + "/"
	}
	if !strings.HasPrefix(kibanaServerName, defaultProtocol) {
		kibanaServerName = defaultProtocol + ":" + "//" + kibanaServerName
	}
	config.Kibana = Kibana{
		ServerName: kibanaServerName,
		Username:   kibanaUsername,
		Password:   kibanaPassword,
	}

	// Views
	config.Views = make(map[string]map[string]transit.MetricDefinition)
	for _, metric := range profile.Metrics {
		if templateMetricName == metric.Name || !metric.Monitored {
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

	// host name prefix
	hostNamePrefix := defaultHostNamePrefix
	if value, has := connection.Extensions[exKeyHostamePrefix]; has {
		hostNamePrefix = value.(string)
	}
	config.HostNamePrefix = hostNamePrefix

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
	config.Timer = float64(DefaultInterval)
	if value, has := connection.Extensions[exKeyInterval]; has {
		config.Timer = value.(float64) * 60
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
