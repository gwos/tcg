package model

import (
	"github.com/gwos/tng/connectors"
	"github.com/gwos/tng/log"
	"github.com/gwos/tng/transit"
)

const (
	hitsMetricName              = "hits"
	warningThresholdNameSuffix  = "_wn"
	criticalThresholdNameSuffix = "_cr"
)

type MonitoringState struct {
	Metrics map[string]transit.MetricDefinition
	Hosts   map[string]Host
	Groups  map[string]map[string]struct{}
}

type Service struct {
	name         string
	hits         int
	timeInterval *transit.TimeInterval
}

type Host struct {
	name      string
	services  []Service
	hostGroup string
}

func (monitoringState *MonitoringState) UpdateHostGroups(hostName string, hostGroupName string) {
	groups := monitoringState.Groups
	if group, exists := groups[hostGroupName]; exists {
		group[hostName] = struct{}{}
	} else {
		group := make(map[string]struct{})
		group[hostName] = struct{}{}
		groups[hostGroupName] = group
	}
	groups[hostGroupName][hostName] = struct{}{}
	monitoringState.Groups = groups
}

func (monitoringState *MonitoringState) UpdateHosts(hostName string, serviceName string, hostGroupName string,
	timeInterval *transit.TimeInterval) {
	hosts := monitoringState.Hosts
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
			service := Service{name: serviceName, hits: 1, timeInterval: timeInterval}
			services = append(services, service)
			host.services = services
		}
		hosts[hostName] = host
	} else {
		service := Service{name: serviceName, hits: 1, timeInterval: timeInterval}
		host := Host{name: hostName, services: []Service{service}, hostGroup: hostGroupName}
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

func (host Host) toTransitResources(metricDefinitions map[string]transit.MetricDefinition) ([]transit.MonitoredService, []transit.InventoryService) {
	monitoredServices := make([]transit.MonitoredService, len(host.services))
	inventoryServices := make([]transit.InventoryService, len(host.services))
	for i, service := range host.services {
		metric, _ := connectors.CreateMetric(hitsMetricName, service.hits, service.timeInterval, transit.UnitCounter)

		serviceName := service.name
		if metricDefinitions != nil {
			if metricDefinition, has := metricDefinitions[serviceName]; has {
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
			}
		}

		monitoredService, _ := connectors.CreateService(serviceName, host.name, []transit.TimeSeries{*metric})
		inventoryService := connectors.CreateInventoryService(serviceName, host.name)
		monitoredServices[i] = *monitoredService
		inventoryServices[i] = inventoryService
	}
	return monitoredServices, inventoryServices
}

func (monitoringState *MonitoringState) ToResourceGroups() []transit.ResourceGroup {
	groups := monitoringState.Groups
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
