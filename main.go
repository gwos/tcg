package main

import (
	"bytes"
	"flag"
	"fmt"
	"github.com/gwos/tng/controller"
	"github.com/gwos/tng/transit"
	"github.com/stealthly/go-avro"
	"math/rand"
	"os"
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
	AvroPlayground()
	if (1 == 1) {
		os.Exit(3)
	}
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
	cores := int64(4)
	localLoadMetric1 := transit.Metric{
		Type:   "local_load_1",
		Labels: map[string]transit.TypedValue{
			"cores": transit.TypedValue{IntegerValue: &cores},
			"sampleTime": transit.TypedValue{IntegerValue: valueOf(1)}},
	}
	localLoadMetric5 := transit.Metric{
		Type:   "local_load_5",
		Labels: map[string]transit.TypedValue{
			"cores": transit.TypedValue{IntegerValue: &cores},
			"sampleTime": transit.TypedValue{IntegerValue: valueOf(5)}},
	}
	localLoadMetric15 := transit.Metric{
		Type:   "local_load_15",
		Labels: map[string]transit.TypedValue{
			"cores": transit.TypedValue{IntegerValue: &cores},
			"sampleTime": transit.TypedValue{IntegerValue: valueOf(15)}},
	}
	sample1 := transit.TimeSeries{
		Metric:     &localLoadMetric1,
		MetricKind: transit.GAUGE,
		Points: 	makePoints(),
		Resource:   &serviceLocalLoad,
		ValueType:  transit.DoubleType,
	}
	sample5 := transit.TimeSeries{
		Metric:     &localLoadMetric5,
		MetricKind: transit.GAUGE,
		Points: 	makePoints(),
		Resource:   &serviceLocalLoad,
		ValueType:  transit.DoubleType,
	}
	sample15 := transit.TimeSeries{
		Metric:     &localLoadMetric15,
		MetricKind: transit.GAUGE,
		Points: 	makePoints(),
		Resource:   &serviceLocalLoad,
		ValueType:  transit.DoubleType,
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

func AvroPlayground() {
	localhost := DummyMonitoredResource{
		Name: "localhost",
		Type:   transit.HostResource,
		Status: "UNSCHEDULED_DOWN",
		Labels: map[string]string{"hostGroup": "egain-21", "appType": "nagios"},
	}
	l2 := new(DummyMonitoredResource)
	l2.Name = "Dumb"
	l2.Type = "MyType"
	// l2.Status = transit.HOST_SCHEDULED_DOWN
	l2.Labels = make(map[string]string)
	l2.Labels["one"] = "b"
	l2.Labels["two"] = "c"
	schema, err := avro.ParseSchema(transit.MONITORED_RESOURCE)
	if err != nil {
		// Should not happen if the schema is valid
		panic(err)
	}
	writer := avro.NewSpecificDatumWriter()
	// SetSchema must be called before calling Write
	writer.SetSchema(schema)
	// Create a new Buffer and Encoder to write to this Buffer
	buffer := new(bytes.Buffer)
	encoder := avro.NewBinaryEncoder(buffer)
	// Write the record
	writer.Write(&localhost, encoder)

	reader := avro.NewSpecificDatumReader()
	// SetSchema must be called before calling Read
	reader.SetSchema(schema)
	// Create a new Decoder with a given buffer
	decoder := avro.NewBinaryDecoder(buffer.Bytes())
	// Create a new TestRecord to decode data into
	decodedRecord := new(DummyMonitoredResource)
	// Read data into a given record with a given Decoder. Unlike GenericDatumReader the first parameter should be the value to map data into.
	err = reader.Read(decodedRecord, decoder)
	if err != nil {
		panic(err)
	}

	fmt.Println("Read a value: ", decodedRecord)
}

type DummyMonitoredResource struct {
	Name string
	Type string
	Status string
	Labels map[string]string
}

