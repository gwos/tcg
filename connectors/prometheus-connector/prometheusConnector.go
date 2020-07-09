package main

import (
	"bytes"
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

var chksum []byte
var inventory = make(map[string]transit.InventoryResource)

var parser expfmt.TextParser

func Synchronize() *[]transit.InventoryResource {
	var inventoryResources []transit.InventoryResource
	for _, resource := range inventory {
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
	var groups []string

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

func validateInventory(inventory *[]transit.InventoryResource) bool {
	if chksum != nil {
		chk, err := connectors.Hashsum(inventory)
		if err != nil || !bytes.Equal(chksum, chk) {
			chksum = chk
			return false
		}
		return true
	} else {
		chksum, _ = connectors.Hashsum(inventory)
		return false
	}
}

func buildInventory(resource *transit.MonitoredResource) *[]transit.InventoryResource {
	var inventoryServices []transit.InventoryService
	for _, service := range resource.Services {
		inventoryServices = append(inventoryServices, connectors.CreateInventoryService(service.Name,
			service.Owner))
	}

	inventoryResource := connectors.CreateInventoryResource(resource.Name, inventoryServices)
	inventory[inventoryResource.Name] = inventoryResource
	return &[]transit.InventoryResource{inventoryResource}
}

func processMetrics(body []byte) error {
	monitoredResource, groups, err := parsePrometheusBody(body)
	if err != nil {
		return err
	}
	inventory := buildInventory(monitoredResource)
	if validateInventory(inventory) {
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

func removeDuplicates(intSlice []string) []string {
	keys := make(map[string]bool)
	list := []string{}
	for _, entry := range intSlice {
		if _, value := keys[entry]; !value {
			keys[entry] = true
			list = append(list, entry)
		}
	}
	return list
}

func validatePrometheusService(service *dto.MetricFamily) error {
	if service.Name == nil {
		return errors.New("Prometheus service name can not be empty ")
	}
	return nil
}

func value(serviceName string, metricType *dto.MetricType, metric *dto.Metric) map[string]interface{} {
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

func extractIntoMetricBuilders(prometheusService *dto.MetricFamily, hostName *string, groups *[]string) []connectors.MetricBuilder {
	var metricBuilders []connectors.MetricBuilder

	for _, metric := range prometheusService.GetMetric() {
		var timestamp = time.Now()
		if metric.TimestampMs != nil {
			timestamp = time.Unix(*metric.TimestampMs, 0)
		}
		values := value(*prometheusService.Name, prometheusService.Type, metric)
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
			}

			for _, label := range metric.GetLabel() {
				switch *label.Name {
				case "unitType":
					metricBuilder.UnitType = *label.Value
				case "resource":
					*hostName = *label.Value
				case "warning":
					if value, err := strconv.Atoi(*label.Value); err == nil {
						metricBuilder.Warning = float64(value)
					} else {
						metricBuilder.Warning = -1
					}
				case "critical":
					if value, err := strconv.Atoi(*label.Value); err == nil {
						metricBuilder.Critical = float64(value)
					} else {
						metricBuilder.Warning = -1
					}
				case "group":
					*groups = append(*groups, *label.Value)
				}
			}
			metricBuilders = append(metricBuilders, metricBuilder)
		}
	}
	return metricBuilders
}

func constructResourceGroups(groupNames []string) []transit.ResourceGroup {
	groupNames = removeDuplicates(groupNames)
	var resourceGroups []transit.ResourceGroup
	for _, name := range groupNames {
		resourceGroups = append(resourceGroups, transit.ResourceGroup{
			GroupName: name,
			Type:      transit.HostGroup,
		})
	}
	return resourceGroups
}

func parsePrometheusServices(prometheusServices map[string]*dto.MetricFamily, hostName *string, groups *[]string) (*[]transit.MonitoredService, error) {
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
			Url:     "/receiver",
			Method:  "Post",
			Handler: receiverHandler,
		},
	)
}

func receiverHandler(c *gin.Context) {
	body, err := c.GetRawData()
	if err != nil {
		log.Error(err.Error())
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
