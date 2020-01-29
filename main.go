package main

import (
	"encoding/json"
	"fmt"
	_ "github.com/gwos/tng/docs"
	"github.com/gwos/tng/log"
	"github.com/gwos/tng/milliseconds"
	"github.com/gwos/tng/serverconnector"
	"github.com/gwos/tng/services"
	"github.com/gwos/tng/transit"
	"math/rand"
	"os/exec"
	"time"
)

var transitService = services.GetTransitService()

// @title TNG API Documentation
// @version 1.0

// @host localhost:8099
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
		serverconnector.LastCheck = milliseconds.MillisecondTimestamp{Time: time.Now()}
		time.Sleep(20 * time.Second)
	}
}

func sendInventoryResources(resource transit.InventoryResource) error {

	monitoredResourceRef := transit.MonitoredResourceRef{
		Name: resource.Name,
		Type: transit.Host,
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
			TimeStamp:  milliseconds.MillisecondTimestamp{Time: time.Now()},
			Version:    transit.TransitModelVersion,
		},
		Resources: []transit.InventoryResource{resource},
		Groups: []transit.ResourceGroup{
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
			AppType:    "VEMA", // TODO: need an appType for ServerConnector, Elastic
			AgentID:    "3939333393342",
			TraceToken: "token-99e93",
			TimeStamp:  milliseconds.MillisecondTimestamp{Time: time.Now()},
			Version:    transit.TransitModelVersion,
		},
		Resources: []transit.MonitoredResource{resource},
	}
	// Test a Time type sample
	timeSample := transit.TimeSeries{
		MetricName: "timeSample",
		SampleType: transit.Value,
		Interval: &transit.TimeInterval{
			EndTime:   milliseconds.MillisecondTimestamp{Time: time.Now()},
			StartTime: milliseconds.MillisecondTimestamp{Time: time.Now()},
		},
		Value: &transit.TypedValue{
			ValueType: transit.TimeType,
			TimeValue: milliseconds.MillisecondTimestamp{Time: time.Now()},
		},
	}
	timeSampleService := transit.MonitoredService{
		Name:             "timeSample",
		Type:             transit.Service,
		Status:           transit.ServiceOk,
		LastCheckTime:    milliseconds.MillisecondTimestamp{Time: time.Now()},
		NextCheckTime:    milliseconds.MillisecondTimestamp{Time: time.Now()},
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

//////////////////////////////////////////////////////////

func example() {
	warningThreshold := transit.ThresholdValue{
		SampleType: transit.Warning,
		Label:      "local_load_5_wn",
		Value:      &transit.TypedValue{ValueType: transit.DoubleType, DoubleValue: 70.0}}
	errorThreshold := transit.ThresholdValue{
		SampleType: transit.Critical,
		Label:      "local_load_5_cr",
		Value:      &transit.TypedValue{ValueType: transit.DoubleType, DoubleValue: 85.0}}
	random := rand.Float64() * 100.0
	now := milliseconds.MillisecondTimestamp{Time: time.Now()}
	sampleValue := transit.TimeSeries{
		MetricName: "local_load_5",
		SampleType: transit.Value,
		Interval:   &transit.TimeInterval{EndTime: now, StartTime: now},
		Value:      &transit.TypedValue{ValueType: transit.DoubleType, DoubleValue: random},
		Thresholds: &[]transit.ThresholdValue{warningThreshold, errorThreshold},
		Unit:       "%{cpu}",
	}

	// Example Service
	var localLoadService = transit.MonitoredService{
		Name:             "local_load",
		Type:             transit.Service,
		Status:           transit.ServiceOk,
		LastCheckTime:    milliseconds.MillisecondTimestamp{Time: time.Now()},
		NextCheckTime:    milliseconds.MillisecondTimestamp{Time: time.Now().Add(time.Minute * 5)},
		LastPlugInOutput: "foo | bar",
		Properties: map[string]transit.TypedValue{
			"stateType":       {StringValue: "SOFT"},
			"checkType":       {StringValue: "ACTIVE"},
			"PerformanceData": {StringValue: "007-321 RAD"},
			"ExecutionTime":   {DoubleValue: 3.0},
			"CurrentAttempt":  {IntegerValue: 2},
			"InceptionTime":   {TimeValue: milliseconds.MillisecondTimestamp{Time: time.Now()}},
		},
		Metrics: []transit.TimeSeries{sampleValue},
	} // Example Monitored Resource of type Host

	geneva := transit.MonitoredResource{
		Name:             "geneva",
		Type:             transit.Host,
		Status:           transit.HostUp,
		LastCheckTime:    milliseconds.MillisecondTimestamp{Time: time.Now()},
		NextCheckTime:    milliseconds.MillisecondTimestamp{Time: time.Now().Add(time.Minute * 5)},
		LastPlugInOutput: "44/55/888 QA00005-BC",
		Properties: map[string]transit.TypedValue{
			"stateType":       {StringValue: "SOFT"},
			"checkType":       {StringValue: "ACTIVE"},
			"PerformanceData": {StringValue: "007-321 RAD"},
			"ExecutionTime":   {DoubleValue: 3.0},
			"CurrentAttempt":  {IntegerValue: 2},
			"InceptionTime":   {TimeValue: milliseconds.MillisecondTimestamp{Time: time.Now()}},
		},
		Services: []transit.MonitoredService{localLoadService},
	}

	// Build Monitored Resources
	resources := []transit.MonitoredResource{geneva}

	// TODO: call into API

	b, err := json.Marshal(resources)
	if err == nil {
		s := string(b)
		println(s)
	}
}
