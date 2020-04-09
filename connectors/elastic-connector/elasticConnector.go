package main

import (
	"github.com/gwos/tng/connectors/elastic-connector/clients"
	"github.com/gwos/tng/connectors/elastic-connector/model"
	"github.com/gwos/tng/log"
	_ "github.com/gwos/tng/milliseconds"
	"github.com/gwos/tng/transit"
)

type ElasticView string

const (
	StoredQueries ElasticView = "storedQueries"
	//StoredSearches ElasticView = "storedSearches"
	//KQL            ElasticView = "kql"
	//SelfMonitoring ElasticView = "selfMonitoring"
)

type ElasticConnector struct {
	Config          *model.ElasticConnectorConfig
	kibanaClient    *clients.KibanaClient
	esClient        *clients.EsClient
	monitoringState *model.MonitoringState
}

func (connector *ElasticConnector) CollectMetrics() ([]transit.MonitoredResource, []transit.InventoryResource, []transit.ResourceGroup) {
	if connector.Config == nil {
		log.Error("ElasticConnector config is missing.")
		return nil, nil, nil
	}

	views := connector.Config.Views
	if views == nil || len(views) == 0 {
		log.Info("No views provided.")
		return nil, nil, nil
	}

	kibanaClient := clients.KibanaClient{ApiRoot: connector.Config.KibanaServer}
	esClient := clients.EsClient{Servers: connector.Config.Servers}
	err := esClient.InitEsClient()
	if err != nil {
		log.Error("Cannot perform collection.")
		return nil, nil, nil
	}
	monitoringState := model.MonitoringState{
		Metrics: make(map[string]transit.MetricDefinition),
		Hosts:   make(map[string]model.Host),
		Groups:  make(map[string]map[string]struct{}),
	}
	connector.kibanaClient = &kibanaClient
	connector.esClient = &esClient
	connector.monitoringState = &monitoringState

	for view, metrics := range views {
		for metricName, metric := range metrics {
			connector.monitoringState.Metrics[metricName] = metric
		}
		switch view {
		case string(StoredQueries):
			queries := retrieveMonitoredServiceNames(StoredQueries, metrics)
			err = connector.collectStoredQueriesMetrics(queries)
			break
		default:
			log.Warn("Not supported view: ", view)
			break
		}
		if err != nil {
			log.Error("Collection interrupted.")
			break
		}

	}

	monitoredResources, inventoryResources := monitoringState.ToTransitResources()
	model.UpdateCheckTimes(monitoredResources, connector.Config.Timer)
	resourceGroups := monitoringState.ToResourceGroups()
	return monitoredResources, inventoryResources, resourceGroups
}

func (connector *ElasticConnector) collectStoredQueriesMetrics(titles []string) error {
	storedQueries := connector.kibanaClient.RetrieveStoredQueries(titles)
	if storedQueries == nil || len(storedQueries) == 0 {
		log.Info("No stored queries retrieved.")
		return nil
	}

	for _, storedQuery := range storedQueries {
		if connector.Config.OverrideTimeFilter || storedQuery.Attributes.TimeFilter == nil {
			storedQuery.Attributes.TimeFilter = &connector.Config.CustomTimeFilter
		}
		indexes := connector.kibanaClient.RetrieveIndexTitles(storedQuery)

		hits, err := connector.esClient.RetrieveHits(indexes, storedQuery)
		// error happens only if could not initialize elasticsearch client - no sense to continue
		if err != nil {
			log.Error("Unable to proceed as ES client could not be initialized.")
			return err
		}
		if hits == nil {
			log.Info("No Hits found for query: ", storedQuery.Attributes.Title)
			continue
		}
		connector.parseStoredQueryHits(storedQuery, hits)
	}

	return nil
}

func retrieveMonitoredServiceNames(view ElasticView, metrics map[string]transit.MetricDefinition) []string {
	var services []string
	if metrics != nil {
		for _, metric := range metrics {
			if metric.ServiceType == string(view) && metric.Monitored {
				services = append(services, metric.Name)
			}
		}
	}
	return services
}

func (connector *ElasticConnector) parseStoredQueryHits(storedQuery model.SavedObject, hits []model.Hit) {
	timeInterval := storedQuery.Attributes.TimeFilter.ToTimeInterval()
	for _, hit := range hits {
		hostName := extractLabelValue(connector.Config.HostNameLabelPath, hit.Source)
		hostGroupName := extractLabelValue(connector.Config.HostGroupLabelPath, hit.Source)
		connector.monitoringState.UpdateHostGroups(hostName, hostGroupName)
		connector.monitoringState.UpdateHosts(hostName, storedQuery.Attributes.Title, hostGroupName, timeInterval)
	}
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
