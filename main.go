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

func valueOf(x int8) *int8 {
	return &x
}

// examples
func main() {
	flag.Parse()
	fmt.Printf("Starting Groundwork Agent on port %d\n", *argPort)
	localhost := transit.MonitoredResource{
		Name: "localhost",
		Type:   transit.HostResource,
		Status: transit.HOST_UP,
		Labels: map[string]string{"hostGroup": "egain-21", "appType": "nagios"},
	}
	serviceLocalLoad := transit.MonitoredResource /* instance state */ {
		Name: "local_load",
		Type: transit.ServiceResource,
		Status: transit.SERVICE_OK,
		Owner: &localhost,
		Labels: map[string]string{
			"appType":          "nagios",		// this is redundant, appTypes are all the same
			"device":           "127.0.0.1",
			"checkType":        "passive",
			"stateType":        "hard",
			"lastStateChange":  "10 mins ago",
			"lastCheckTime":    "10 mins ago",
			"lastPluginOutput": "foo | bar",
		},
	}
	cores := int8(4)
	localLoadMetric1 := transit.Metric{
		Type:   "local_load_1",
		Labels: map[string]transit.TypedValue{
			"cores": transit.TypedValue{Int8Value: &cores},
			"sampleTime": transit.TypedValue{Int8Value: valueOf(1)}},
	}
	localLoadMetric5 := transit.Metric{
		Type:   "local_load_5",
		Labels: map[string]transit.TypedValue{
			"cores": transit.TypedValue{Int8Value: &cores},
			"sampleTime": transit.TypedValue{Int8Value: valueOf(5)}},
	}
	localLoadMetric15 := transit.Metric{
		Type:   "local_load_15",
		Labels: map[string]transit.TypedValue{
			"cores": transit.TypedValue{Int8Value: &cores},
			"sampleTime": transit.TypedValue{Int8Value: valueOf(15)}},
	}
	sample1 := transit.TimeSeries{
		Metric:     &localLoadMetric1,
		MetricKind: transit.GAUGE,
		Points: 	makePoints(),
		Resource:   &serviceLocalLoad,
		ValueType:  transit.DOUBLE,
	}
	sample5 := transit.TimeSeries{
		Metric:     &localLoadMetric5,
		MetricKind: transit.GAUGE,
		Points: 	makePoints(),
		Resource:   &serviceLocalLoad,
		ValueType:  transit.DOUBLE,
	}
	sample15 := transit.TimeSeries{
		Metric:     &localLoadMetric15,
		MetricKind: transit.GAUGE,
		Points: 	makePoints(),
		Resource:   &serviceLocalLoad,
		ValueType:  transit.DOUBLE,
	}
	// create a Groundwork Configuration
	config := transit.GroundworkConfig{
		HostName: "localhost",
		Account:  "RESTAPIACCESS",
		Token:    "63c5bt",
		SSL:      false,
	}
	// Connect with Transit...
	var transitServices = transit.Connect(config)
	// Send Metrics with Transit ...
	transitServices.SendMetrics(&[]transit.TimeSeries{sample1, sample5, sample15})
	// Retrieve Metrics List with Transit
	metrics, _ := transitServices.ListMetrics()
	for _, metric := range *metrics {
		// see transit.ListMetrics() for example creation of Metric definitions
		fmt.Println(metric)
	}
	// complete
	transit.Disconnect(&transitServices)

	// Controller Example
	var controllerServices = controller.CreateController()
	controllerServices.Start()
	controllerServices.Stop()
	controllerServices.Status()
	stats, _ := controllerServices.Stats()
	fmt.Println(*stats);
}

func makePoints()  []*transit.Point {
	points := make([]*transit.Point, 3)
	for i := range points {
		random := rand.Float64()
		//now := strconv.FormatInt(time.Now().Unix(), 10)
		now := time.Now()
		points[i] = &transit.Point{
			Interval: &transit.TimeInterval{EndTime: now, StartTime: now},
			Value:    &transit.TypedValue{DoubleValue: &random},
		}
	}
	return points;
}


