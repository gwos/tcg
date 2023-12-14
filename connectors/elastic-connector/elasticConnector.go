package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gwos/tcg/config"
	"github.com/gwos/tcg/connectors"
	"github.com/gwos/tcg/connectors/elastic-connector/clients"
	"github.com/gwos/tcg/sdk/transit"
	"github.com/gwos/tcg/services"
	"github.com/gwos/tcg/tracing"
	"github.com/rs/zerolog/log"
	"go.opentelemetry.io/otel/attribute"
)

// ElasticView describes flow
type ElasticView string

// Define flows
const (
	StoredQueries ElasticView = "storedQueries"
	//StoredSearches ElasticView = "storedSearches"
	//KQL            ElasticView = "kql"
	//SelfMonitoring ElasticView = "selfMonitoring"
)

// Kibana defines the connection props
type Kibana struct {
	ServerName string `json:"serverName"`
	Username   string `json:"userName"`
	Password   string `json:"password"`
}

// ExtConfig defines the MonitorConnection extensions configuration
// extended with general configuration fields
type ExtConfig struct {
	Kibana                Kibana              `json:"kibana"`
	Servers               []string            `json:"servers"`
	CustomTimeFilter      clients.KTimeFilter `json:"timefilter"`
	OverrideTimeFilter    bool                `json:"override"`
	FilterHostsWithLucene string              `json:"filterHostsWithLucene"`
	HostNameField         string              `json:"hostNameLabelPath"`
	HostGroupField        string              `json:"hostGroupLabelPath"`
	GroupNameByUser       bool                `json:"hostGroupNameByUser"`
	CheckInterval         time.Duration       `json:"checkIntervalMinutes"`
	AppType               string
	AgentID               string
	GWConnections         config.GWConnections
	Ownership             transit.HostOwnershipType
	Views                 map[string]map[string]transit.MetricDefinition
}

// UnmarshalJSON implements json.Unmarshaler.
func (cfg *ExtConfig) UnmarshalJSON(input []byte) error {
	type plain ExtConfig
	c := plain(*cfg)
	if err := json.Unmarshal(input, &c); err != nil {
		return err
	}
	if c.CheckInterval != cfg.CheckInterval {
		c.CheckInterval = c.CheckInterval * time.Minute
	}
	if c.CustomTimeFilter.Override != nil {
		c.OverrideTimeFilter = *c.CustomTimeFilter.Override
		c.CustomTimeFilter.Override = nil
	}
	*cfg = ExtConfig(c)
	return nil
}

const (
	// default config values
	defaultProtocol                 = "http"
	defaultElasticServer            = "http://localhost:9200"
	defaultKibanaServerName         = "http://localhost:5601/"
	defaultKibanaUsername           = ""
	defaultKibanaPassword           = ""
	defaultTimeFilterFrom           = "now-$interval"
	defaultTimeFilterTo             = "now"
	defaultAlwaysOverrideTimeFilter = false
	defaultHostNameLabel            = "container.name.keyword"
	defaultHostGroupLabel           = "container.labels.com_docker_compose_project.keyword"
	defaultHostGroupName            = "Elastic Search"
	defaultGroupNameByUser          = false

	intervalTemplate = "$interval"
)

func initClients(cfg ExtConfig) (clients.KibanaClient, clients.EsClient, error) {
	kibanaClient := clients.KibanaClient{
		APIRoot:  cfg.Kibana.ServerName,
		Username: cfg.Kibana.Username,
		Password: cfg.Kibana.Password,
	}
	esClient := clients.EsClient{
		Addresses: cfg.Servers,
		Username:  cfg.Kibana.Username,
		Password:  cfg.Kibana.Password,
	}
	if err := kibanaClient.InitClient(); err != nil {
		err = fmt.Errorf("could not initialize kibana client: %w", err)
		log.Err(err).Msg("")
		return kibanaClient, esClient, err
	}
	if err := esClient.InitClient(); err != nil {
		err = fmt.Errorf("could not initialize ES client: %w", err)
		log.Err(err).Msg("")
		return kibanaClient, esClient, err
	}

	return kibanaClient, esClient, nil
}

func (cfg *ExtConfig) replaceIntervalTemplates() {
	cfg.CustomTimeFilter.From = replaceIntervalTemplate(cfg.CustomTimeFilter.From,
		cfg.CheckInterval)
	cfg.CustomTimeFilter.To = replaceIntervalTemplate(cfg.CustomTimeFilter.To,
		cfg.CheckInterval)
}

func replaceIntervalTemplate(templateString string, intervalValue time.Duration) string {
	interval := fmt.Sprintf("%ds", int64(intervalValue.Seconds()))
	if strings.Contains(templateString, intervalTemplate) {
		templateString = strings.ReplaceAll(templateString, intervalTemplate, interval)
	}
	return templateString
}

// ElasticConnector handles the state
type ElasticConnector struct {
	config          ExtConfig
	kibanaClient    clients.KibanaClient
	esClient        clients.EsClient
	monitoringState MonitoringState
}

// LoadConfig updates state
func (connector *ElasticConnector) LoadConfig(config ExtConfig) error {
	clients.FilterHostsWithLucene = config.FilterHostsWithLucene

	kibanaClient, esClient, err := initClients(config)
	if err != nil {
		return err
	}
	monitoringState := config.initMonitoringState(connector.monitoringState, &esClient)

	connector.config = config
	connector.kibanaClient = kibanaClient
	connector.esClient = esClient
	connector.monitoringState = monitoringState

	return nil
}

// CollectMetrics retrieves metric data
func (connector *ElasticConnector) CollectMetrics() ([]transit.MonitoredResource, []transit.InventoryResource, []transit.ResourceGroup) {
	var err error

	ctx, spanCollectMetrics := tracing.StartTraceSpan(context.Background(), "connectors", "CollectMetrics")
	defer func() {
		spanCollectMetrics.SetAttributes(
			attribute.String("error", fmt.Sprint(err)),
		)
		spanCollectMetrics.End()
	}()
	_, spanMonitoringState := tracing.StartTraceSpan(ctx, "connectors", "initMonitoringState")

	monitoringState := connector.config.initMonitoringState(connector.monitoringState, &connector.esClient)
	connector.monitoringState = monitoringState

	spanMonitoringState.SetAttributes(
		attribute.Int("monitoringState.Hosts", len(monitoringState.Hosts)),
		attribute.Int("monitoringState.Metrics", len(monitoringState.Metrics)),
	)
	spanMonitoringState.End()

	for view, metrics := range connector.config.Views {
		for metricName, metric := range metrics {
			connector.monitoringState.Metrics[metricName] = metric
		}
		switch view {
		case string(StoredQueries):
			queries := retrieveMonitoredServiceNames(StoredQueries, metrics)
			err = connector.collectStoredQueriesMetrics(queries)
		default:
			log.Warn().Str("view", view).Msg("not supported view")
		}
		if err != nil {
			log.Err(err).Msg("collection interrupted")
			break
		}
	}

	monitoredResources, inventoryResources := monitoringState.toTransitResources()
	resourceGroups := monitoringState.toResourceGroups()
	return monitoredResources, inventoryResources, resourceGroups
}

// ListSuggestions provides suggestions by view
func (connector *ElasticConnector) ListSuggestions(view string, name string) []string {
	var suggestions []string
	if connector.kibanaClient.APIRoot == "" {
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
	default:
		log.Warn().Str("view", view).Msg("not supported view")
	}
	return suggestions
}

func (connector *ElasticConnector) getInventoryHashSum() ([]byte, error) {
	var (
		hosts   []string
		metrics []string
	)
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

	hostGroups := monitoringState.HostGroups
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

func (connector *ElasticConnector) collectStoredQueriesMetrics(titles []string) error {
	storedQueries := connector.kibanaClient.RetrieveStoredQueries(titles)
	if len(storedQueries) == 0 {
		log.Info().Msg("no stored queries retrieved")
		return nil
	}

	for _, storedQuery := range storedQueries {
		if connector.config.OverrideTimeFilter || storedQuery.Attributes.TimeFilter == nil {
			storedQuery.Attributes.TimeFilter = &connector.config.CustomTimeFilter
		}
		indexes := connector.kibanaClient.RetrieveIndexTitles(storedQuery)
		query := clients.BuildEsQuery(storedQuery)
		timeInterval := storedQuery.Attributes.TimeFilter.ToTimeInterval()
		isAggregatable, err := connector.esClient.IsAggregatable([]string{connector.config.HostNameField}, indexes)
		if err != nil {
			log.Err(err).Msg("unable to proceed as ES client could not be initialized")
			return err
		}
		var result map[string]int
		if isAggregatable[connector.config.HostNameField] {
			result, err = connector.esClient.CountHits(connector.config.HostNameField, indexes, &query)
			if err != nil {
				log.Err(err).Msg("unable to proceed as ES client could not be initialized")
				return err
			}
		} else {
			result := make(map[string]int)
			for hostName := range connector.monitoringState.Hosts {
				hits, err := connector.esClient.CountHitsForHost(hostName, connector.config.HostNameField, indexes, &query)
				if err != nil {
					log.Err(err).Msg("unable to proceed as ES client could not be initialized")
					return err
				}
				result[hostName] = hits
			}
		}
		connector.monitoringState.updateHosts(result, storedQuery.Attributes.Title, timeInterval)
	}

	return nil
}

func retrieveMonitoredServiceNames(view ElasticView, metrics map[string]transit.MetricDefinition) []string {
	var srvs []string
	for _, metric := range metrics {
		if metric.ServiceType == string(view) && metric.Monitored {
			srvs = append(srvs, metric.Name)
		}
	}
	return srvs
}

func initializeEntrypoints() []services.Entrypoint {
	return []services.Entrypoint{
		{
			URL:    "/suggest/:viewName",
			Method: http.MethodGet,
			Handler: func(c *gin.Context) {
				c.JSON(http.StatusOK, connector.ListSuggestions(c.Param("viewName"), ""))
			},
		},
		{
			URL:    "/suggest/:viewName/:name",
			Method: http.MethodGet,
			Handler: func(c *gin.Context) {
				c.JSON(http.StatusOK, connector.ListSuggestions(c.Param("viewName"), c.Param("name")))
			},
		},
		{
			URL:    "/expressions/suggest/:name",
			Method: http.MethodGet,
			Handler: func(c *gin.Context) {
				c.JSON(http.StatusOK, connectors.ListExpressions(c.Param("name")))
			},
		},
		{
			URL:     "/expressions/evaluate",
			Method:  http.MethodPost,
			Handler: connectors.EvaluateExpressionHandler,
		},
	}
}
