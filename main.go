package main

import (
	"flag"
	"fmt"
	"github.com/gwos/tng/controller"
	"github.com/gwos/tng/transit"
	"math/rand"
	"time"
)

// Flags
var argPort =  flag.Int("port", 8080, "port to listen")

func valueOf(x int64) *int64 {
	return &x
}

// examples
func main2() {
	flag.Parse()
	fmt.Printf("Starting Groundwork Agent on port %d\n", *argPort)

	// Example Usage with a host
	geneva := transit.MonitoredResource{
		Name: "geneva",
		Type:   transit.HostResource,
		Status: transit.HOST_UP,
		LastCheckTime: time.Now(),
		NextCheckTime: time.Now().Add(time.Minute * 5),
		LastPlugInOutput: "44/55/888 QA00005-BC",
		Description: "Subversion Server",
		Properties: map[string]transit.TypedValue{
			"stateType":       transit.TypedValue{StringValue: "SOFT"},
			"checkType":       transit.TypedValue{StringValue: "ACTIVE"},
			"PerformanceData": transit.TypedValue{StringValue: "007-321 RAD"},
			"ExecutionTime":   transit.TypedValue{DoubleValue: 3.0},
			"CurrentAttempt":  transit.TypedValue{IntegerValue: 2},
			"InceptionDate":   transit.TypedValue{DateValue: time.Now()},
		},
	}
	localLoadService := transit.MonitoredResource{
		Name: "local_load",
		Type:   transit.ServiceResource,
		Status: transit.SERVICE_OK,
		LastCheckTime: time.Now(),
		NextCheckTime: time.Now().Add(time.Minute * 5),
		LastPlugInOutput: "foo | bar",
		Description: "Load on subversion",
		Properties: map[string]transit.TypedValue{
			"stateType":       transit.TypedValue{StringValue: "SOFT"},
			"checkType":       transit.TypedValue{StringValue: "ACTIVE"},
			"PerformanceData": transit.TypedValue{StringValue: "007-321 RAD"},
			"ExecutionTime":   transit.TypedValue{DoubleValue: 3.0},
			"CurrentAttempt":  transit.TypedValue{IntegerValue: 2},
			"InceptionDate":   transit.TypedValue{DateValue: time.Now()},
		},
	}
	point := makePoint()
	sampleValue := transit.TimeSeries{
		MetricName:   "local_load_5",
		SampleType:	transit.Value,
		Tags: map[string]string{
			"deviceTag":     "127.0.0.1",
			"httpMethodTag": "POST",
			"httpStatusTag": "200",
		},
		Interval: point.Interval,
		Value: point.Value,
		Unit: "load",
	}
	point = makePoint()
	sampleCritical := transit.TimeSeries{
		MetricName:   "local_load_5_cr",
		SampleType:	transit.Critical,
		Tags: map[string]string{
			"deviceTag":     "127.0.0.1",
			"httpMethodTag": "POST",
			"httpStatusTag": "200",
		},
		Interval: point.Interval,
		Value: point.Value,
		Unit: "load",
	}
	point = makePoint()
	sampleWarning := transit.TimeSeries{
		MetricName:   "local_load_5_wn",
		SampleType:	transit.Warning,
		Tags: map[string]string{
			"deviceTag":     "127.0.0.1",
			"httpMethodTag": "POST",
			"httpStatusTag": "200",
		},
		Interval: point.Interval,
		Value: point.Value,
		Unit: "load",
	}

	// Build Payload
	resources := []transit.ResourceWithMetrics{
		{Resource: geneva, Metrics: make([]transit.TimeSeries, 0)},
		{Resource: localLoadService, Metrics: []transit.TimeSeries{sampleValue, sampleCritical, sampleWarning}},
	}


	// create a Groundwork Configuration
	//config := transit.GroundworkConfig{
	//	HostName: "localhost",
	//	Account:  "RESTAPIACCESS",
	//	Token:    "63c5bt",
	//	SSL:      false,
	//}
	// Connect with Transit...
	var transitServices,_ = transit.Connect(transit.Credentials{
		User:     "test",
		Password: "test",
	})

	// Send Metrics with Transit ...
	transitServices.SendResourcesWithMetrics(resources)

	// Retrieve Metrics List with Transit
	metrics, _ := transitServices.ListMetrics()
	for _, metric := range *metrics {
		// see transit.ListMetrics() for example creation of Metric definitions
		fmt.Println(metric)
	}
	// complete
	transit.Disconnect(transitServices)

	// Controller Example
	var controllerServices = controller.CreateController()
	controllerServices.Start()
	controllerServices.Stop()
	controllerServices.Status()
	stats, _ := controllerServices.Stats()
	fmt.Println(*stats);
}

func makePoint()  *transit.Point {
	random := rand.Float64()
	now := time.Now()
	return &transit.Point{
		Interval: &transit.TimeInterval{EndTime: now, StartTime: now},
		Value:    &transit.TypedValue{ValueType: transit.DoubleType, DoubleValue: random},
	}
}

func makePoints()  []*transit.Point {
	points := make([]*transit.Point, 3)
	for i := range points {
		random := rand.Float64()
		//now := strconv.FormatInt(time.Now().Unix(), 10)
		now := time.Now()
		points[i] = &transit.Point{
			Interval: &transit.TimeInterval{EndTime: now, StartTime: now},
			Value:    &transit.TypedValue{DoubleValue: random},
		}
	}
	return points;
}


