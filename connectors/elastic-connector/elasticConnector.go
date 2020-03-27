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

const (
	defaultTimeFilterFrom = "now-$interval"
	defaultTimeFilterTo   = "now"

	defaultHostNameLabel  = "container.name"
	defaultHostGroupLabel = "container.labels.com_docker_compose_project"
)

const (
	intervalTemplate      = "$interval"
	intervalPeriodSeconds = "s"
)

func CollectMetrics() ([]transit.MonitoredResource, []transit.InventoryResource, []transit.ResourceGroup) {
	// TODO: selected queries with overridden time intervals will be retrieved from configs, here we will need only their titles to retrieve filters
	storedQueries := retrieveStoredQueries(nil)
	hosts, groups := make(map[string]tempHost), make(map[string]map[string]struct{})
	for _, storedQuery := range storedQueries {
		overrideTimeFilter(&storedQuery)
		hits := retrieveHits(storedQuery)
		if hits == nil {
			log.Info("Hits not found for query: ", storedQuery.Attributes.Title)
			continue
		}
		timeInterval := getTimeInterval(storedQuery)
		hosts, groups = parseHits(storedQuery.Attributes.Title, timeInterval, hosts, groups, hits)
	}
	monitoredResources, inventoryResources := convertHosts(hosts)
	resourceGroups := convertGroups(groups)
	return monitoredResources, inventoryResources, resourceGroups
}

func overrideTimeFilter(storedQuery *SavedObject) {
	var customTimeFilter TimeFilter
	// TODO: get custom filter from profile, if time filter is not set in profile, then use default i.e. instead of if storedQuery.Attributes.Timefilter == nil here will be if profileTimeFilter == nil
	if storedQuery.Attributes.Timefilter == nil {
		customTimeFilter.From = defaultTimeFilterFrom
		customTimeFilter.To = defaultTimeFilterTo
	} else {
		// TODO: instead of if storedQuery.Attributes.Timefilter here will be profileTimeFilter
		customTimeFilter.From = storedQuery.Attributes.Timefilter.From
		customTimeFilter.To = storedQuery.Attributes.Timefilter.To
	}
	customTimeFilter.From = convertIntervalTemplate(customTimeFilter.From)
	customTimeFilter.To = convertIntervalTemplate(customTimeFilter.To)
	storedQuery.Attributes.Timefilter = &customTimeFilter
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

func parseHits(queryTitle string, timeInterval *transit.TimeInterval, hosts map[string]tempHost, groups map[string]map[string]struct{},
	hits []Hit) (map[string]tempHost, map[string]map[string]struct{}) {
	hostNameLabelsHierarchy := getHostNameLabelsHierarchy()
	hostGroupLabelsHierarchy := getHostGroupLabelsHierarchy()
	for _, hit := range hits {
		hostName := extractLabelValue(hostNameLabelsHierarchy, hit.Source)
		hostGroupName := extractLabelValue(hostGroupLabelsHierarchy, hit.Source)
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

func getHostNameLabelsHierarchy() []string {
	// TODO: get hostNameLabel from config, if not set then use default
	return strings.Split(defaultHostNameLabel, ".")
}

func getHostGroupLabelsHierarchy() []string {
	// TODO: get hostGroupLabel from config, if not set then use default
	return strings.Split(defaultHostGroupLabel, ".")
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
