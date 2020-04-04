package main

import (
	"github.com/gwos/tng/connectors"
	"github.com/gwos/tng/log"
	"github.com/gwos/tng/milliseconds"
	_ "github.com/gwos/tng/milliseconds"
	"github.com/gwos/tng/transit"
	"strconv"
	"strings"
	"time"
)

// keys for extensions
const (
	ekKibanaEndpoint     = "kibanaEndpoint"
	ekTimeFilter         = "timefilter"
	ekTimeFilterFrom     = "from"
	ekTimeFilterTo       = "to"
	ekTimeFilterOverride = "override"
	ekHostNameLabelPath  = "hostNameLabelPath"
	ekHostGroupLabelPath = "hostGroupLabelPath"
)

// default extensions values
const (
	defaultServer         = "http://localhost:9200"
	defaultKibanaEndpoint = "http://localhost:5601/kibana/api/"

	defaultTimeFilterFrom           = "now-$interval"
	defaultTimeFilterTo             = "now"
	defaultAlwaysOverrideTimeFilter = true

	defaultHostNameLabel  = "container.name"
	defaultHostGroupLabel = "container.labels.com_docker_compose_project"
)

const (
	intervalTemplate      = "$interval"
	intervalPeriodSeconds = "s"
)

type ElasticView string

const (
	StoredQueries  ElasticView = "storedQueries"
	StoredSearches             = "storedSearches"
	KQL                        = "kql"
	SelfMonitoring             = "selfMonitoring"
)

func CollectMetrics(connection transit.MonitorConnection, profile transit.MetricsProfile) ([]transit.MonitoredResource,
	[]transit.InventoryResource, []transit.ResourceGroup) {
	views := connection.Views
	if views == nil || len(views) == 0 {
		log.Info("No views found.")
		return nil, nil, nil
	}

	config := buildElasticConnectorConfig(connection)

	hosts, groups := make(map[string]tempHost), make(map[string]map[string]struct{})
	for _, view := range views {
		if view.Enabled {
			switch view.Name {
			case string(StoredQueries):
				hosts, groups = collectStoredQueriesMetrics(hosts, groups, profile, config)
				break
			default:
				log.Warn("Not supported view: ", view.Name)
				break
			}
		}
	}

	monitoredResources, inventoryResources := convertHosts(hosts)
	resourceGroups := convertGroups(groups)
	return monitoredResources, inventoryResources, resourceGroups
}

// Builds elastic connector configuration based on monitor connection settings and default values
func buildElasticConnectorConfig(connection transit.MonitorConnection) elasticConnectorConfig {
	var config elasticConnectorConfig

	// servers
	servers := connection.Server
	if servers == "" {
		servers = defaultServer
	}
	config.servers = strings.Split(servers, ",")

	// kibana
	kibanaApiEndpoint := defaultKibanaEndpoint
	if connection.Extensions[ekKibanaEndpoint] != nil {
		kibanaApiEndpoint = connection.Extensions[ekKibanaEndpoint].(string)
	}
	config.kibanaApiEndpoint = kibanaApiEndpoint

	// time filter
	if connection.Extensions[ekTimeFilter] == nil {
		defaultTimeFilter := TimeFilter{
			From: defaultTimeFilterFrom,
			To:   defaultTimeFilterTo,
		}
		config.timeFilter = defaultTimeFilter
		config.alwaysOverrideTimeFilter = defaultAlwaysOverrideTimeFilter
	} else {
		from := defaultTimeFilterFrom
		to := defaultTimeFilterTo
		if connection.Extensions[ekTimeFilter].(map[string]interface{})[ekTimeFilterFrom] != nil {
			from = connection.Extensions[ekTimeFilter].(map[string]interface{})[ekTimeFilterFrom].(string)
		}
		if connection.Extensions[ekTimeFilter].(map[string]interface{})[ekTimeFilterTo] != nil {
			to = connection.Extensions[ekTimeFilter].(map[string]interface{})[ekTimeFilterTo].(string)
		}
		from = convertIntervalTemplate(from)
		to = convertIntervalTemplate(to)
		customTimeFilter := TimeFilter{
			From: from,
			To:   to,
		}
		config.timeFilter = customTimeFilter
		if connection.Extensions[ekTimeFilter].(map[string]interface{})[ekTimeFilterOverride] != nil {
			config.alwaysOverrideTimeFilter = connection.Extensions[ekTimeFilter].(map[string]interface{})[ekTimeFilterOverride].(bool)
		} else {
			config.alwaysOverrideTimeFilter = defaultAlwaysOverrideTimeFilter
		}
	}

	// host name and host group labels
	var hostNameLabels, hostGroupLabels string
	if connection.Extensions[ekHostNameLabelPath] == nil {
		hostNameLabels = connection.Extensions[ekHostNameLabelPath].(string)
	} else {
		hostNameLabels = defaultHostNameLabel
	}
	if connection.Extensions[ekHostGroupLabelPath] == nil {
		hostGroupLabels = connection.Extensions[ekHostGroupLabelPath].(string)
	} else {
		hostGroupLabels = defaultHostGroupLabel
	}
	config.hostNameLabelPath = strings.Split(hostNameLabels, ".")
	config.hostGroupLabelPath = strings.Split(hostGroupLabels, ".")

	return config
}

func collectStoredQueriesMetrics(hosts map[string]tempHost, groups map[string]map[string]struct{}, profile transit.MetricsProfile,
	config elasticConnectorConfig) (map[string]tempHost, map[string]map[string]struct{}) {
	monitoredStoredQueriesTitles := retrieveMonitoredServiceNames(StoredQueries, profile.Metrics)
	storedQueries := retrieveStoredQueries(config.kibanaApiEndpoint, monitoredStoredQueriesTitles)
	if storedQueries == nil || len(storedQueries) == 0 {
		log.Info("No stored queries retrieved.")
		return nil, nil
	}
	for _, storedQuery := range storedQueries {
		if config.alwaysOverrideTimeFilter || storedQuery.Attributes.Timefilter == nil {
			storedQuery.Attributes.Timefilter = &config.timeFilter
		}
		indexIds := extractIndexIds(storedQuery)
		indexes := retrieveIndexTitles(config.kibanaApiEndpoint, indexIds)
		hits, err := retrieveHits(config.servers, indexes, storedQuery)
		// error happens only if could not initialize client - no sense to continue
		if err != nil {
			log.Error(err)
			break
		}
		if hits == nil {
			log.Info("Hits not found for query: ", storedQuery.Attributes.Title)
			continue
		}
		timeInterval := getTimeInterval(storedQuery)
		hosts, groups = parseHits(storedQuery.Attributes.Title, timeInterval, config.hostNameLabelPath, config.hostGroupLabelPath,
			hosts, groups, hits)
	}
	return hosts, groups
}

func retrieveMonitoredServiceNames(view ElasticView, services []transit.MetricDefinition) []string {
	var names []string
	for _, service := range services {
		if service.ServiceType == string(view) {
			names = append(names, service.Name)
		}
	}
	return names
}

func convertIntervalTemplate(timeValue string) string {
	interval := strconv.Itoa(connectors.Timer) + intervalPeriodSeconds
	if strings.Contains(timeValue, intervalTemplate) {
		timeValue = strings.ReplaceAll(timeValue, intervalTemplate, interval)
	}
	return timeValue
}

func getTimeInterval(storedQuery SavedObject) *transit.TimeInterval {
	location := time.Now().Location()
	startTime := parseTime(storedQuery.Attributes.Timefilter.From, true, location)
	endTime := parseTime(storedQuery.Attributes.Timefilter.To, false, location)
	timeInterval := &transit.TimeInterval{
		EndTime:   milliseconds.MillisecondTimestamp{Time: startTime},
		StartTime: milliseconds.MillisecondTimestamp{Time: endTime},
	}
	return timeInterval
}

func parseHits(queryTitle string, timeInterval *transit.TimeInterval, hostNameLabels []string, hostGroupLabels []string,
	hosts map[string]tempHost, groups map[string]map[string]struct{}, hits []Hit) (map[string]tempHost, map[string]map[string]struct{}) {
	for _, hit := range hits {
		hostName := extractLabelValue(hostNameLabels, hit.Source)
		hostGroupName := extractLabelValue(hostGroupLabels, hit.Source)
		groups = updateHostGroups(hostName, hostGroupName, groups)
		hosts = updateHosts(hostName, queryTitle, hostGroupName, timeInterval, hosts)
	}
	return hosts, groups
}

func extractLabelValue(labelsHierarchy []string, source map[string]interface{}) string {
	var value string
	for i, label := range labelsHierarchy {
		if i != len(labelsHierarchy)-1 {
			source = source[label].(map[string]interface{})
			continue
		} else {
			value = source[label].(string)
		}
	}
	return value
}

func updateHostGroups(hostName string, hostGroupName string, groups map[string]map[string]struct{}) map[string]map[string]struct{} {
	if group, exists := groups[hostGroupName]; exists {
		group[hostName] = struct{}{}
	} else {
		group := make(map[string]struct{})
		group[hostName] = struct{}{}
		groups[hostGroupName] = group
	}
	groups[hostGroupName][hostName] = struct{}{}
	return groups
}

func updateHosts(hostName string, serviceName string, hostGroupName string, timeInterval *transit.TimeInterval,
	hosts map[string]tempHost) map[string]tempHost {
	if host, exists := hosts[hostName]; exists {
		services := host.services
		var found = false
		for i := range services {
			if services[i].name == serviceName {
				services[i].hits = services[i].hits + 1
				found = true
				break
			}
		}
		if !found {
			service := tempService{name: serviceName, hits: 1, timeInterval: timeInterval}
			services = append(services, service)
			host.services = services
		}
		hosts[hostName] = host
	} else {
		service := tempService{name: serviceName, hits: 1, timeInterval: timeInterval}
		host := tempHost{name: hostName, services: []tempService{service}, hostGroup: hostGroupName}
		hosts[hostName] = host
	}
	return hosts
}

func convertHosts(hosts map[string]tempHost) ([]transit.MonitoredResource, []transit.InventoryResource) {
	mrs := make([]transit.MonitoredResource, len(hosts))
	irs := make([]transit.InventoryResource, len(hosts))
	i := 0
	for _, host := range hosts {
		monitoredServices, inventoryServices := convertServices(host)
		monitoredResource, _ := connectors.CreateResource(host.name, monitoredServices)
		inventoryResource := connectors.CreateInventoryResource(host.name, inventoryServices)
		mrs[i] = *monitoredResource
		irs[i] = inventoryResource
		i++
	}
	return mrs, irs
}

func convertServices(host tempHost) ([]transit.MonitoredService, []transit.InventoryService) {
	monitoredServices := make([]transit.MonitoredService, len(host.services))
	inventoryServices := make([]transit.InventoryService, len(host.services))
	for i, service := range host.services {
		metric, _ := connectors.CreateMetric("hits", service.hits, service.timeInterval, transit.UnitCounter)
		monitoredService, _ := connectors.CreateService(service.name, host.name, []transit.TimeSeries{*metric})
		inventoryService := connectors.CreateInventoryService(service.name, host.name)
		monitoredServices[i] = *monitoredService
		inventoryServices[i] = inventoryService
	}
	return monitoredServices, inventoryServices
}

func convertGroups(groups map[string]map[string]struct{}) []transit.ResourceGroup {
	rgs := make([]transit.ResourceGroup, len(groups))
	j := 0
	for group, hostsInGroup := range groups {
		monitoredResourceRefs := make([]transit.MonitoredResourceRef, len(hostsInGroup))
		k := 0
		for host := range hostsInGroup {
			monitoredResourceRef := connectors.CreateMonitoredResourceRef(host, "", transit.Host)
			monitoredResourceRefs[k] = monitoredResourceRef
			k++
		}
		resourceGroup := connectors.CreateResourceGroup(group, group, transit.HostGroup, monitoredResourceRefs)
		rgs[j] = resourceGroup
		j++
	}
	return rgs
}

type elasticConnectorConfig struct {
	servers                  []string
	kibanaApiEndpoint        string
	timeFilter               TimeFilter
	alwaysOverrideTimeFilter bool
	hostNameLabelPath        []string
	hostGroupLabelPath       []string
}

type tempService struct {
	name         string
	hits         int
	timeInterval *transit.TimeInterval
}

type tempHost struct {
	name      string
	services  []tempService
	hostGroup string
}
