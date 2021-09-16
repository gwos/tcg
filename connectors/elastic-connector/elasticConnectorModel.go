package main

import (
	"strings"
	"sync"
	"time"

	"github.com/gwos/tcg/clients"
	"github.com/gwos/tcg/config"
	"github.com/gwos/tcg/connectors"
	ecClients "github.com/gwos/tcg/connectors/elastic-connector/clients"
	"github.com/gwos/tcg/transit"
	"github.com/rs/zerolog/log"
)

var doOnce sync.Once

// status message templates for different statuses
// if template for service's status not listed service's status text will be only thresholds if they exist,
//     otherwise empty
var initStatusMessages = map[transit.MonitorStatus]string{
	transit.ServiceOk:                  "Query matched {value} messages in the last {interval}.",
	transit.ServiceWarning:             "Query matched {value} messages in the last {interval}.",
	transit.ServiceUnscheduledCritical: "Query matched {value} messages in the last {interval}.",
	transit.ServiceScheduledCritical:   "Query matched {value} messages in the last {interval}.",
	transit.ServicePending:             "Service Pending.",
	transit.ServiceUnknown:             "Service Unknown.",
}

// MonitoringState describes state
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

func (connectorConfig *ExtConfig) initMonitoringState(previousState MonitoringState, esClient *ecClients.EsClient) MonitoringState {
	currentState := MonitoringState{
		Metrics: make(map[string]transit.MetricDefinition),
		Hosts:   make(map[string]monitoringHost),
	}

	for _, metrics := range connectorConfig.Views {
		for metricName, metric := range metrics {
			if metric.Monitored {
				currentState.Metrics[metricName] = metric
			}
		}
	}

	doOnce.Do(func() {
		log.Info().Msgf("initializing state with GW hosts for agent %s", connectorConfig.AgentID)
		// add hosts form GW to current state
		if connectorConfig.GWConnections == nil || len(connectorConfig.GWConnections) == 0 {
			log.Error().Msg("could not get GW hosts to initialize state: GW connections are not set")
		} else {
			gwHosts := initGwHosts(connectorConfig.AppType, connectorConfig.AgentID, connectorConfig.GWConnections)
			if gwHosts != nil {
				currentState.Hosts = gwHosts
			} else {
				log.Info().Msg("no GW hosts received")
			}
		}
	})

	// update with hosts from prev runs
	for _, host := range previousState.Hosts {
		currentState.Hosts[host.name] = host
	}

	// update with hosts extracted from ES right now
	esHosts := connectorConfig.initEsHosts(esClient)
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

func (monitoringState *MonitoringState) updateHosts(values map[string]int, serviceName string,
	timeInterval *transit.TimeInterval) {
	for hostName, host := range monitoringState.Hosts {
		if host.services != nil {
			if service, exists := host.services[serviceName]; exists {
				service.hits = values[hostName]
				if service.timeInterval == nil {
					service.timeInterval = timeInterval
				}
				host.services[serviceName] = service
			}
		}
	}
}

func (monitoringState *MonitoringState) toTransitResources() ([]transit.DynamicMonitoredResource, []transit.DynamicInventoryResource) {
	hosts := monitoringState.Hosts
	mrs := make([]transit.DynamicMonitoredResource, len(hosts))
	irs := make([]transit.DynamicInventoryResource, len(hosts))
	i := 0
	for _, host := range hosts {
		monitoredServices, inventoryServices := host.toTransitResources(monitoringState.Metrics)

		inventoryResource := connectors.CreateInventoryResource(host.name, inventoryServices)
		irs[i] = inventoryResource

		monitoredResource, err := connectors.CreateResource(host.name, monitoredServices)
		if err != nil {
			log.Err(err).Msgf("could not create resource %s", host.name)
		}
		if monitoredResource != nil {
			mrs[i] = *monitoredResource
		}

		i++
	}
	return mrs, irs
}

func (host monitoringHost) toTransitResources(metricDefinitions map[string]transit.MetricDefinition) ([]transit.DynamicMonitoredService, []transit.DynamicInventoryService) {
	monitoredServices := make([]transit.DynamicMonitoredService, len(host.services))
	inventoryServices := make([]transit.DynamicInventoryService, len(host.services))
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
				Graphed:     metricDefinition.Graphed,
			}

			var intervalReplacement string
			if service.timeInterval != nil {
				metricBuilder.StartTimestamp = service.timeInterval.StartTime
				metricBuilder.EndTimestamp = service.timeInterval.EndTime

				endTimeNano := service.timeInterval.EndTime.UnixNano()
				startTimeNano := service.timeInterval.StartTime.UnixNano()
				timeInterval := time.Duration(endTimeNano - startTimeNano)
				intervalReplacement = connectors.FormatTimeForStatusMessage(timeInterval, time.Minute)
			}

			// copy status message templates already replacing {interval} where applicable
			statusMessages := make(map[transit.MonitorStatus]string)
			for status, statusMessage := range initStatusMessages {
				var message string
				// if service has its own interval replace "{interval}" pattern in all status messages with this value now
				// otherwise it will be replaced by check interval in minutes
				if strings.Contains(statusMessage, "{interval}") && intervalReplacement != "" {
					message = strings.ReplaceAll(statusMessage, "{interval}", intervalReplacement)
				} else {
					message = statusMessage
				}
				statusMessages[status] = message
			}
			monitoredService, err := connectors.BuildServiceForMetricWithStatusText(host.name, metricBuilder, statusMessages)
			if err != nil {
				log.Err(err).Msgf("could not create service %s:%s", host.name, customServiceName)
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

func initGwHosts(appType string, agentID string, gwConnections config.GWConnections) map[string]monitoringHost {
	gwHosts := make(map[string]monitoringHost)

	for _, gwConnection := range gwConnections {
		gwClient := clients.GWClient{
			AppName:      appType,
			AppType:      appType,
			GWConnection: gwConnection,
		}
		err := gwClient.Connect()
		if err != nil {
			log.Err(err).Msg("could not connect to GW to get hosts to initialize state")
			continue
		}
		gwServices, err := gwClient.GetServicesByAgent(agentID)
		if err != nil || gwServices == nil {
			log.Error().Err(err).
				Msg("could not get GW hosts to initialize state")
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

		if len(hostNames) > 0 {
			gwHostGroups, err := gwClient.GetHostGroupsByHostNamesAndAppType(hostNames, appType)
			if err != nil || gwHostGroups == nil {
				log.Error().Err(err).
					Msg("could not get GW host groups to initialize state")
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
	}

	return gwHosts
}

func (connectorConfig *ExtConfig) initEsHosts(esClient *ecClients.EsClient) map[string]monitoringHost {
	esHosts := make(map[string]monitoringHost)

	hostNameField := connectorConfig.HostNameField
	hostGroupField := &connectorConfig.HostGroupField
	groupNameByUser := connectorConfig.GroupNameByUser

	if groupNameByUser {
		hostGroupField = nil
	}
	fieldNames := []string{hostNameField}
	if hostGroupField != nil {
		fieldNames = append(fieldNames, *hostGroupField)
	}

	isAggregatable, err := esClient.IsAggregatable(fieldNames, nil)
	if err != nil {
		log.Error().Msg("could not retrieve ES hosts")
		return esHosts
	}
	allAggregatable := true
	if !isAggregatable[hostNameField] && !strings.HasSuffix(hostNameField, ".keyword") {
		allAggregatable = false
		hostNameField = hostNameField + ".keyword"
	}
	if hostGroupField != nil {
		if !isAggregatable[*hostGroupField] && !strings.HasSuffix(*hostGroupField, ".keyword") {
			allAggregatable = false
			newHostGroupField := *hostGroupField + ".keyword"
			hostGroupField = &newHostGroupField
		}
	}

	if !allAggregatable {
		fieldNames = []string{hostNameField}
		if hostGroupField != nil {
			fieldNames = append(fieldNames, *hostGroupField)
		}
		isAggregatable, err = esClient.IsAggregatable(fieldNames, nil)
		if isAggregatable[hostNameField] && (hostGroupField == nil || isAggregatable[*hostGroupField]) {
			allAggregatable = true
		}
	}

	if !allAggregatable {
		log.Error().
			Strs("fieldNames", fieldNames).
			Msgf("could not retrieve ES hosts: not all fields are aggregatable")
		return esHosts
	}

	connectorConfig.HostNameField = hostNameField
	if !groupNameByUser && hostGroupField != nil {
		connectorConfig.HostGroupField = *hostGroupField
	}

	keys, err := esClient.GetHosts(connectorConfig.HostNameField, hostGroupField)

	for _, key := range keys {
		hostNameKey := key.Host
		hostGroupKey := key.HostGroup
		if esHost, exists := esHosts[hostNameKey]; exists {
			if !groupNameByUser && hostGroupKey != nil {
				esHostGroups := esHost.hostGroups
				esHostGroups = append(esHostGroups, *hostGroupKey)
				esHost.hostGroups = esHostGroups
				esHosts[hostNameKey] = esHost
			}
		} else {
			var hostGroups []string
			if groupNameByUser {
				hostGroups = []string{connectorConfig.HostGroupField}
			} else {
				if hostGroupKey != nil {
					hostGroups = []string{*hostGroupKey}
				}
			}
			host := monitoringHost{
				name:       hostNameKey,
				hostGroups: hostGroups,
			}
			esHosts[hostNameKey] = host
		}
	}

	return esHosts
}
