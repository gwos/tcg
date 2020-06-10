package main

import (
	"encoding/json"
	"github.com/gwos/tcg/clients"
	"github.com/gwos/tcg/config"
	"github.com/gwos/tcg/connectors"
	ecClients "github.com/gwos/tcg/connectors/elastic-connector/clients"
	"github.com/gwos/tcg/log"
	"github.com/gwos/tcg/transit"
	"strings"
	"sync"
)

var doOnce sync.Once

type MonitoringState struct {
	Metrics    map[string]transit.MetricDefinition
	Hosts      map[string]monitoringHost
	HostGroups map[string]map[string]struct{}
}

type monitoringService struct {
	name         string
	hits         int
	timeInterval *transit.TimeInterval
}

type monitoringHost struct {
	name       string
	services   map[string]monitoringService
	hostGroups []string
}

func initMonitoringState(previousState MonitoringState, config ElasticConnectorConfig, esClient *ecClients.EsClient) MonitoringState {
	currentState := MonitoringState{
		Metrics: make(map[string]transit.MetricDefinition),
		Hosts:   make(map[string]monitoringHost),
	}

	for _, metrics := range config.Views {
		for metricName, metric := range metrics {
			currentState.Metrics[metricName] = metric
		}
	}

	doOnce.Do(func() {
		log.Info("Initializing state with GW hosts for agent ", config.AgentId)
		// add hosts form GW to current state
		if config.GWConnections == nil || len(config.GWConnections) == 0 {
			log.Error("Unable to get GW hosts to initialize state: GW connections are not set.")
		} else {
			gwHosts := initGwHosts(config.AppType, config.AgentId, config.GWConnections)
			if gwHosts != nil {
				currentState.Hosts = gwHosts
			} else {
				log.Info("No GW hosts received.")
			}
		}
	})

	// update with hosts from prev runs
	for _, host := range previousState.Hosts {
		currentState.Hosts[host.name] = host
	}

	// update with hosts extracted from ES right now
	esHosts := initEsHosts(config, esClient)
	for _, host := range esHosts {
		currentState.Hosts[host.name] = host
	}

	// nullify services
	if currentState.Metrics != nil {
		for _, host := range currentState.Hosts {
			host.services = make(map[string]monitoringService, len(currentState.Metrics))
			for metricName := range currentState.Metrics {
				service := monitoringService{name: metricName, hits: 0}
				host.services[metricName] = service
			}
			currentState.Hosts[host.name] = host
		}
	}

	return currentState
}

func (monitoringState *MonitoringState) updateHost(hostName string, serviceName string, value int, timeInterval *transit.TimeInterval) {
	if host, exists := monitoringState.Hosts[hostName]; exists {
		if host.services != nil {
			if service, exists := host.services[serviceName]; exists {
				service.hits = value
				if service.timeInterval == nil {
					service.timeInterval = timeInterval
				}
				host.services[serviceName] = service
			}
		}
		monitoringState.Hosts[hostName] = host
	} else {
		log.Error("[Elastic Connector]: Host not found in monitoring state: ", hostName)
	}
}

func (monitoringState *MonitoringState) toTransitResources() ([]transit.MonitoredResource, []transit.InventoryResource) {
	hosts := monitoringState.Hosts
	mrs := make([]transit.MonitoredResource, len(hosts))
	irs := make([]transit.InventoryResource, len(hosts))
	i := 0
	for _, host := range hosts {
		monitoredServices, inventoryServices := host.toTransitResources(monitoringState.Metrics)

		inventoryResource := connectors.CreateInventoryResource(host.name, inventoryServices)
		irs[i] = inventoryResource

		monitoredResource, err := connectors.CreateResource(host.name, monitoredServices)
		if err != nil {
			log.Error("Error when creating resource ", host.name)
			log.Error(err)
		}
		if monitoredResource != nil {
			mrs[i] = *monitoredResource
		}

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
	i := 0
	for serviceName, service := range host.services {
		if metricDefinition, has := metricDefinitions[serviceName]; has {
			customServiceName := connectors.Name(serviceName, metricDefinition.CustomName)

			inventoryService := connectors.CreateInventoryService(customServiceName, host.name)
			inventoryServices[i] = inventoryService

			metricBuilder := connectors.MetricBuilder{
				Name:       serviceName,
				CustomName: metricDefinition.CustomName,
				Value:      service.hits,
				UnitType:   transit.UnitCounter,
				Warning:    metricDefinition.WarningThreshold,
				Critical:   metricDefinition.CriticalThreshold,
			}
			if service.timeInterval != nil {
				metricBuilder.StartTimestamp = &service.timeInterval.StartTime
				metricBuilder.EndTimestamp = &service.timeInterval.EndTime
			}

			monitoredService, err := connectors.BuildServiceForMetric(host.name, metricBuilder)
			if err != nil {
				log.Error("Error when creating service ", host.name, ":", customServiceName)
				log.Error(err)
			}
			if monitoredService != nil {
				monitoredServices[i] = *monitoredService
			}
		}
		i = i + 1
	}
	return monitoredServices, inventoryServices
}

func (monitoringState *MonitoringState) toResourceGroups() []transit.ResourceGroup {
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
	for hostName, host := range monitoringState.Hosts {
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
		}
	}
	monitoringState.HostGroups = groups
	return groups
}

func initGwHosts(appType string, agentId string, gwConnections config.GWConnections) map[string]monitoringHost {
	gwHosts := make(map[string]monitoringHost)

	for _, gwConnection := range gwConnections {
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

func initEsHosts(config ElasticConnectorConfig, esClient *ecClients.EsClient) map[string]monitoringHost {
	esHosts := make(map[string]monitoringHost)

	hostNameField := config.HostNameField
	hostGroupField := config.HostGroupField
	hostBuckets := esClient.GetHosts(hostNameField, hostGroupField)

	if len(hostBuckets) == 0 {
		if !strings.HasSuffix(hostNameField, ".keyword") || !!strings.HasSuffix(hostGroupField, ".keyword") {
			if !strings.HasSuffix(hostNameField, ".keyword") {
				hostNameField = hostNameField + ".keyword"
			}
			if !strings.HasSuffix(hostGroupField, ".keyword") {
				hostGroupField = hostGroupField + ".keyword"
			}
			hostBuckets = esClient.GetHosts(hostNameField, hostGroupField)
		}
	}

	for _, hostBucket := range hostBuckets {
		hostName := hostBucket.Key
		hostGroupBuckets := hostBucket.HostGroupBuckets.Buckets
		var hostGroups []string
		if hostGroupBuckets != nil {
			for _, hostGroupBucket := range hostGroupBuckets {
				hostGroupName := hostGroupBucket.Key
				hostGroups = append(hostGroups, hostGroupName)
			}
		}
		host := monitoringHost{
			name:       hostName,
			hostGroups: hostGroups,
		}
		esHosts[hostName] = host
	}

	return esHosts
}
