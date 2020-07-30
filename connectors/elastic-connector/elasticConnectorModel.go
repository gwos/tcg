package main

import (
	"github.com/gwos/tcg/cache"
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
			if metric.Monitored {
				currentState.Metrics[metricName] = metric
			}
		}
	}

	doOnce.Do(func() {
		log.Info("Initializing state with GW hosts for agent ", config.AgentId)
		// add hosts form GW to current state
		if config.GWConnections == nil || len(config.GWConnections) == 0 {
			log.Error("|elasticConnectorModel.go| : [initMonitoringState] : Unable to get GW hosts to initialize state: GW connections are not set.")
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
		log.Error("|elasticConnectorModel.go| : [updateHost] : Host not found in monitoring state: ", hostName)
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
			log.Error("|elasticConnectorModel.go| : [toTransitResources] : Error when creating resource ", host.name, " Reason: ", err)
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
				Name:        serviceName,
				CustomName:  metricDefinition.CustomName,
				ComputeType: metricDefinition.ComputeType,
				Expression:  metricDefinition.Expression,
				Value:       service.hits,
				UnitType:    transit.UnitCounter,
				Warning:     metricDefinition.WarningThreshold,
				Critical:    metricDefinition.CriticalThreshold,
			}
			if service.timeInterval != nil {
				metricBuilder.StartTimestamp = &service.timeInterval.StartTime
				metricBuilder.EndTimestamp = &service.timeInterval.EndTime
			}

			monitoredService, err := connectors.BuildServiceForMetric(host.name, metricBuilder)
			if err != nil {
				log.Error("|elasticConnectorModel.go| : [toTransitResources] : Error when creating service ",
					host.name, ":", customServiceName, " Reason: ", err)
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
			log.Error("|elasticConnectorModel.go| : [initGwHosts] : Unable to connect to GW to get hosts to initialize state: ", err)
			continue
		}
		gwServices, err := gwClient.GetServicesByAgent(agentId)
		if err != nil || gwServices == nil {
			log.Error("|elasticConnectorModel.go| : [initGwHosts] : Unable to get GW hosts to initialize state.")
			if err != nil {
				log.Error("|elasticConnectorModel.go| : [initGwHosts] : ", err)
			} else {
				log.Error("|elasticConnectorModel.go| : [initGwHosts] : Response is nil.")
			}
			continue
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
			gwHostGroups, err := gwClient.GetHostGroupsByHostNamesAndAppType(hostNames, appType)
			if err != nil || gwHostGroups == nil {
				log.Error("|elasticConnectorModel.go| : [initGwHosts] : Unable to get GW host groups to initialize state.")
				if err != nil {
					log.Error("|elasticConnectorModel.go| : [initGwHosts] : ", err)
				} else {
					log.Error("|elasticConnectorModel.go| : [initGwHosts] : Response is nil.")
				}
				continue
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

		// set count hosts to which services of curr agent are already assigned in GWOS
		// as "last sent hosts count" in cache to not duplicate his when check license before send inventory
		cache.LastSentHostsCountCache.SetDefault(gwConnection.HostName, len(hostNames))
	}

	return gwHosts
}

func initEsHosts(config ElasticConnectorConfig, esClient *ecClients.EsClient) map[string]monitoringHost {
	esHosts := make(map[string]monitoringHost)

	hostNameField := config.HostNameField
	hostGroupField := config.HostGroupField
	groupNameByUser := config.GroupNameByUser

	if groupNameByUser {
		hostGroupField = ""
	}
	hostBuckets := esClient.GetHosts(config.HostNameField, hostGroupField)

	if len(hostBuckets) == 0 {
		if !strings.HasSuffix(hostNameField, ".keyword") || !!strings.HasSuffix(hostGroupField, ".keyword") {
			if !strings.HasSuffix(hostNameField, ".keyword") {
				hostNameField = hostNameField + ".keyword"
			}
			if !groupNameByUser && !strings.HasSuffix(hostGroupField, ".keyword") {
				hostGroupField = hostGroupField + ".keyword"
			}
			hostBuckets = esClient.GetHosts(hostNameField, hostGroupField)
		}
	}

	for _, hostBucket := range hostBuckets {
		hostName := hostBucket.Key
		hostGroupBuckets := hostBucket.HostGroupBuckets
		var hostGroups []string
		if hostGroupBuckets != nil {
			for _, hostGroupBucket := range hostGroupBuckets.Buckets {
				hostGroupName := hostGroupBucket.Key
				hostGroups = append(hostGroups, hostGroupName)
			}
		} else {
			if groupNameByUser {
				hostGroups = append(hostGroups, config.HostGroupField)
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
