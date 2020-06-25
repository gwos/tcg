package main

import (
	"bytes"
	"errors"
	"github.com/gin-gonic/gin"
	"github.com/gwos/tcg/connectors"
	"github.com/gwos/tcg/log"
	"github.com/gwos/tcg/milliseconds"
	"github.com/gwos/tcg/services"
	"github.com/gwos/tcg/transit"
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

	var monitoredServices []transit.MonitoredService
	var groups []string
	var hostName = ""

	for _, prometheusService := range prometheusServices {
		if len(prometheusService.GetMetric()) == 0 {
			continue
		}

		metricBuilder := connectors.MetricBuilder{
			Name:           *prometheusService.Name,
			Value:          *prometheusService.Metric[0].Untyped.Value,
			StartTimestamp: &milliseconds.MillisecondTimestamp{Time: time.Unix(*prometheusService.GetMetric()[0].TimestampMs, 0)},
			EndTimestamp:   &milliseconds.MillisecondTimestamp{Time: time.Unix(*prometheusService.GetMetric()[0].TimestampMs, 0)},
		}

		for _, label := range prometheusService.GetMetric()[0].GetLabel() {
			switch *label.Name {
			case "unitType":
				metricBuilder.UnitType = *label.Value
			case "resource":
				hostName = *label.Value
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
				groups = append(groups, *label.Value)
			}
		}

		if hostName == "" {
			return nil, nil, errors.New("HostName cannot be empty")
		}
		service, err := connectors.BuildServiceForMetric(hostName, metricBuilder)
		if err != nil {
			return nil, nil, err
		}
		monitoredServices = append(monitoredServices, *service)
	}

	groups = removeDuplicates(groups)
	resourceGroups := []transit.ResourceGroup{}
	for _, name := range groups {
		resourceGroups = append(resourceGroups, transit.ResourceGroup{
			GroupName: name,
			Type:      transit.HostGroup,
		})
	}

	monitoredResource, err := connectors.CreateResource(hostName)
	if err != nil {
		return nil, nil, err
	}

	monitoredResource.Services = monitoredServices

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
	monitoredResource, groups, err := parsePrometheusBody(body)
	if err != nil {
		log.Error(err.Error())
		c.JSON(http.StatusBadRequest, err.Error())
		return
	}
	inventory := buildInventory(monitoredResource)
	if validateInventory(inventory) {
		err := connectors.SendMetrics([]transit.MonitoredResource{*monitoredResource})
		if err != nil {
			log.Error(err.Error())
		}
	} else {
		err := connectors.SendInventory(*inventory, *groups, transit.Yield)
		if err != nil {
			log.Error(err.Error())
		}
		time.Sleep(2 * time.Second)
		err = connectors.SendMetrics([]transit.MonitoredResource{*monitoredResource})
		if err != nil {
			log.Error(err.Error())
		}
	}
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
