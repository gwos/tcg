package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gwos/tcg/config"
	"github.com/gwos/tcg/connectors"
	"github.com/gwos/tcg/connectors/elastic-connector/clients"
	"github.com/gwos/tcg/log"
	"github.com/gwos/tcg/services"
	"github.com/gwos/tcg/transit"
	"go.opentelemetry.io/otel/label"
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
	Kibana             Kibana              `json:"kibana"`
	Servers            []string            `json:"servers"`
	CustomTimeFilter   clients.KTimeFilter `json:"timefilter"`
	OverrideTimeFilter bool                `json:"override"`
	HostNameField      string              `json:"hostNameLabelPath"`
	HostGroupField     string              `json:"hostGroupLabelPath"`
	GroupNameByUser    bool                `json:"hostGroupNameByUser"`
	CheckInterval      time.Duration       `json:"checkIntervalMinutes"`
	MonitorConnector   bool                `json:"connectorMonitored"`
	AppType            string
	AgentID            string
	GWConnections      config.GWConnections
	Ownership          transit.HostOwnershipType
	Views              map[string]map[string]transit.MetricDefinition
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
		ApiRoot:  cfg.Kibana.ServerName,
		Username: cfg.Kibana.Username,
		Password: cfg.Kibana.Password,
	}
	esClient := clients.EsClient{Servers: cfg.Servers}
	err := esClient.InitEsClient()
	if err != nil {
		log.Error("|elasticConnector.go| : [initClients] : Cannot initialize ES client.")
		return kibanaClient, esClient, errors.New("cannot initialize ES client")
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

// CollectMetrics retrives metric data
func (connector *ElasticConnector) CollectMetrics() ([]transit.DynamicMonitoredResource, []transit.DynamicInventoryResource, []transit.ResourceGroup) {
	var err error

	ctx, spanCollectMetrics := services.StartTraceSpan(context.Background(), "connectors", "CollectMetrics")
	defer func() {
		spanCollectMetrics.SetAttributes(
			label.String("error", fmt.Sprint(err)),
		)
		spanCollectMetrics.End()
	}()
	_, spanMonitoringState := services.StartTraceSpan(ctx, "connectors", "initMonitoringState")

	monitoringState := connector.config.initMonitoringState(connector.monitoringState, &connector.esClient)
	connector.monitoringState = monitoringState

	spanMonitoringState.SetAttributes(
		label.Int("monitoringState.Hosts", len(monitoringState.Hosts)),
		label.Int("monitoringState.Metrics", len(monitoringState.Metrics)),
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
			break
		default:
			log.Warn("Not supported view: ", view)
			break
		}
		if err != nil {
			log.Error("|elasticConnector.go| : [CollectMetrics] : Collection interrupted.")
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
		query := clients.BuildEsQuery(storedQuery)
		timeInterval := storedQuery.Attributes.TimeFilter.ToTimeInterval()
		isAggregatable, err := connector.esClient.IsAggregatable([]string{connector.config.HostNameField}, indexes)
		if err != nil {
			log.Error("|elasticConnector.go| : [collectStoredQueriesMetrics] : Unable to proceed as ES client could not be initialized.")
			return err
		}
		if isAggregatable[connector.config.HostNameField] {
			result, err := connector.esClient.CountHits(connector.config.HostNameField, indexes, &query)
			if err != nil {
				log.Error("|elasticConnector.go| : [collectStoredQueriesMetrics] :" +
					" Unable to proceed as ES client could not be initialized.")
				return err
			}
			for hostName, docsCount := range result {
				connector.monitoringState.updateHost(hostName, storedQuery.Attributes.Title, docsCount, timeInterval)
			}
		} else {
			for hostName := range connector.monitoringState.Hosts {
				hits, err := connector.esClient.CountHitsForHost(hostName, connector.config.HostNameField, indexes, &query)
				if err != nil {
					log.Error("|elasticConnector.go| : [collectStoredQueriesMetrics] :" +
						" Unable to proceed as ES client could not be initialized.")
					return err
				}
				connector.monitoringState.updateHost(hostName, storedQuery.Attributes.Title,
					hits, timeInterval)
			}
		}
	}

	return nil
}

func retrieveMonitoredServiceNames(view ElasticView, metrics map[string]transit.MetricDefinition) []string {
	var srvs []string
	if metrics != nil {
		for _, metric := range metrics {
			if metric.ServiceType == string(view) && metric.Monitored {
				srvs = append(srvs, metric.Name)
			}
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
