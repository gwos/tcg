package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/gwos/tng/milliseconds"
	serverConnector "github.com/gwos/tng/serverconnector"
	"github.com/gwos/tng/transit"
	"log"
	"math/rand"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

// Flags
var argPort = flag.Int("port", 8080, "port to listen")

func valueOf(x int64) *int64 {
	return &x
}

// examples
func main() {
	flag.Parse()
	fmt.Printf("Starting Groundwork Agent on port %d\n", *argPort)

	// VLAD - I think the gatherMetrics could be made into an advanced feature built into the ServerConnector:
	// 	1. From the PID, we can get the process name
	//  2. provide a list of process names that we want to monitor
	//  3. Look up the CPU usage for a process with a given name and turn it into a Service
	//  This way we can (a) get the status of processes running on a server (b) get their cpu usage with thresholds
	//  After finishing serverConnector.CollectMetrics(), please implement this in the ServerConnector
	processes := gatherMetrics()

	for _, p := range processes {
		log.Println("Process ", p.pid, " takes ", p.cpu, " % of the CPU")
	}

	server := serverConnector.CollectMetrics()
	println(server.Name)

	// TODO: start TNG
	sendInventoryResources()
	sendMonitoredResources()
	// TODO: stop TNG:
}

func sendInventoryResources() {
	// Example Service
	localLoadService := transit.InventoryService{
		Name: "local_load",
	}

	// Example Monitored Resource of type Host
	geneva := transit.InventoryResource{
		Name:     "geneva",
		Type:     transit.Host,
		Services: []transit.InventoryService{localLoadService},
	}
	// Build Inventory
	inventory := []transit.InventoryResource{geneva}

	// TODO: call into API

	b, err := json.Marshal(inventory)
	if err == nil {
		s := string(b)
		println(s)
	}

}

func sendMonitoredResources() {
	// Create a Metrics Sample
	metricSample := makeMetricSample()
	sampleValue := transit.TimeSeries{
		MetricName: "local_load_5",
		// Labels:      []*LabelDescriptor{&cores, &sampleTime},
		MetricSamples: []*transit.MetricSample{
			{
				SampleType: transit.Value,
				Interval:   metricSample.Interval,
				Value:      metricSample.Value,
			},
		},
		Tags: map[string]string{
			"deviceTag":     "127.0.0.1",
			"httpMethodTag": "POST",
			"httpStatusTag": "200",
		},
		Unit: "%{cpu}",
	}
	metricSample = makeMetricSample()
	sampleCritical := transit.TimeSeries{
		MetricName: "local_load_5_cr",
		MetricSamples: []*transit.MetricSample{
			{
				SampleType: transit.Critical,
				Interval:   metricSample.Interval,
				Value:      metricSample.Value,
			},
		},
		Tags: map[string]string{
			"deviceTag":     "127.0.0.1",
			"httpMethodTag": "POST",
			"httpStatusTag": "200",
		},
		Unit: "%{cpu}",
	}
	metricSample = makeMetricSample()
	sampleWarning := transit.TimeSeries{
		MetricName: "local_load_5_wn",
		MetricSamples: []*transit.MetricSample{
			{
				SampleType: transit.Warning,
				Interval:   metricSample.Interval,
				Value:      metricSample.Value,
			},
		},
		Tags: map[string]string{
			"deviceTag":     "127.0.0.1",
			"httpMethodTag": "POST",
			"httpStatusTag": "200",
		},
		Unit: "%{cpu}",
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
		Metrics: []transit.TimeSeries{sampleValue, sampleWarning, sampleCritical},
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

func makeMetricSample() *transit.MetricSample {
	random := rand.Float64() * 100.0
	now := milliseconds.MillisecondTimestamp{Time: time.Now()}
	return &transit.MetricSample{
		SampleType: transit.Value,
		Interval:   &transit.TimeInterval{EndTime: now, StartTime: now},
		Value:      &transit.TypedValue{ValueType: transit.DoubleType, DoubleValue: random},
	}
}

func makeMetricSamples() []*transit.MetricSample {
	metricSamples := make([]*transit.MetricSample, 3)
	for i := range metricSamples {
		random := rand.Float64() * 100.0
		now := milliseconds.MillisecondTimestamp{Time: time.Now()}
		metricSamples[i] = &transit.MetricSample{
			SampleType: transit.Value,
			Interval:   &transit.TimeInterval{EndTime: now, StartTime: now},
			Value:      &transit.TypedValue{DoubleValue: random},
		}
	}
	return metricSamples
}

// Use this if we can't find something better

type Process struct {
	pid int
	cpu float64
}

func gatherMetrics() []*Process{
	cmd := exec.Command("ps", "aux")
	var out bytes.Buffer
	cmd.Stdout = &out
	err := cmd.Run()
	if err != nil {
		log.Fatal(err)
	}
	processes := make([]*Process, 0)
	for {
		line, err := out.ReadString('\n')
		if err != nil {
			break
		}
		tokens := strings.Split(line, " ")
		ft := make([]string, 0)
		for _, t := range tokens {
			if t != "" && t != "\t" {
				ft = append(ft, t)
			}
		}
		pid, err := strconv.Atoi(ft[1])
		if err != nil {
			continue
		}
		cpu, err := strconv.ParseFloat(ft[2], 64)
		if err != nil {
			log.Fatal(err)
		}
		processes = append(processes, &Process{pid, cpu})
	}

	return processes
}
