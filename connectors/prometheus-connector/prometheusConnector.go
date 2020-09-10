package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/gwos/tcg/clients"
	"github.com/gwos/tcg/connectors"
	"github.com/gwos/tcg/log"
	"github.com/gwos/tcg/milliseconds"
	"github.com/gwos/tcg/services"
	"github.com/gwos/tcg/transit"
	dto "github.com/prometheus/client_model/go"
	"github.com/prometheus/common/expfmt"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"
)

// Resource defines Prometheus Source
type Resource struct {
	Headers map[string]string `json:"headers"`
	URL     string            `json:"url"`
}

// UnmarshalJSON implements json.Unmarshaler.
func (r *Resource) UnmarshalJSON(input []byte) error {
	p := struct {
		Headers []struct {
			Key   string `json:"key"`
			Value string `json:"value"`
		} `json:"headers"`
		URL string `json:"url"`
	}{}

	if err := json.Unmarshal(input, &p); err != nil {
		return err
	}

	r.URL = p.URL
	for _, h := range p.Headers {
		r.Headers[h.Key] = h.Value
	}

	return nil
}

// ExtConfig defines the MonitorConnection extensions configuration
type ExtConfig struct {
	Groups           []transit.ResourceGroup   `json:"groups"`
	Resources        []Resource                `json:"resources"`
	Services         []string                  `json:"services"`
	CheckInterval    time.Duration             `json:"checkIntervalMinutes"`
	Ownership        transit.HostOwnershipType `json:"ownership,omitempty"`
	DefaultHost      string                    `json:"defaultHost"`
	DefaultHostGroup string                    `json:"defaultHostGroup"`
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
	*cfg = ExtConfig(c)
	return nil
}

const defaultHostName = "Prometheus-Host"
const defaultHostGroupName = "Servers"

var parser expfmt.TextParser

// Synchronize makes InventoryResource
func Synchronize() *[]transit.InventoryResource {
	var inventoryResources []transit.InventoryResource
	for _, resource := range connectors.Inventory {
		inventoryResources = append(inventoryResources, resource)
	}
	return &inventoryResources
}

func parsePrometheusBody(body []byte) (*[]transit.MonitoredResource, *[]transit.ResourceGroup, error) {
	prometheusServices, err := parser.TextToMetricFamilies(strings.NewReader(string(body)))
	if err != nil {
		return nil, nil, err
	}
	groups := make(map[string][]transit.MonitoredResourceRef)

	monitoredResources, err := parsePrometheusServices(prometheusServices, groups)
	if err != nil {
		return nil, nil, err
	}
	resourceGroups := constructResourceGroups(groups)
	return monitoredResources, &resourceGroups, nil
}

func processMetrics(body []byte) error {
	monitoredResources, groups, err := parsePrometheusBody(body)
	if err != nil {
		return err
	}
	inventory := connectors.BuildInventory(monitoredResources)
	if connectors.ValidateInventory(*inventory) {
		err := connectors.SendMetrics(*monitoredResources)
		if err != nil {
			return err
		}
	} else {
		err := connectors.SendInventory(*inventory, *groups, transit.Yield)
		if err != nil {
			return err
		}
		time.Sleep(2 * time.Second) // TODO: better way to assure synch completion?
		err = connectors.SendMetrics(*monitoredResources)
		if err != nil {
			return err
		}
	}
	return nil
}

func validatePrometheusService(service *dto.MetricFamily) error {
	if service.Name == nil {
		return errors.New("Prometheus service name can not be empty ")
	}
	return nil
}

func makeValue(serviceName string, metricType *dto.MetricType, metric *dto.Metric) map[string]interface{} {
	result := make(map[string]interface{})
	switch *metricType {
	case dto.MetricType_COUNTER:
		result[serviceName] = metric.GetCounter().GetValue()
	case dto.MetricType_UNTYPED:
		result[serviceName] = metric.GetUntyped().GetValue()
	case dto.MetricType_GAUGE:
		result[serviceName] = metric.GetGauge().GetValue()
	case dto.MetricType_HISTOGRAM:
		if metric.GetHistogram().SampleSum != nil {
			result[serviceName+"_sample_sum"] = metric.GetHistogram().GetSampleSum()
		}
		if metric.GetHistogram().SampleCount != nil {
			result[serviceName+"_sample_count"] = float64(metric.GetHistogram().GetSampleCount())
		}
		for i, bucket := range metric.GetHistogram().GetBucket() {
			result[fmt.Sprintf("%s_%s_%d", serviceName, "bucket", i)] = float64(bucket.GetCumulativeCount())
		}
	}
	// TODO: need summary
	return result
}

// extracts from Prometheus format to intermediate Host Maps format
// modifies hostsMap parameter
func extractIntoMetricBuilders(prometheusService *dto.MetricFamily,
	groups map[string][]transit.MonitoredResourceRef,
	hostsMap map[string]map[string][]connectors.MetricBuilder) {
	var groupName, hostName, serviceName string

	for _, metric := range prometheusService.GetMetric() {
		groupName, hostName, serviceName = "", "", ""
		var timestamp = time.Now()
		if metric.TimestampMs != nil {
			timestamp = time.Unix(*metric.TimestampMs, 0)
		}
		values := makeValue(*prometheusService.Name, prometheusService.Type, metric)
		if len(values) == 0 {
			log.Error(fmt.Sprintf("[Prometheus Connector]:  Value for metric '%s' can not be empty", *prometheusService.Name))
			continue
		}

		for name, value := range values {
			metricBuilder := connectors.MetricBuilder{
				Name:           name,
				Value:          value,
				StartTimestamp: &milliseconds.MillisecondTimestamp{Time: timestamp},
				EndTimestamp:   &milliseconds.MillisecondTimestamp{Time: timestamp},
				Graphed:        true,
				Tags:           make(map[string]string),
			}

			for _, label := range metric.GetLabel() {
				switch *label.Name {
				case "unitType":
					metricBuilder.UnitType = *label.Value
				case "resource":
					hostName = *label.Value
				case "warning":
					if value, err := strconv.ParseFloat(*label.Value, 64); err == nil {
						metricBuilder.Warning = value
					} else {
						metricBuilder.Warning = -1
					}
				case "critical":
					if value, err := strconv.ParseFloat(*label.Value, 64); err == nil {
						metricBuilder.Critical = value
					} else {
						metricBuilder.Warning = -1
					}
				case "service":
					serviceName = *label.Value
				case "group":
					groupName = *label.Value
				default:
					metricBuilder.Tags[*label.Name] = *label.Value
				}
			}

			// process defaults
			if groupName == "" {
				if extConfig.DefaultHostGroup == "" {
					groupName = defaultHostGroupName
				} else {
					groupName = extConfig.DefaultHostGroup
				}
			}
			if hostName == "" {
				if extConfig.DefaultHost == "" {
					hostName = defaultHostName
				} else {
					hostName = extConfig.DefaultHost
				}
			}
			if serviceName == "" {
				serviceName = name
			}

			// build the host->service->metric tree
			host, hostFound := hostsMap[hostName]
			if !hostFound {
				host = make(map[string][]connectors.MetricBuilder)
				hostsMap[hostName] = host
			}
			metrics, metricsFound := host[serviceName]
			if !metricsFound {
				metrics = []connectors.MetricBuilder{}
				host[serviceName] = metrics
			}
			host[serviceName] = append(host[serviceName], metricBuilder)

			// add or update the groups collection
			refs, groupFound := groups[groupName]
			if !groupFound {
				refs = []transit.MonitoredResourceRef{}
				refs = groups[groupName]
			}
			if !containsRef(refs, hostName) {
				groups[groupName] = append(groups[groupName], transit.MonitoredResourceRef{
					Name: hostName,
					Type: transit.Host,
				})
			}
		}
	}
}

func containsRef(refs []transit.MonitoredResourceRef, hostName string) bool {
	for _, host := range refs {
		if host.Name == hostName {
			return true
		}
	}
	return false
}

func constructResourceGroups(groups map[string][]transit.MonitoredResourceRef) []transit.ResourceGroup {
	var resourceGroups []transit.ResourceGroup
	for groupName, resources := range groups {
		resourceGroups = append(resourceGroups, transit.ResourceGroup{
			GroupName: groupName,
			Type:      transit.HostGroup,
			Resources: resources,
		})
	}
	return resourceGroups
}

func parsePrometheusServices(prometheusServices map[string]*dto.MetricFamily, groups map[string][]transit.MonitoredResourceRef) (*[]transit.MonitoredResource, error) {
	var monitoredResources []transit.MonitoredResource
	hostsMap := make(map[string]map[string][]connectors.MetricBuilder)
	for _, prometheusService := range prometheusServices {
		if len(prometheusService.GetMetric()) == 0 {
			continue
		}

		if err := validatePrometheusService(prometheusService); err != nil {
			log.Error(fmt.Sprintf("[Prometheus Connector]: %s", err.Error()))
			continue
		}
		extractIntoMetricBuilders(prometheusService, groups, hostsMap)
	}
	for hostName, host := range hostsMap {
		monitoredResource, err := connectors.CreateResource(hostName)
		if err != nil {
			return nil, err
		}

		// sort the keys for hashSum consistency
		keys := make([]string, 0, len(host))
		for k := range host {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		// for serviceName, metrics := range host {
		for _, serviceName := range keys {
			metrics := host[serviceName]
			if service, err := connectors.BuildServiceForMetrics(serviceName, hostName, metrics); err == nil {
				monitoredResource.Services = append(monitoredResource.Services, *service)
			}
		}
		monitoredResources = append(monitoredResources, *monitoredResource)
	}
	return &monitoredResources, nil
}

// initializeEntrypoints - function for setting entrypoints,
// that will be available through the Connector API
func initializeEntrypoints() []services.Entrypoint {
	return []services.Entrypoint{{
		URL:     "/metrics/job/:name",
		Method:  http.MethodPost,
		Handler: receiverHandler,
	}}
}

func receiverHandler(c *gin.Context) {
	body, err := c.GetRawData()
	if err != nil {
		log.Error("|prometheusConnector.go| : [receiverHandler] : ", err)
		c.JSON(http.StatusBadRequest, err.Error())
		return
	}
	err = processMetrics(body)
	if err != nil {
		log.Error(fmt.Sprintf("[Prometheus Connector]~[Push]: %s", err))
	}
}

func pull(resources []Resource) {
	for _, resource := range resources {
		statusCode, byteResponse, err := clients.SendRequest(http.MethodGet, resource.URL, resource.Headers, nil, nil)

		if err != nil {
			log.Error(fmt.Sprintf("[Prometheus Connector]~[Pull]: Can not get data from resource [%s]. Reason: %s.",
				resource.URL, err.Error()))
			continue
		}
		if !(statusCode == 200 || statusCode == 201) {
			log.Error(fmt.Sprintf("[Prometheus Connector]~[Pull]: Can not get data from resource [%s]. Reason: %s.",
				resource.URL, string(byteResponse)))
			continue
		}
		err = processMetrics(byteResponse)
		if err != nil {
			log.Error(fmt.Sprintf("[Prometheus Connector]~[Pull]: %s", err))
		}
	}
}
