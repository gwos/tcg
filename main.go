package main

import (
	"encoding/json"
	"github.com/gwos/tng/milliseconds"
	"github.com/gwos/tng/transit"
	"math/rand"
	"time"
)

func main() {}

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
			"InceptionTime":   {TimeValue: &milliseconds.MillisecondTimestamp{Time: time.Now()}},
		},
		Metrics: []transit.TimeSeries{sampleValue},
	}

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
			"InceptionTime":   {TimeValue: &milliseconds.MillisecondTimestamp{Time: time.Now()}},
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
