package main
import (
	"flag"
	"fmt"
	"github.com/gwos/tng/transit"
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
	// Example Usage with a host
	geneva := transit.MonitoredResource{
		Name: "geneva",
		Type:   transit.HostResource,
		Status: transit.HOST_UP,
		LastCheckTime: transit.MillisecondTimestamp{Time: time.Now()},
		NextCheckTime: transit.MillisecondTimestamp{Time: time.Now().Add(time.Minute * 5)},
		LastPlugInOutput: "44/55/888 QA00005-BC",
		Description: "Subversion Server",
		Properties: map[string]transit.TypedValue{
			"stateType":       transit.TypedValue{StringValue: "SOFT"},
			"checkType":       transit.TypedValue{StringValue: "ACTIVE"},
			"PerformanceData": transit.TypedValue{StringValue: "007-321 RAD"},
			"ExecutionTime":   transit.TypedValue{DoubleValue: 3.0},
			"CurrentAttempt":  transit.TypedValue{IntegerValue: 2},
			"InceptionTime":   transit.TypedValue{TimeValue: transit.MillisecondTimestamp{Time: time.Now()}},
		},
	}
	localLoadService := transit.MonitoredResource{
		Name: "local_load",
		Type:   transit.ServiceResource,
		Status: transit.SERVICE_OK,
		LastCheckTime: transit.MillisecondTimestamp{Time: time.Now()},
		NextCheckTime: transit.MillisecondTimestamp{Time: time.Now().Add(time.Minute * 5)},
		LastPlugInOutput: "foo | bar",
		Description: "Load on subversion",
		Properties: map[string]transit.TypedValue{
			"stateType":       transit.TypedValue{StringValue: "SOFT"},
			"checkType":       transit.TypedValue{StringValue: "ACTIVE"},
			"PerformanceData": transit.TypedValue{StringValue: "007-321 RAD"},
			"ExecutionTime":   transit.TypedValue{DoubleValue: 3.0},
			"CurrentAttempt":  transit.TypedValue{IntegerValue: 2},
			"InceptionTime":   transit.TypedValue{TimeValue: transit.MillisecondTimestamp{Time: time.Now()}},
		},
	}
	metricSample := makeMetricSample()
	sampleValue := transit.TimeSeries{
		MetricName: "local_load_5",
		// Labels:      []*LabelDescriptor{&cores, &sampleTime},
		MetricSamples: []*transit.MetricSample{
		    &transit.MetricSample{
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
		    &transit.MetricSample{
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
		    &transit.MetricSample{
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
	fmt.Println(*stats)
}

func makeMetricSample() *transit.MetricSample {
	random := rand.Float64() * 100.0
	now := transit.MillisecondTimestamp{Time: time.Now()}
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
		now := transit.MillisecondTimestamp{Time: time.Now()}
		metricSamples[i] = &transit.MetricSample{
			SampleType: transit.Value,
			Interval:   &transit.TimeInterval{EndTime: now, StartTime: now},
			Value:      &transit.TypedValue{DoubleValue: random},
		}
	}
	return metricSamples;
}
