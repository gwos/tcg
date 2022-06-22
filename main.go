package main

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"time"

	"github.com/gwos/tcg/connectors"
	"github.com/gwos/tcg/sdk/transit"
)

//////////////////////////////////////////////////////////
//         Examples of High Level API functions         //
//////////////////////////////////////////////////////////

const (
	Resource1                 = "tcg-host-1"
	Resource2                 = "tcg-host-1"
	Service1                  = "tcg-server-1"
	Service2                  = "tcg-server-2"
	CPUMetric                 = "tcg-cpu"
	CPUMetricWarning          = "tcg-cpu-warning"
	CPUMetricCritical         = "tcg-cpu-critical"
	PercentFreeMetric         = "tcg-percent-free"
	PercentFreeMetricWarning  = "tcg-percent-free-warning"
	PercentFreeMetricCritical = "tcg-percent-free-critical"
	DiskUsed                  = "tcg-memory-used"
)

func main() {
	//////////////////////////////////////////////////////////////////////////////////////////
	//                                Inventory Examples                                    //
	//////////////////////////////////////////////////////////////////////////////////////////
	var iServices []transit.InventoryService
	is1 := connectors.CreateInventoryService(Service1, Resource1)
	is2 := connectors.CreateInventoryService(Service2, Resource1)
	iServices = append(iServices, is1, is2)
	iResource1 := connectors.CreateInventoryResource(Resource1, iServices)
	println(iResource1.Services[0].Description)
	println(iResource1.Description)
	//if (enableTransit) {
	//	connectors.SendInventory()
	//}

	//////////////////////////////////////////////////////////////////////////////////////////
	//                                  Metrics Examples                                    //
	//////////////////////////////////////////////////////////////////////////////////////////
	// Create Integer Metric with Thresholds
	metric1, _ := connectors.CreateMetric(CPUMetric, 75)
	warning1, _ := connectors.CreateWarningThreshold(CPUMetricWarning, 60)
	critical1, _ := connectors.CreateCriticalThreshold(CPUMetricCritical, 90)
	metric1.Thresholds = []transit.ThresholdValue{*warning1, *critical1}
	// Create Double Metric with Thresholds and Unit Type
	metric2, _ := connectors.CreateMetric(PercentFreeMetric, 99.82, transit.PercentCPU)
	warning2, _ := connectors.CreateWarningThreshold(PercentFreeMetricWarning, 80.5)
	critical2, _ := connectors.CreateCriticalThreshold(PercentFreeMetricCritical, 90.8)
	metric2.Thresholds = []transit.ThresholdValue{*warning2, *critical2}
	// create with interval
	timestamp := transit.NewTimestamp()
	interval := &transit.TimeInterval{
		EndTime:   timestamp,
		StartTime: timestamp,
	}
	metric3, _ := connectors.CreateMetric(DiskUsed, 65.82, transit.GB, interval)
	// add tags
	metric3.SetTag("myTag1", "myTagValue1")
	metric3.SetTag("myTag2", "myTagValue2")

	// display ...
	fmt.Printf("metric 1 created with thresholds: %+v\n", metric1)
	fmt.Printf("metric 2 created with thresholds: %+v\n", metric2)
	fmt.Printf("metric 3 created with thresholds: %+v\n", metric3)

	// Create a Service and add metrics ...
	service1, _ := connectors.CreateService(Service1, Resource1, []transit.TimeSeries{*metric1, *metric2, *metric3})
	fmt.Printf("service 1 created with metrics: %+v\n", service1)
	service2, _ := connectors.CreateService(Service2, Resource2, []transit.TimeSeries{*metric1})
	fmt.Printf("service 1 created with metrics: %+v\n", service2)

	// Create a Monitored Resource and Add Service
	resource1, _ := connectors.CreateResource(Resource1, []transit.MonitoredService{*service1})
	fmt.Printf("resource 1 created with services: %+v\n", resource1)
	resource2, _ := connectors.CreateResource(Resource2, []transit.MonitoredService{*service2})
	fmt.Printf("resource 2 created with services: %+v\n", resource2)
}

//////////////////////////////////////////////////////////
//         Examples of Low Level API functions          //
//////////////////////////////////////////////////////////

func LowLevelExamples() { //nolint
	warningThreshold := transit.ThresholdValue{
		SampleType: transit.Warning,
		Label:      "local_load_5_wn",
		Value:      transit.NewTypedValue(70.0),
	}
	errorThreshold := transit.ThresholdValue{
		SampleType: transit.Critical,
		Label:      "local_load_5_cr",
		Value:      transit.NewTypedValue(85.0),
	}
	random := rand.Float64() * 100.0
	lastCheckTime := *transit.NewTimestamp()
	nextCheckTime := lastCheckTime.Add(time.Minute * 5)
	sampleValue := transit.TimeSeries{
		MetricName: "local_load_5",
		SampleType: transit.Value,
		Interval:   &transit.TimeInterval{EndTime: &lastCheckTime, StartTime: &lastCheckTime},
		Value:      transit.NewTypedValue(random),
		Thresholds: []transit.ThresholdValue{warningThreshold, errorThreshold},
		Unit:       "%{cpu}",
	}

	// Example Service
	var localLoadService = transit.MonitoredService{
		BaseInfo: transit.BaseInfo{
			Name: "local_load",
			Type: transit.ResourceTypeService,
			Properties: map[string]transit.TypedValue{
				"stateType":       *transit.NewTypedValue("SOFT"),
				"checkType":       *transit.NewTypedValue("ACTIVE"),
				"PerformanceData": *transit.NewTypedValue("007-321 RAD"),
				"ExecutionTime":   *transit.NewTypedValue(3.0),
				"CurrentAttempt":  *transit.NewTypedValue(2),
				"InceptionTime":   *transit.NewTypedValue(lastCheckTime),
			},
		},
		MonitoredInfo: transit.MonitoredInfo{
			Status:           transit.ServiceOk,
			LastCheckTime:    &lastCheckTime,
			NextCheckTime:    &nextCheckTime,
			LastPluginOutput: "foo | bar",
		}, Metrics: []transit.TimeSeries{sampleValue},
	}

	geneva := transit.MonitoredResource{
		BaseResource: transit.BaseResource{
			BaseInfo: transit.BaseInfo{
				Name: "geneva",
				Type: transit.ResourceTypeHost,
				Properties: map[string]transit.TypedValue{
					"stateType":       *transit.NewTypedValue("SOFT"),
					"checkType":       *transit.NewTypedValue("ACTIVE"),
					"PerformanceData": *transit.NewTypedValue("007-321 RAD"),
					"ExecutionTime":   *transit.NewTypedValue(3.0),
					"CurrentAttempt":  *transit.NewTypedValue(2),
					"InceptionTime":   *transit.NewTypedValue(lastCheckTime),
				},
			},
		},
		MonitoredInfo: transit.MonitoredInfo{
			Status:           transit.HostUp,
			LastCheckTime:    &lastCheckTime,
			NextCheckTime:    &nextCheckTime,
			LastPluginOutput: "44/55/888 QA00005-BC",
		}, Services: []transit.MonitoredService{localLoadService},
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
