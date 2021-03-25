package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gwos/tcg/clients"
	"github.com/gwos/tcg/connectors"
	"github.com/gwos/tcg/log"
	"github.com/gwos/tcg/milliseconds"
	"github.com/gwos/tcg/services"
	"github.com/gwos/tcg/transit"
	dto "github.com/prometheus/client_model/go"
	"github.com/prometheus/common/expfmt"
)

var (
	inventoryGroupsStorage = make(map[string]map[string][]transit.MonitoredResourceRef)
)

// Resource defines APM Source
type Resource struct {
	Headers          map[string]string `json:"headers"`
	URL              string            `json:"url"`
	DefaultHost      string            `json:"defaultHost"`
	DefaultHostGroup string            `json:"defaultHostGroup"`
}

// UnmarshalJSON implements json.Unmarshaler.
func (resource *Resource) UnmarshalJSON(input []byte) error {
	config := struct {
		Headers []struct {
			Key   string `json:"key"`
			Value string `json:"value"`
		} `json:"headers"`
		URL              string `json:"url"`
		DefaultHost      string `json:"defaultHost"`
		DefaultHostGroup string `json:"defaultHostGroup"`
	}{}

	if err := json.Unmarshal(input, &config); err != nil {
		return err
	}

	resource.URL = config.URL
	if resource.Headers == nil {
		resource.Headers = make(map[string]string)
	}
	for _, h := range config.Headers {
		resource.Headers[h.Key] = h.Value
	}
	resource.DefaultHost = config.DefaultHost
	resource.DefaultHostGroup = config.DefaultHostGroup

	return nil
}

// ExtConfig defines the MonitorConnection extensions configuration
type ExtConfig struct {
	Groups        []transit.ResourceGroup   `json:"groups"`
	Resources     []Resource                `json:"resources"`
	Services      []string                  `json:"services"`
	CheckInterval time.Duration             `json:"checkIntervalMinutes"`
	Ownership     transit.HostOwnershipType `json:"ownership,omitempty"`
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

const defaultHostName = "APM-Host"
const defaultHostGroupName = "Servers"

func parsePrometheusBody(body []byte, resourceIndex int) (*[]transit.DynamicMonitoredResource, *[]transit.ResourceGroup, error) {
	var parser expfmt.TextParser
	prometheusServices, err := parser.TextToMetricFamilies(strings.NewReader(string(body)))
	if err != nil {
		return nil, nil, err
	}
	groups := make(map[string][]transit.MonitoredResourceRef)

	monitoredResources, err := parsePrometheusServices(prometheusServices, groups, resourceIndex)
	if err != nil {
		return nil, nil, err
	}
	resourceGroups := constructResourceGroups(groups)
	return monitoredResources, &resourceGroups, nil
}

func processMetrics(body []byte, resourceIndex int, statusDown bool) error {
	if monitoredResources, resourceGroups, err := parsePrometheusBody(body, resourceIndex); err == nil {
		if statusDown {
			for i := 0; i < len(*monitoredResources); i++ {
				(*monitoredResources)[i].Status = transit.HostUnscheduledDown
			}
		}
		if err := connectors.SendMetrics(context.Background(), *monitoredResources, resourceGroups); err != nil {
			return err
		}
	} else {
		return err
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
	hostsMap map[string]map[string][]connectors.MetricBuilder, resourceIndex int, hostToDeviceMap map[string]string) {
	var groupName, hostName, serviceName, device string

	for _, metric := range prometheusService.GetMetric() {
		groupName, hostName, serviceName, device = "", "", "", ""
		var timestamp = time.Now()
		if metric.TimestampMs != nil {
			timestamp = time.Unix(0, *metric.TimestampMs*int64(time.Millisecond))
		}
		values := makeValue(*prometheusService.Name, prometheusService.Type, metric)
		if len(values) == 0 {
			log.Error(fmt.Sprintf("[APM Connector]:  Value for metric '%s' can not be empty", *prometheusService.Name))
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
				case "device":
					device = *label.Value
				default:
					metricBuilder.Tags[*label.Name] = *label.Value
				}
			}

			// process defaults
			if groupName == "" {
				if resourceIndex == -1 || extConfig.Resources[resourceIndex].DefaultHostGroup == "" {
					groupName = defaultHostGroupName
				} else {
					groupName = extConfig.Resources[resourceIndex].DefaultHostGroup
				}
			}
			if hostName == "" {
				if resourceIndex == -1 || extConfig.Resources[resourceIndex].DefaultHost == "" {
					hostName = defaultHostName
				} else {
					hostName = extConfig.Resources[resourceIndex].DefaultHost
				}
			}
			if serviceName == "" {
				serviceName = name
			}
			hostToDeviceMap[hostName] = device

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
			if resourceIndex != -1 {
				inventoryGroupsStorage[extConfig.Resources[resourceIndex].URL] = groups
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

func parsePrometheusServices(prometheusServices map[string]*dto.MetricFamily,
	groups map[string][]transit.MonitoredResourceRef, resourceIndex int) (*[]transit.DynamicMonitoredResource, error) {
	var monitoredResources []transit.DynamicMonitoredResource
	hostsMap := make(map[string]map[string][]connectors.MetricBuilder)
	hostToDeviceMap := make(map[string]string)
	for _, prometheusService := range prometheusServices {
		if len(prometheusService.GetMetric()) == 0 {
			continue
		}
		if err := validatePrometheusService(prometheusService); err != nil {
			log.Error(fmt.Sprintf("[APM Connector]: %s", err.Error()))
			continue
		}
		extractIntoMetricBuilders(prometheusService, groups, hostsMap, resourceIndex, hostToDeviceMap)
	}
	for hostName, host := range hostsMap {
		services := make([]transit.DynamicMonitoredService, 0, len(host))
		/* sort names for hashSum consistency */
		serviceNames := make([]string, 0, len(host))
		for s := range host {
			serviceNames = append(serviceNames, s)
		}
		sort.Strings(serviceNames)
		for _, serviceName := range serviceNames {
			metrics := host[serviceName]
			if service, err := connectors.BuildServiceForMetrics(serviceName, hostName, metrics); err == nil {
				var mStatus *transit.MonitorStatus = nil
				for _, m := range metrics {
					if status, ok := m.Tags["status"]; ok {
						if monitorStatus, err := parseStatus(status); err == nil {
							delete(m.Tags, "status")
							if mStatus == nil || connectors.MonitorStatusWeightService[monitorStatus] > connectors.MonitorStatusWeightService[*mStatus] {
								mStatus = &monitorStatus
							}
						}
					}
					if message, ok := m.Tags["message"]; ok {
						service.LastPlugInOutput = message
						delete(m.Tags, "message")
					}
				}
				if mStatus != nil {
					service.Status = *mStatus
				}
				services = append(services, *service)
			}
		}
		monitoredResource, err := connectors.CreateResource(hostName, hostToDeviceMap[hostName], services)
		if err != nil {
			return nil, err
		}
		monitoredResources = append(monitoredResources, *monitoredResource)
	}
	return &monitoredResources, nil
}

func parseStatus(str string) (transit.MonitorStatus, error) {
	switch strings.ToUpper(str) {
	case "OK", "SERVICE_OK":
		return transit.ServiceOk, nil
	case "WARNING", "SERVICE_WARNING":
		return transit.ServiceWarning, nil
	case "CRITICAL", "UNSCHEDULED_CRITICAL", "SERVICE_UNSCHEDULED_CRITICAL":
		return transit.ServiceUnscheduledCritical, nil
	case "PENDING", "SERVICE_PENDING":
		return transit.ServicePending, nil
	case "SCHEDULED_CRITICAL", "SERVICE_SCHEDULED_CRITICAL":
		return transit.ServiceScheduledCritical, nil
	case "UNKNOWN", "SERVICE_UNKNOWN":
		return transit.ServiceUnknown, nil
	default:
		return transit.ServiceUnknown, errors.New("unknown status provided")
	}
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
		log.Error("|apmConnector.go| : [receiverHandler] : ", err)
		c.JSON(http.StatusBadRequest, err.Error())
		return
	}
	err = processMetrics(body, -1, false)
	if err != nil {
		log.Error(fmt.Sprintf("[APM Connector]~[Push]: %s", err))
	}
}

func pull(resources []Resource) {
	for index, resource := range resources {
		statusCode, byteResponse, err := clients.SendRequest(http.MethodGet, resource.URL, resource.Headers, nil, nil)

		if err != nil {
			log.Error(fmt.Sprintf("[APM Connector]~[Pull]: Can not get data from resource [%s]. Reason: %s.",
				resource.URL, err.Error()))
			continue
		}
		if !(statusCode == 200 || statusCode == 201 || statusCode == 220) {
			log.Error(fmt.Sprintf("[APM Connector]~[Pull]: Can not get data from resource [%s]. Reason: %s.",
				resource.URL, string(byteResponse)))
			continue
		}
		err = processMetrics(byteResponse, index, statusCode == 220)
		if err != nil {
			log.Error(fmt.Sprintf("[APM Connector]~[Pull]: %s", err))
		}
	}
}
