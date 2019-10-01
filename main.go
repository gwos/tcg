package main

import (
	"./controller"
	"./transit"
	"flag"
	"fmt"
	"math/rand"
	"time"
)

// Flags
var argPort = flag.Int("port", 8080, "port to listen")

func valueOf(x int64) *int64 {
	return &x
}

// examples
func main2() {
	flag.Parse()
	fmt.Printf("Starting Groundwork Agent on port %d\n", *argPort)
	//localhost := transit.MonitoredResource{
	//	Name: "localhost",
	//	Type:   transit.HostResource,
	//	Status: transit.HOST_UP,
	//	Labels: map[string]string{"hostGroup": "egain-21", "appType": "nagios"},
	//}
	//serviceLocalLoad := transit.MonitoredResource /* instance state */ {
	//	Name: "local_load",
	//	Type: transit.ServiceResource,
	//	Status: transit.SERVICE_OK,
	//	Owner: &localhost,
	//	Labels: map[string]string{
	//		"appType":          "nagios",		// this is redundant, appTypes are all the same
	//		"device":           "127.0.0.1",
	//		"checkType":        "passive",
	//		"stateType":        "hard",
	//		"lastStateChange":  "10 mins ago",
	//		"lastCheckTime":    "10 mins ago",
	//		"lastPluginOutput": "foo | bar",
	//	},
	//}
	cores := int64(4)
	localLoadMetric1 := transit.Metric{
		Type: "local_load_1",
		Labels: map[string]transit.TypedValue{
			"cores":      transit.TypedValue{IntegerValue: cores},
			"sampleTime": transit.TypedValue{IntegerValue: 1}},
	}
	localLoadMetric5 := transit.Metric{
		Type: "local_load_5",
		Labels: map[string]transit.TypedValue{
			"cores":      transit.TypedValue{IntegerValue: cores},
			"sampleTime": transit.TypedValue{IntegerValue: 5}},
	}
	localLoadMetric15 := transit.Metric{
		Type: "local_load_15",
		Labels: map[string]transit.TypedValue{
			"cores":      transit.TypedValue{IntegerValue: cores},
			"sampleTime": transit.TypedValue{IntegerValue: 15}},
	}
	println(localLoadMetric1.Type)
	println(localLoadMetric5.Type)
	println(localLoadMetric15.Type)

	point := makePoint()
	sampleValue := transit.TimeSeries{
		MetricName: "local_load_5",
		SampleType: transit.Value,
		Tags: map[string]string{
			"deviceTag":     "127.0.0.1",
			"httpMethodTag": "POST",
			"httpStatusTag": "200",
		},
		Interval: point.Interval,
		Value:    point.Value,
	}
	point = makePoint()
	sampleCritical := transit.TimeSeries{
		MetricName: "local_load_5_cr",
		SampleType: transit.Critical,
		Tags: map[string]string{
			"deviceTag":     "127.0.0.1",
			"httpMethodTag": "POST",
			"httpStatusTag": "200",
		},
		Interval: point.Interval,
		Value:    point.Value,
	}
	point = makePoint()
	sampleWarning := transit.TimeSeries{
		MetricName: "local_load_5_wn",
		SampleType: transit.Warning,
		Tags: map[string]string{
			"deviceTag":     "127.0.0.1",
			"httpMethodTag": "POST",
			"httpStatusTag": "200",
		},
		Interval: point.Interval,
		Value:    point.Value,
	}

	// Connect with Transit...
	var transitServices, _ = transit.Connect(transit.Credentials{
		User:     "RESTAPIACCESS",
		Password: "! PASSWORD_HERE !",
	})
	// Send Metrics with Transit ...
	_, _ = transitServices.SendMetrics(&[]transit.TimeSeries{sampleValue, sampleWarning, sampleCritical})
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
	fmt.Println(*stats)
}

func makePoint() *transit.Point {
	random := rand.Float64()
	now := time.Now()
	return &transit.Point{
		Interval: &transit.TimeInterval{EndTime: now.String(), StartTime: now.String()},
		Value:    &transit.TypedValue{ValueType: transit.DoubleType, DoubleValue: random},
	}
}

func makePoints() []*transit.Point {
	points := make([]*transit.Point, 3)
	for i := range points {
		random := rand.Float64()
		now := time.Now()
		points[i] = &transit.Point{
			Interval: &transit.TimeInterval{EndTime: now.String(), StartTime: now.String()},
			Value:    &transit.TypedValue{DoubleValue: random},
		}
	}
	return points
}

type DummyMonitoredResource struct {
	Name   string
	Type   string
	Status string
	Labels map[string]string
}
