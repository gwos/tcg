package main

import (
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
	"strconv"
	"strings"
	"time"
)

var parser expfmt.TextParser

func Synchronize() *[]transit.InventoryResource {
	var inventoryResources []transit.InventoryResource
	for _, resource := range connectors.Inventory {
		inventoryResources = append(inventoryResources, resource)
	}
	return &inventoryResources
}

func parsePrometheusBody(body []byte) (*transit.MonitoredResource, *[]transit.ResourceGroup, error) {
	prometheusServices, err := parser.TextToMetricFamilies(strings.NewReader(string(body)))
	if err != nil {
		return nil, nil, err
	}
	var hostName string
	groups := make(map[string][]transit.MonitoredResourceRef)

	monitoredServices, err := parsePrometheusServices(prometheusServices, &hostName, &groups)
	if err != nil {
		return nil, nil, err
	}

	resourceGroups := constructResourceGroups(groups)

	monitoredResource, err := connectors.CreateResource(hostName)
	if err != nil {
		return nil, nil, err
	}

	monitoredResource.Services = *monitoredServices

	return monitoredResource, &resourceGroups, nil
}

func processMetrics(body []byte) error {
	monitoredResource, groups, err := parsePrometheusBody(body)
	if err != nil {
		return err
	}
	inventory := connectors.BuildInventory(&[]transit.MonitoredResource{*monitoredResource})
	if connectors.ValidateInventory(*inventory) {
		err := connectors.SendMetrics([]transit.MonitoredResource{*monitoredResource})
		if err != nil {
			return err
		}
	} else {
		err := connectors.SendInventory(*inventory, *groups, transit.Yield)
		if err != nil {
			return err
		}
		time.Sleep(2 * time.Second)
		err = connectors.SendMetrics([]transit.MonitoredResource{*monitoredResource})
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
	return result
}

func extractIntoMetricBuilders(prometheusService *dto.MetricFamily, hostName *string, groups *map[string][]transit.MonitoredResourceRef) []connectors.MetricBuilder {
	var metricBuilders []connectors.MetricBuilder

	for _, metric := range prometheusService.GetMetric() {
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
			}

			for _, label := range metric.GetLabel() {
				switch *label.Name {
				case "unitType":
					metricBuilder.UnitType = *label.Value
				case "resource":
					*hostName = *label.Value
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
				}

				for _, label := range metric.GetLabel() {
					switch *label.Name {
					case "group":
						(*groups)[*label.Value] = append((*groups)[*label.Value], transit.MonitoredResourceRef{
							Name: *hostName,
							Type: transit.Host,
						})
					}
				}
			}
			metricBuilders = append(metricBuilders, metricBuilder)
		}
	}
	return metricBuilders
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

func parsePrometheusServices(prometheusServices map[string]*dto.MetricFamily, hostName *string, groups *map[string][]transit.MonitoredResourceRef) (*[]transit.MonitoredService, error) {
	var monitoredServices []transit.MonitoredService
	for _, prometheusService := range prometheusServices {
		if len(prometheusService.GetMetric()) == 0 {
			continue
		}

		if err := validatePrometheusService(prometheusService); err != nil {
			log.Error(fmt.Sprintf("[Prometheus Connector]: %s", err.Error()))
			continue
		}

		metricBuilders := extractIntoMetricBuilders(prometheusService, hostName, groups)

		if *hostName == "" {
			return nil, errors.New("HostName cannot be empty")
		}

		if service, err := connectors.BuildServiceForMetrics(*prometheusService.Name, *hostName, metricBuilders); err == nil {
			monitoredServices = append(monitoredServices, *service)
		} else {
			return nil, err
		}
	}
	return &monitoredServices, nil
}

// initializeEntrypoints - function for setting entrypoints,
// that will be available through the Server Connector API
func initializeEntrypoints() []services.Entrypoint {
	return append(make([]services.Entrypoint, 1),
		services.Entrypoint{
			Url:     "/metrics/job/:name",
			Method:  "Post",
			Handler: receiverHandler,
		},
	)
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
		statusCode, byteResponse, err := clients.SendRequest(http.MethodGet, resource.url, resource.headers, nil, nil)

		if err != nil {
			log.Error(fmt.Sprintf("[Prometheus Connector]~[Pull]: Can not get data from resource [%s]. Reason: %s.",
				resource.url, err.Error()))
			continue
		}
		if !(statusCode == 200 || statusCode == 201) {
			log.Error(fmt.Sprintf("[Prometheus Connector]~[Pull]: Can not get data from resource [%s]. Reason: %s.",
				resource.url, string(byteResponse)))
			continue
		}
		err = processMetrics(byteResponse)
		if err != nil {
			log.Error(fmt.Sprintf("[Prometheus Connector]~[Pull]: %s", err))
		}
	}
}
