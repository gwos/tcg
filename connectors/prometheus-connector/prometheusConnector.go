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
var inventory transit.InventoryResource

var parser expfmt.TextParser

func Synchronize() *transit.InventoryResource {
	return &inventory
}

func parsePrometheusBody(body []byte) (*transit.MonitoredResource, error) {
	prometheusServices, err := parser.TextToMetricFamilies(strings.NewReader(string(body)))
	if err != nil {
		return nil, err
	}

	var monitoredServices []transit.MonitoredService
	var hostName string

	for _, prometheusService := range prometheusServices {
		if len(prometheusService.GetMetric()) == 0 {
			continue
		}
		hostName = ""

		metricBuilder := connectors.MetricBuilder{
			Name:           *prometheusService.Name,
			UnitType:       transit.MB, // TODO: discuss UnitType - I can add all fields that I need
			Value:          *prometheusService.Metric[0].Untyped.Value,
			StartTimestamp: &milliseconds.MillisecondTimestamp{Time: time.Unix(*prometheusService.GetMetric()[0].TimestampMs, 0)},
			EndTimestamp:   &milliseconds.MillisecondTimestamp{Time: time.Unix(*prometheusService.GetMetric()[0].TimestampMs, 0)},
		}

		for _, label := range prometheusService.GetMetric()[0].GetLabel() {
			switch *label.Name {
			case "resource":
				hostName = *label.Value
			case "warning":
				if i, err := strconv.Atoi(*label.Value); err == nil {
					metricBuilder.Warning = i
				} else {
					metricBuilder.Warning = -1
				}
			case "critical":
				if i, err := strconv.Atoi(*label.Value); err == nil {
					metricBuilder.Critical = i
				} else {
					metricBuilder.Warning = -1
				}
			}
		}

		if hostName == "" {
			return nil, errors.New("HostName cannot be empty")
		}
		service, err := connectors.BuildServiceForMetric(hostName, metricBuilder)
		if err != nil {
			return nil, err
		}
		monitoredServices = append(monitoredServices, *service)
	}

	monitoredResource, err := connectors.CreateResource(hostName)
	if err != nil {
		return nil, err
	}

	monitoredResource.Services = monitoredServices

	return monitoredResource, nil
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

	inventoryResource := connectors.CreateInventoryResource(resource.Owner, inventoryServices)
	inventory = inventoryResource
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
	monitoredResource, err := parsePrometheusBody(body)
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
		err := connectors.SendInventory(*inventory, []transit.ResourceGroup{}, "")
		if err != nil {
			log.Error(err.Error())
		}
		// TODO: ensure sending metrics after inventory processed
		err = connectors.SendMetrics([]transit.MonitoredResource{*monitoredResource})
		if err != nil {
			log.Error(err.Error())
		}
	}
}
