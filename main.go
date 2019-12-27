package main

import (
	"encoding/json"
	"fmt"
	_ "github.com/gwos/tng/docs"
	"github.com/gwos/tng/log"
	"github.com/gwos/tng/serverconnector"
	"github.com/gwos/tng/services"
	"github.com/gwos/tng/subseconds"
	"github.com/gwos/tng/transit"
	"os/exec"
	"time"
)

var transitService = services.GetTransitService()

// @title TNG API Documentation
// @version 1.0

// @host localhost:8081
// @BasePath /api/v1
func main() {
	err := transitService.StartNats()
	if err != nil {
		log.Error(err.Error())
		return
	}
	defer func() {
		err = transitService.StopNats()
		if err != nil {
			log.Error(err.Error())
		}
		cmd := exec.Command("rm", "-rf", "src")
		_, err = cmd.Output()
		if err != nil {
			log.Error(err.Error())
		}
	}()

	err = transitService.StartTransport()
	if err != nil {
		fmt.Printf("%s", err.Error())
		return
	}

	defer func() {
		err = transitService.StopTransport()
		if err != nil {
			fmt.Printf("%s", err.Error())
		}
	}()

	err = transitService.StartController()
	if err != nil {
		fmt.Printf("%s", err.Error())
		return
	}

	defer func() {
		err = transitService.StopController()
		if err != nil {
			fmt.Printf("%s", err.Error())
		}
	}()

	err = sendInventoryResources(*serverconnector.Synchronize())

	for {
		err := sendMonitoredResources(*serverconnector.CollectMetrics())
		if err != nil {
			log.Error(err.Error())
		}
		serverconnector.LastCheck = subseconds.MillisecondTimestamp{Time: time.Now()}
		time.Sleep(20 * time.Second)
	}
}

func sendInventoryResources(resource transit.InventoryResource) error {

	monitoredResourceRef := transit.MonitoredResourceRef{
		Name:  resource.Name,
		Type:  transit.Host,
	}

	resourceGroup := transit.ResourceGroup{
		GroupName: "LocalServer",
		Type:      transit.HostGroup,
		Resources: []transit.MonitoredResourceRef{monitoredResourceRef},
	}
	inventoryRequest := transit.InventoryRequest{
		Context: transit.TracerContext{
			AppType:    "VEMA",
			AgentID:    "3939333393342",
			TraceToken: "token-99e93",
			TimeStamp:  subseconds.MillisecondTimestamp{Time: time.Now()},
		},
		Resources: []transit.InventoryResource{resource},
		Groups:    []transit.ResourceGroup{
				resourceGroup,
		},
	}

	b, err := json.Marshal(inventoryRequest)
	if err != nil {
		return err
	}

	err = transitService.SynchronizeInventory(b)

	return err
}

func sendMonitoredResources(resource transit.MonitoredResource) error {
	request := transit.ResourcesWithServicesRequest{
		Context: transit.TracerContext{
			AppType:    "VEMA",				// TODO: need an appType for ServerConnector, Elastic
			AgentID:    "3939333393342",
			TraceToken: "token-99e93",
			TimeStamp:  subseconds.MillisecondTimestamp{Time: time.Now()},
		},
		Resources: []transit.MonitoredResource{resource},
	}
	// Test a Time type sample
	timeSample := transit.TimeSeries{
		MetricName: "timeSample",
		SampleType: transit.Value,
		Interval: &transit.TimeInterval{
			EndTime:   subseconds.MillisecondTimestamp{Time: time.Now()},
			StartTime: subseconds.MillisecondTimestamp{Time: time.Now()},
		},
		Value: &transit.TypedValue{
			ValueType:    transit.TimeType,
			TimeValue:    subseconds.MillisecondTimestamp{Time: time.Now()},
		},
	}
	timeSampleService := transit.MonitoredService{
		Name:             "timeSample",
		Type:             transit.Service,
		Status:           transit.ServiceOk,
		LastCheckTime:    subseconds.MillisecondTimestamp{Time: time.Now()},
		NextCheckTime:    subseconds.MillisecondTimestamp{Time: time.Now()},
		LastPlugInOutput: "test",
		Owner:            request.Resources[0].Name, // set host
		Metrics:          []transit.TimeSeries{timeSample},
	}
	request.Resources[0].Services = append(request.Resources[0].Services, timeSampleService)
	b, err := json.Marshal(request)
	if err != nil {
		return err
	}

	return transitService.SendResourceWithMetrics(b)
}