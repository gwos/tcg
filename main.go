package main

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"time"

	"github.com/gwos/tcg/connectors"
	"github.com/gwos/tcg/milliseconds"
	"github.com/gwos/tcg/transit"
)

//////////////////////////////////////////////////////////
//         Examples of High Level API functions         //
//////////////////////////////////////////////////////////

var enableTransit = false

const (
	Resource1                 = "tcg-host-1"
	Resource2                 = "tcg-host-1"
	Service1                  = "tcg-server-1"
	Service2                  = "tcg-server-2"
	CpuMetric                 = "tcg-cpu"
	CpuMetricWarning          = "tcg-cpu-warning"
	CpuMetricCritical         = "tcg-cpu-critical"
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
	metric1, _ := connectors.CreateMetric(CpuMetric, 75)
	warning1, _ := connectors.CreateWarningThreshold(CpuMetricWarning, 60)
	critical1, _ := connectors.CreateCriticalThreshold(CpuMetricCritical, 90)
	metric1.Thresholds = &[]transit.ThresholdValue{*warning1, *critical1}
	// Create Double Metric with Thresholds and Unit Type
	metric2, _ := connectors.CreateMetric(PercentFreeMetric, 99.82, transit.PercentCPU)
	warning2, _ := connectors.CreateWarningThreshold(PercentFreeMetricWarning, 80.5)
	critical2, _ := connectors.CreateCriticalThreshold(PercentFreeMetricCritical, 90.8)
	metric2.Thresholds = &[]transit.ThresholdValue{*warning2, *critical2}
	// create with interval
	now := time.Now()
	interval := &transit.TimeInterval{
		EndTime:   milliseconds.MillisecondTimestamp{Time: now},
		StartTime: milliseconds.MillisecondTimestamp{Time: now},
	}
	metric3, _ := connectors.CreateMetric(DiskUsed, 65.82, transit.GB, interval)
	// add tags
	metric3.CreateTag("myTag1", "myTagValue1")
	metric3.CreateTag("myTag2", "myTagValue2")

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

	if enableTransit {
		connectors.SendMetrics(context.Background(), []transit.MonitoredResource{*resource1, *resource2})
	}
}

//////////////////////////////////////////////////////////
//         Examples of Low Level API functions          //
//////////////////////////////////////////////////////////

func LowLevelExamples() {
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
		BaseTransitData: transit.BaseTransitData{
			Name: "local_load",
			Type: transit.Service,
			Properties: map[string]transit.TypedValue{
				"stateType":       {StringValue: "SOFT"},
				"checkType":       {StringValue: "ACTIVE"},
				"PerformanceData": {StringValue: "007-321 RAD"},
				"ExecutionTime":   {DoubleValue: 3.0},
				"CurrentAttempt":  {IntegerValue: 2},
				"InceptionTime":   {TimeValue: &milliseconds.MillisecondTimestamp{Time: time.Now()}},
			},
		},
		Status:           transit.ServiceOk,
		LastCheckTime:    milliseconds.MillisecondTimestamp{Time: time.Now()},
		NextCheckTime:    milliseconds.MillisecondTimestamp{Time: time.Now().Add(time.Minute * 5)},
		LastPlugInOutput: "foo | bar",
		Metrics:          []transit.TimeSeries{sampleValue},
	}

	geneva := transit.MonitoredResource{
		BaseResource: transit.BaseResource{
			BaseTransitData: transit.BaseTransitData{
				Name: "geneva",
				Type: transit.Host,
				Properties: map[string]transit.TypedValue{
					"stateType":       {StringValue: "SOFT"},
					"checkType":       {StringValue: "ACTIVE"},
					"PerformanceData": {StringValue: "007-321 RAD"},
					"ExecutionTime":   {DoubleValue: 3.0},
					"CurrentAttempt":  {IntegerValue: 2},
					"InceptionTime":   {TimeValue: &milliseconds.MillisecondTimestamp{Time: time.Now()}},
				},
			},
		},
		Status:           transit.HostUp,
		LastCheckTime:    milliseconds.MillisecondTimestamp{Time: time.Now()},
		NextCheckTime:    milliseconds.MillisecondTimestamp{Time: time.Now().Add(time.Minute * 5)},
		LastPlugInOutput: "44/55/888 QA00005-BC",
		Services:         []transit.MonitoredService{localLoadService},
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
