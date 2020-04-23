package model

import (
	"github.com/gwos/tng/clients"
	"github.com/gwos/tng/config"
	"github.com/gwos/tng/connectors"
	"github.com/gwos/tng/log"
	"github.com/gwos/tng/milliseconds"
	"github.com/gwos/tng/transit"
	"time"
)

const (
	hitsMetricName              = "hits"
	warningThresholdNameSuffix  = "_wn"
	criticalThresholdNameSuffix = "_cr"
)

type MonitoringState struct {
	Metrics map[string]transit.MetricDefinition
	Hosts   map[string]monitoringHost
	Groups  map[string]map[string]struct{}
}

type monitoringService struct {
	name         string
	hits         int
	timeInterval *transit.TimeInterval
}

type monitoringHost struct {
	name      string
	services  []monitoringService
	hostGroup string
}

func InitMonitoringState(previousState *MonitoringState, config *ElasticConnectorConfig) MonitoringState {
	var currentState MonitoringState

	if config.GWConnection == nil {
		log.Error("Cannot get previous state")
	}
	getPreviousState(config.AppType, config.AgentId, config.GWConnection)
	// TODO build inventory of prev state

	currentState.Metrics = make(map[string]transit.MetricDefinition)
	for _, metrics := range config.Views {
		for metricName, metric := range metrics {
			currentState.Metrics[metricName] = metric
		}
	}

	currentState.Hosts = make(map[string]monitoringHost)
	if previousState != nil && previousState.Hosts != nil {
		for _, host := range previousState.Hosts {
			var services []monitoringService
			for metricName := range currentState.Metrics {
				service := monitoringService{name: metricName, hits: 0}
				services = append(services, service)
			}
			host.services = services
			currentState.Hosts[host.name] = host
		}
	}

	currentState.Groups = make(map[string]map[string]struct{})
	if previousState != nil && previousState.Groups != nil {
		currentState.Groups = previousState.Groups
	}

	return currentState
}

func (monitoringState *MonitoringState) UpdateHosts(hostName string, hostNamePrefix string, serviceName string, hostGroupName string,
	timeInterval *transit.TimeInterval) {
	hostName = hostNamePrefix + hostName
	hosts := monitoringState.Hosts
	if host, exists := hosts[hostName]; exists {
		services := host.services
		for i := range services {
			if services[i].name == serviceName {
				services[i].hits = services[i].hits + 1
				if services[i].timeInterval == nil {
					services[i].timeInterval = timeInterval
				}
				break
			}
		}
		host.hostGroup = hostGroupName
		hosts[hostName] = host
	} else {
		service := monitoringService{name: serviceName, hits: 1, timeInterval: timeInterval}
		host := monitoringHost{name: hostName, services: []monitoringService{service}, hostGroup: hostGroupName}
		hosts[hostName] = host
	}
	monitoringState.Hosts = hosts
}

func (monitoringState *MonitoringState) ToTransitResources() ([]transit.MonitoredResource, []transit.InventoryResource) {
	hosts := monitoringState.Hosts
	mrs := make([]transit.MonitoredResource, len(hosts))
	irs := make([]transit.InventoryResource, len(hosts))
	i := 0
	for _, host := range hosts {
		monitoredServices, inventoryServices := host.toTransitResources(monitoringState.Metrics)
		monitoredResource, _ := connectors.CreateResource(host.name, monitoredServices)
		inventoryResource := connectors.CreateInventoryResource(host.name, inventoryServices)
		mrs[i] = *monitoredResource
		irs[i] = inventoryResource
		i++
	}
	return mrs, irs
}

func (host monitoringHost) toTransitResources(metricDefinitions map[string]transit.MetricDefinition) ([]transit.MonitoredService, []transit.InventoryService) {
	monitoredServices := make([]transit.MonitoredService, len(host.services))
	inventoryServices := make([]transit.InventoryService, len(host.services))
	if metricDefinitions == nil {
		return monitoredServices, inventoryServices
	}
	for i, service := range host.services {
		serviceName := service.name
		if metricDefinition, has := metricDefinitions[serviceName]; has {
			metric, _ := connectors.CreateMetric(hitsMetricName, service.hits, service.timeInterval, transit.UnitCounter)

			if metricDefinition.CustomName != "" {
				serviceName = metricDefinition.CustomName
			}
			warningThreshold, err := connectors.CreateWarningThreshold(hitsMetricName+warningThresholdNameSuffix,
				metricDefinition.WarningThreshold)
			if err != nil {
				log.Error("Error creating warning threshold for metric ", serviceName, ": ", err)
			}
			criticalThreshold, err := connectors.CreateCriticalThreshold(hitsMetricName+criticalThresholdNameSuffix,
				metricDefinition.CriticalThreshold)
			if err != nil {
				log.Error("Error creating critical threshold for metric ", serviceName, ": ", err)
			}
			thresholds := []transit.ThresholdValue{*warningThreshold, *criticalThreshold}
			metric.Thresholds = &thresholds
			monitoredService, _ := connectors.CreateService(serviceName, host.name, []transit.TimeSeries{*metric})
			inventoryService := connectors.CreateInventoryService(serviceName, host.name)
			monitoredServices[i] = *monitoredService
			inventoryServices[i] = inventoryService
		}
	}
	return monitoredServices, inventoryServices
}

func (monitoringState *MonitoringState) ToResourceGroups() []transit.ResourceGroup {
	groups := monitoringState.buildGroups()
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

func (monitoringState *MonitoringState) buildGroups() map[string]map[string]struct{} {
	groups := make(map[string]map[string]struct{})
	for _, host := range monitoringState.Hosts {
		hostName := host.name
		groupName := host.hostGroup
		if group, exists := groups[groupName]; exists {
			group[hostName] = struct{}{}
		} else {
			group := make(map[string]struct{})
			group[hostName] = struct{}{}
			groups[groupName] = group
		}
		groups[groupName][hostName] = struct{}{}
		monitoringState.Groups = groups
	}
	return groups
}

func UpdateCheckTimes(resources []transit.MonitoredResource, timer float64) {
	lastCheckTime := time.Now().Local()
	nextCheckTime := lastCheckTime.Add(time.Second * time.Duration(timer))
	for i := range resources {
		resources[i].LastCheckTime = milliseconds.MillisecondTimestamp{Time: lastCheckTime}
		resources[i].NextCheckTime = milliseconds.MillisecondTimestamp{Time: nextCheckTime}
		for j := range resources[i].Services {
			resources[i].Services[j].LastCheckTime = milliseconds.MillisecondTimestamp{Time: lastCheckTime}
			resources[i].Services[j].NextCheckTime = milliseconds.MillisecondTimestamp{Time: nextCheckTime}
		}
	}
}

func getPreviousState(appType string, agentId string, gwConnection *config.GWConnection) {
	gwClient := clients.GWClient{
		AppName:      appType,
		GWConnection: gwConnection,
	}
	gwClient.Connect()
	services, err := gwClient.GetServices(agentId)
	if err != nil {
		log.Error(err)
	}
	log.Info(services)
}
