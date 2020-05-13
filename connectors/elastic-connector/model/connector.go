package model

import (
	"encoding/json"
	"github.com/gwos/tcg/clients"
	"github.com/gwos/tcg/config"
	"github.com/gwos/tcg/connectors"
	"github.com/gwos/tcg/log"
	"github.com/gwos/tcg/transit"
	"sync"
)

const (
	hitsMetricName              = "hits"
	warningThresholdNameSuffix  = "_wn"
	criticalThresholdNameSuffix = "_cr"
)

var doOnce sync.Once

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
	name       string
	services   []monitoringService
	hostGroups []string
}

func InitMonitoringState(previousState *MonitoringState, config *ElasticConnectorConfig) MonitoringState {
	var currentState MonitoringState

	currentState.Metrics = make(map[string]transit.MetricDefinition)
	for _, metrics := range config.Views {
		for metricName, metric := range metrics {
			currentState.Metrics[metricName] = metric
		}
	}

	currentState.Hosts = make(map[string]monitoringHost)

	doOnce.Do(func() {
		log.Info("Initializing state with GW hosts for agent ", config.AgentId)
		// add hosts form GW to current state
		if config.GWConnections == nil || len(*config.GWConnections) == 0 {
			log.Error("Unable to get GW hosts to initialize state: GW connections are not set.")
		} else {
			gwHosts := retrieveExistingGwHosts(config.AppType, config.AgentId, config.GWConnections)
			if gwHosts != nil {
				currentState.Hosts = gwHosts
			} else {
				log.Info("No GW hosts received.")
			}
		}
	})

	// update with hosts from prev runs
	if previousState != nil && previousState.Hosts != nil {
		for _, host := range previousState.Hosts {
			currentState.Hosts[host.name] = host
		}
	}

	// nullify services
	for _, host := range currentState.Hosts {
		var services []monitoringService
		for metricName := range currentState.Metrics {
			service := monitoringService{name: metricName, hits: 0}
			services = append(services, service)
		}
		host.services = services
		currentState.Hosts[host.name] = host
	}

	// update with groups from prev runs
	currentState.Groups = make(map[string]map[string]struct{})
	if previousState != nil && previousState.Groups != nil {
		currentState.Groups = previousState.Groups
	}

	return currentState
}

func (monitoringState *MonitoringState) UpdateHosts(hostName string, serviceName string, hostGroupName string,
	timeInterval *transit.TimeInterval) {
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
		hostGroups := []string{hostGroupName}
		host.hostGroups = hostGroups
		hosts[hostName] = host
	} else {
		service := monitoringService{name: serviceName, hits: 1, timeInterval: timeInterval}
		hostGroups := []string{hostGroupName}
		host := monitoringHost{name: hostName, services: []monitoringService{service}, hostGroups: hostGroups}
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
		for _, groupName := range host.hostGroups {
			if groupName == "" {
				continue
			}
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
	}
	return groups
}

//
//func UpdateCheckTimes(resources []transit.MonitoredResource, timer float64) {
//	lastCheckTime := time.Now().Local()
//	nextCheckTime := lastCheckTime.Add(time.Second * time.Duration(timer))
//	for i := range resources {
//		resources[i].LastCheckTime = milliseconds.MillisecondTimestamp{Time: lastCheckTime}
//		resources[i].NextCheckTime = milliseconds.MillisecondTimestamp{Time: nextCheckTime}
//		for j := range resources[i].Services {
//			resources[i].Services[j].LastCheckTime = milliseconds.MillisecondTimestamp{Time: lastCheckTime}
//			resources[i].Services[j].NextCheckTime = milliseconds.MillisecondTimestamp{Time: nextCheckTime}
//		}
//	}
//}

func retrieveExistingGwHosts(appType string, agentId string, gwConnections *config.GWConnections) map[string]monitoringHost {
	gwHosts := make(map[string]monitoringHost)

	for _, gwConnection := range *gwConnections {
		gwClient := clients.GWClient{
			AppName:      appType,
			AppType:      appType,
			GWConnection: gwConnection,
		}
		err := gwClient.Connect()
		if err != nil {
			log.Error("Unable to connect to GW to get hosts to initialize state: ", err)
			return gwHosts
		}
		response, err := gwClient.GetServicesByAgent(agentId)
		if err != nil {
			log.Error("Unable to get GW hosts to initialize state: ", err)
			return gwHosts
		}
		var gwServices struct {
			Services []struct {
				HostName string `json:"hostName"`
			} `json:"services"`
		}
		err = json.Unmarshal(response, &gwServices)
		if err != nil {
			log.Error("Unable to parse received GW hosts to initialize state: ", err)
			return gwHosts
		}
		var hostNames []string
		for _, gwService := range gwServices.Services {
			if _, exists := gwHosts[gwService.HostName]; exists {
			} else {
				host := monitoringHost{
					name: gwService.HostName,
				}
				gwHosts[gwService.HostName] = host
				hostNames = append(hostNames, gwService.HostName)
			}
		}

		if hostNames != nil && len(hostNames) > 0 {
			response, err = gwClient.GetHostGroupsByHostNamesAndAppType(hostNames, appType)
			if err != nil {
				log.Error("Unable to get GW host groups to initialize state: ", err)
				return gwHosts
			}
			if response == nil {
				log.Error("Unable to get GW host groups to initialize state.")
				return gwHosts
			}

			var gwHostGroups struct {
				HostGroups []struct {
					Name  string `json:"name"`
					Hosts []struct {
						HostName string `json:"hostName"`
					} `json:"hosts"`
				} `json:"hostGroups"`
			}
			err = json.Unmarshal(response, &gwHostGroups)
			if err != nil {
				log.Error("Unable to parse received GW host groups to initialize state: ", err)
				return gwHosts
			}

			for _, gwHostGroup := range gwHostGroups.HostGroups {
				for _, gwHost := range gwHostGroup.Hosts {
					if host, exists := gwHosts[gwHost.HostName]; exists {
						hostGroups := host.hostGroups
						hostGroups = append(hostGroups, gwHostGroup.Name)
						host.hostGroups = hostGroups
						gwHosts[gwHost.HostName] = host
					}
				}
			}
		}

		if len(gwHosts) > 0 {
			break
		}
	}

	return gwHosts
}
