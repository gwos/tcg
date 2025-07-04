package apm

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang/snappy"
	"github.com/gwos/tcg/connectors"
	"github.com/gwos/tcg/sdk/clients"
	sdklog "github.com/gwos/tcg/sdk/log"
	"github.com/gwos/tcg/sdk/transit"
	"github.com/gwos/tcg/services"
	dto "github.com/prometheus/client_model/go"
	"github.com/prometheus/common/expfmt"
	"github.com/rs/zerolog/log"
)

var (
	inventoryGroupsStorage = make(map[string]map[string][]transit.ResourceRef)
	availableMetrics       = make(map[string][]string)
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

type promMetricsData struct {
	resource      string
	resourceIndex int
	data          []byte
	isStatusDown  bool
	isProtobuf    bool
	withFilters   bool
}

func (mi *promMetricsData) parse() (*[]transit.MonitoredResource, *[]transit.ResourceGroup, error) {
	var (
		promParser         PromParser
		textParser         expfmt.TextParser
		prometheusServices map[string]*dto.MetricFamily
	)
	if mi.isProtobuf {
		dest := make([]byte, len(mi.data))
		dst, err := snappy.Decode(dest, mi.data)
		if err != nil {
			return nil, nil, err
		}
		prometheusServices, err = promParser.Parse(dst, mi.withFilters, mi.resource)
		if err != nil {
			return nil, nil, err
		}
	} else {
		ps, err := textParser.TextToMetricFamilies(strings.NewReader(string(mi.data)))
		prometheusServices = make(map[string]*dto.MetricFamily, 0)
		availableMetrics[mi.resource] = []string{}
		for key, service := range ps {
			availableMetrics[mi.resource] = append(availableMetrics[mi.resource], *service.Name)
			if (mi.withFilters && profileContainsMetric(metricsProfile, *service.Name)) ||
				!mi.withFilters {
				prometheusServices[key] = service
			}
		}
		if err != nil {
			return nil, nil, err
		}
	}
	groups := make(map[string][]transit.ResourceRef)
	monitoredResources, err := parsePrometheusServices(prometheusServices, groups, mi.resource, mi.resourceIndex)
	if err != nil {
		return nil, nil, err
	}
	resourceGroups := constructResourceGroups(groups)
	return monitoredResources, &resourceGroups, nil
}

func (mi *promMetricsData) process() error {
	monitoredResources, resourceGroups, err := mi.parse()
	if err != nil {
		return err
	}

	if mi.isStatusDown {
		for i := 0; i < len(*monitoredResources); i++ {
			(*monitoredResources)[i].Status = transit.HostUnscheduledDown
		}
	}

	if err = connectors.SendMetrics(context.Background(), *monitoredResources, resourceGroups); err != nil {
		return err
	}

	return nil
}

func validatePrometheusService(service *dto.MetricFamily) error {
	if service.Name == nil {
		return errors.New("prometheus service name can not be empty")
	}
	return nil
}

func makeValue(serviceName string, metricType *dto.MetricType, metric *dto.Metric) map[string]any {
	result := make(map[string]any)
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
	groups map[string][]transit.ResourceRef, hostsMap map[string]map[string][]connectors.MetricBuilder,
	resource string, resourceIndex int, hostToDeviceMap map[string]string) {
	var groupName, hostName, serviceName, device string

	for _, metric := range prometheusService.GetMetric() {
		groupName, hostName, serviceName, device = "", "", "", ""
		timestamp := transit.NewTimestamp()
		if metric.TimestampMs != nil {
			*timestamp = transit.Timestamp{Time: time.Unix(0, *metric.TimestampMs*int64(time.Millisecond))}
		}
		values := makeValue(*prometheusService.Name, prometheusService.Type, metric)
		if len(values) == 0 {
			log.Error().Msgf("value for metric '%s' can not be empty", *prometheusService.Name)
			continue
		}

		for name, value := range values {
			metricBuilder := connectors.MetricBuilder{
				Name:           name,
				Value:          value,
				StartTimestamp: timestamp,
				EndTimestamp:   timestamp,
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
						metricBuilder.Critical = -1
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

			// process overrides
			ok, graphed, customName, warning, critical := getOverrides(name)
			if ok {
				if customName != "" {
					metricBuilder.Name = customName
				}
				if warning != -1 {
					metricBuilder.Warning = warning
				}
				if critical != -1 {
					metricBuilder.Critical = critical
				}
				metricBuilder.Graphed = graphed
			}

			// process mappings
			if ok, hostOverride := applyResourceMapping(resource); ok {
				hostName = hostOverride
			}
			// applyLabelMapping for hosts overrides applyResourceMapping results
			if ok, hostOverride := applyLabelMapping(mappings.HostLabel, metricBuilder.Tags); ok {
				hostName = hostOverride
			}
			if ok, serviceOverride := applyLabelMapping(mappings.ServiceLabel, metricBuilder.Tags); ok {
				serviceName = serviceOverride
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

			// Sanitize host and service names to exclude special characters
			hostName = connectors.SanitizeString(hostName)
			serviceName = connectors.SanitizeString(serviceName)

			// build the host->service->metric tree
			host, hostFound := hostsMap[hostName]
			if !hostFound {
				host = make(map[string][]connectors.MetricBuilder)
				hostsMap[hostName] = host
			}
			_, metricsFound := host[serviceName]
			if !metricsFound {
				host[serviceName] = []connectors.MetricBuilder{}
			}
			host[serviceName] = append(host[serviceName], metricBuilder)

			// add or update the groups collection
			_, groupFound := groups[groupName]
			if !groupFound {
				groups[groupName] = []transit.ResourceRef{}
			}
			if !containsRef(groups[groupName], hostName) {
				groups[groupName] = append(groups[groupName], transit.ResourceRef{
					Name: hostName,
					Type: transit.ResourceTypeHost,
				})
			}
			if resourceIndex != -1 {
				inventoryGroupsStorage[extConfig.Resources[resourceIndex].URL] = groups
			}
		}
	}
}

func containsRef(refs []transit.ResourceRef, hostName string) bool {
	for _, host := range refs {
		if host.Name == hostName {
			return true
		}
	}
	return false
}

func constructResourceGroups(groups map[string][]transit.ResourceRef) []transit.ResourceGroup {
	resourceGroups := make([]transit.ResourceGroup, 0, len(groups))
	for groupName, resources := range groups {
		resourceGroups = append(resourceGroups, transit.ResourceGroup{
			GroupName: groupName,
			Type:      transit.HostGroup,
			Resources: resources,
		})
	}
	return resourceGroups
}

func parsePrometheusServices(
	prometheusServices map[string]*dto.MetricFamily,
	groups map[string][]transit.ResourceRef,
	resource string,
	resourceIndex int,
) (*[]transit.MonitoredResource, error) {
	var (
		hostsMap        = make(map[string]map[string][]connectors.MetricBuilder)
		hostToDeviceMap = make(map[string]string)
	)

	for _, prometheusService := range prometheusServices {
		if len(prometheusService.GetMetric()) == 0 {
			continue
		}
		if err := validatePrometheusService(prometheusService); err != nil {
			log.Err(err).Msg("could not validate prometheus service")
			continue
		}
		extractIntoMetricBuilders(prometheusService, groups, hostsMap, resource, resourceIndex, hostToDeviceMap)
	}

	monitoredResources := make([]transit.MonitoredResource, 0, len(hostsMap))

	for hostName, host := range hostsMap {
		ss := make([]transit.MonitoredService, 0, len(host))
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
							if mStatus == nil || transit.MonitorStatusWeightService[monitorStatus] > transit.MonitorStatusWeightService[*mStatus] {
								mStatus = &monitorStatus
							}
						}
					}
					if message, ok := m.Tags["message"]; ok {
						service.LastPluginOutput = message
						delete(m.Tags, "message")
					}
				}
				if mStatus != nil {
					service.Status = *mStatus
				}
				ss = append(ss, *service)
			}
		}
		monitoredResource, err := connectors.CreateResource(hostName, hostToDeviceMap[hostName], ss)
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

func getOverrides(metricName string) (bool, bool, string, int, int) {
	for _, override := range metricsProfile.Metrics {
		if override.Name == metricName {
			return true, override.Graphed, override.CustomName, override.WarningThreshold, override.CriticalThreshold
		}
	}
	return false, false, "", -1, -1
}

// initializeEntrypoints - function for setting entrypoints,
// that will be available through the Connector API
func initializeEntrypoints() []services.Entrypoint {
	return []services.Entrypoint{
		{
			URL:     "/metrics/job/:name",
			Method:  http.MethodPost,
			Handler: receiverHandler,
		},
		{
			URL:    "/metrics/available",
			Method: http.MethodGet,
			Handler: func(c *gin.Context) {
				resultSet := make([]string, 0)
				for _, arr := range availableMetrics {
					resultSet = append(resultSet, arr...)
				}
				c.JSON(http.StatusOK, resultSet)
			},
		},
	}
}

func receiverHandler(c *gin.Context) {
	body, err := c.GetRawData()
	logEvt := log.Err(err).
		Str("content-type", c.GetHeader("Content-Type"))

	if err != nil {
		logEvt.Msg("could not process Prometheus Push")
		c.JSON(http.StatusBadRequest, err.Error())
		return
	}
	if len(body) < 2 {
		logEvt.Msg("received Prometheus Push heartbeat...")
		return
	}
	isProtobuf := c.GetHeader("Content-Type") == "application/x-protobuf"
	pmd := promMetricsData{
		resource:      "",
		resourceIndex: -1,
		data:          body,
		isStatusDown:  false,
		isProtobuf:    isProtobuf,
		withFilters:   false,
	}
	if err = pmd.process(); err != nil {
		logEvt.Discard() // recreate log event with error level
		log.Err(err).
			Str("content-type", c.GetHeader("Content-Type")).
			Msg("could not process Prometheus Push")
		c.JSON(http.StatusBadRequest, err.Error())
	}
}

func pull(resources []Resource) {
	for index, resource := range resources {
		req := clients.Req{
			URL:     resource.URL,
			Method:  http.MethodGet,
			Headers: resource.Headers,
		}
		err := req.Send()

		if err != nil {
			sdklog.Logger.LogAttrs(context.Background(), slog.LevelError, "could not pull data from resource", req.LogAttrs()...)
			continue
		}
		if req.Status != 200 && req.Status != 201 && req.Status != 220 {
			sdklog.Logger.LogAttrs(context.Background(), slog.LevelError, "could not pull data from resource", req.Details()...)
			continue
		}

		pmd := promMetricsData{
			resource:      resource.URL,
			resourceIndex: index,
			data:          req.Response,
			isStatusDown:  req.Status == 220,
			isProtobuf:    false,
			withFilters:   true,
		}
		if err = pmd.process(); err != nil {
			log.Err(err).Msg("could not process metrics")
		}
	}
}
