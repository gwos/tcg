package main

import (
	"errors"
	"github.com/gwos/tcg/connectors/elastic-connector/clients"
	"github.com/gwos/tcg/log"
	_ "github.com/gwos/tcg/milliseconds"
	"github.com/gwos/tcg/transit"
	"strings"
)

type ElasticView string

const (
	StoredQueries ElasticView = "storedQueries"
	//StoredSearches ElasticView = "storedSearches"
	//KQL            ElasticView = "kql"
	//SelfMonitoring ElasticView = "selfMonitoring"
)

type ElasticConnector struct {
	config          ElasticConnectorConfig
	kibanaClient    clients.KibanaClient
	esClient        clients.EsClient
	monitoringState MonitoringState
}

func (connector *ElasticConnector) LoadConfig(config ElasticConnectorConfig) error {
	kibanaClient, esClient, err := initClients(config)
	if err != nil {
		return err
	}
	monitoringState := initMonitoringState(connector.monitoringState, config)

	connector.config = config
	connector.kibanaClient = kibanaClient
	connector.esClient = esClient
	connector.monitoringState = monitoringState

	return nil
}

func (connector *ElasticConnector) performCollection() {
	mrs, irs, rgs := connector.CollectMetrics()

	log.Info("[Elastic Connector]: Sending inventory ...")
	err := connectors.SendInventory(irs, rgs, transit.Yield) //TODO
	if err != nil {
		log.Error(err.Error())
	}

	log.Info("[Elastic Connector]: Monitoring resources ...")
	err = connectors.SendMetrics(mrs)
	if err != nil {
		log.Error(err.Error())
	}
}

func (connector *ElasticConnector) CollectMetrics() ([]transit.MonitoredResource, []transit.InventoryResource, []transit.ResourceGroup) {
	monitoringState := initMonitoringState(connector.monitoringState, connector.config)
	connector.monitoringState = monitoringState

	var err error
	for view, metrics := range connector.config.Views {
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

	monitoredResources, inventoryResources := monitoringState.toTransitResources()
	resourceGroups := monitoringState.toResourceGroups()
	return monitoredResources, inventoryResources, resourceGroups
}

func (connector *ElasticConnector) ListSuggestions(view string, name string) []string {
	var suggestions []string
	if connector.kibanaClient.ApiRoot == "" {
		// client is not configured yet
		return suggestions
	}
	switch view {
	case string(StoredQueries):
		storedQueries := connector.kibanaClient.RetrieveStoredQueries(nil)
		for _, query := range storedQueries {
			if strings.Contains(query.Attributes.Title, name) {
				suggestions = append(suggestions, query.Attributes.Title)
			}
		}
		break
	default:
		log.Warn("Not supported view: ", view)
		break
	}
	return suggestions
}

func (connector *ElasticConnector) getInventoryHashSum() ([]byte, error) {
	var hosts []string
	var metrics []string
	hostGroups := make(map[string]map[string]struct{})
	monitoringState := connector.monitoringState
	if monitoringState.Hosts != nil {
		for hostName := range monitoringState.Hosts {
			hosts = append(hosts, hostName)
		}
	}
	if monitoringState.Metrics != nil {
		for metricName := range monitoringState.Metrics {
			metrics = append(metrics, metricName)
		}
	}
	hostGroups = monitoringState.buildGroups()
	return connectors.Hashsum(hosts, metrics, hostGroups)
}

func initClients(config ElasticConnectorConfig) (clients.KibanaClient, clients.EsClient, error) {
	kibanaClient := clients.KibanaClient{
		ApiRoot:  config.Kibana.ServerName,
		Username: config.Kibana.Username,
		Password: config.Kibana.Password,
	}
	esClient := clients.EsClient{Servers: config.Servers}
	err := esClient.InitEsClient()
	if err != nil {
		log.Error("Cannot initialize ES client.")
		return kibanaClient, esClient, errors.New("cannot initialize ES client")
	}
	return kibanaClient, esClient, nil
}

func (connector *ElasticConnector) collectStoredQueriesMetrics(titles []string) error {
	storedQueries := connector.kibanaClient.RetrieveStoredQueries(titles)
	if storedQueries == nil || len(storedQueries) == 0 {
		log.Info("No stored queries retrieved.")
		return nil
	}

	for _, storedQuery := range storedQueries {
		if connector.config.OverrideTimeFilter || storedQuery.Attributes.TimeFilter == nil {
			storedQuery.Attributes.TimeFilter = &connector.config.CustomTimeFilter
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

func (connector *ElasticConnector) parseStoredQueryHits(storedQuery clients.SavedObject, hits []clients.Hit) {
	timeInterval := storedQuery.Attributes.TimeFilter.ToTimeInterval()
	for _, hit := range hits {
		hostName := extractLabelValue(connector.config.HostNameLabelPath, hit.Source)
		hostGroupName := extractLabelValue(connector.config.HostGroupLabelPath, hit.Source)
		connector.monitoringState.updateHosts(hostName, storedQuery.Attributes.Title,
			hostGroupName, timeInterval)
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
