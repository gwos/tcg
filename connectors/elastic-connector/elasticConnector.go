package main

import (
	"context"
	"errors"
	"github.com/gwos/tcg/connectors"
	"github.com/gwos/tcg/connectors/elastic-connector/clients"
	"github.com/gwos/tcg/log"
	_ "github.com/gwos/tcg/milliseconds"
	"github.com/gwos/tcg/services"
	"github.com/gwos/tcg/transit"
	"go.opentelemetry.io/otel/api/trace"
	"sort"
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
	monitoringState := initMonitoringState(connector.monitoringState, config, &esClient)

	connector.config = config
	connector.kibanaClient = kibanaClient
	connector.esClient = esClient
	connector.monitoringState = monitoringState

	return nil
}

func (connector *ElasticConnector) CollectMetrics() ([]transit.MonitoredResource, []transit.InventoryResource, []transit.ResourceGroup) {
	var err error
	var spanCollectMetrics trace.Span
	var spanMonitoringState trace.Span
	var ctx context.Context

	if services.GetTransitService().TelemetryProvider != nil {
		tr := services.GetTransitService().TelemetryProvider.Tracer("elasticConnector")
		ctx, spanCollectMetrics = tr.Start(context.Background(), "CollectMetrics")
		_, spanMonitoringState = tr.Start(ctx, "initMonitoringState")
		defer func() {
			spanCollectMetrics.SetAttribute("error", err)
			spanCollectMetrics.End()
		}()
	}

	monitoringState := initMonitoringState(connector.monitoringState, connector.config, &connector.esClient)
	connector.monitoringState = monitoringState
	if services.GetTransitService().TelemetryProvider != nil {
		spanMonitoringState.SetAttribute("monitoringState.Hosts", len(monitoringState.Hosts))
		spanMonitoringState.SetAttribute("monitoringState.Metrics", len(monitoringState.Metrics))
		spanMonitoringState.End()
	}

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
			if name == "" || strings.Contains(query.Attributes.Title, name) {
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
		for metricName, metric := range monitoringState.Metrics {
			name := connectors.Name(metricName, metric.CustomName)
			metrics = append(metrics, name)
		}
	}
	sort.Strings(hosts)
	sort.Strings(metrics)

	hostGroups = monitoringState.HostGroups
	groupNames := make([]string, 0, len(hostGroups))
	for groupName := range hostGroups {
		groupNames = append(groupNames, groupName)
	}
	sort.Strings(groupNames)

	groupsSorted := make(map[string][]string, len(hostGroups))
	for _, groupName := range groupNames {
		hosts := hostGroups[groupName]
		hostNames := make([]string, 0, len(hosts))
		for hostName := range hosts {
			hostNames = append(hostNames, hostName)
		}
		sort.Strings(hostNames)
		groupsSorted[groupName] = hostNames
	}
	return connectors.Hashsum(hosts, metrics, groupsSorted)
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
		query := clients.BuildSearchQueryFromStoredQuery(storedQuery)
		timeInterval := storedQuery.Attributes.TimeFilter.ToTimeInterval()
		for hostName := range connector.monitoringState.Hosts {
			hits, err := connector.esClient.CountHitsForHost(hostName, connector.config.HostNameField, indexes, query)
			// error happens only if could not initialize elasticsearch client - no sense to continue
			if err != nil {
				log.Error("Unable to proceed as ES client could not be initialized.")
				return err
			}
			connector.monitoringState.updateHost(hostName, storedQuery.Attributes.Title,
				hits, timeInterval)
		}
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
