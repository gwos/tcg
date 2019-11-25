package main

import (
    "encoding/json"
    "flag"
    "fmt"
    "github.com/gwos/tng/milliseconds"
    "github.com/gwos/tng/transit"
    "math/rand"
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
    geneva := transit.ResourceStatus{
        Name:             "geneva",
        Type:             transit.HostResource,
        Status:           transit.HostUp,
        LastCheckTime:    milliseconds.MillisecondTimestamp{Time: time.Now()},
        NextCheckTime:    milliseconds.MillisecondTimestamp{Time: time.Now().Add(time.Minute * 5)},
        LastPlugInOutput: "44/55/888 QA00005-BC",
        Properties: map[string]transit.TypedValue{
            "stateType":       transit.TypedValue{StringValue: "SOFT"},
            "checkType":       transit.TypedValue{StringValue: "ACTIVE"},
            "PerformanceData": transit.TypedValue{StringValue: "007-321 RAD"},
            "ExecutionTime":   transit.TypedValue{DoubleValue: 3.0},
            "CurrentAttempt":  transit.TypedValue{IntegerValue: 2},
            "InceptionTime":   transit.TypedValue{TimeValue: milliseconds.MillisecondTimestamp{Time: time.Now()}},
        },
    }
    localLoadService := transit.ResourceStatus{
        Name:             "local_load",
        Type:             transit.ServiceResource,
        Status:           transit.ServiceOk,
        LastCheckTime:    milliseconds.MillisecondTimestamp{Time: time.Now()},
        NextCheckTime:    milliseconds.MillisecondTimestamp{Time: time.Now().Add(time.Minute * 5)},
        LastPlugInOutput: "foo | bar",
        Properties: map[string]transit.TypedValue{
            "stateType":       transit.TypedValue{StringValue: "SOFT"},
            "checkType":       transit.TypedValue{StringValue: "ACTIVE"},
            "PerformanceData": transit.TypedValue{StringValue: "007-321 RAD"},
            "ExecutionTime":   transit.TypedValue{DoubleValue: 3.0},
            "CurrentAttempt":  transit.TypedValue{IntegerValue: 2},
            "InceptionTime":   transit.TypedValue{TimeValue: milliseconds.MillisecondTimestamp{Time: time.Now()}},
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
    println("Resources: ", resources[0].Resource.Name)
    bytes, error := json.Marshal(resources)
    if error == nil {
        s := string(bytes)
        println(s);
    }
    //var transitServices,_ = transit.Connect(transit.Credentials{
    //	User:     "test",
    //	Password: "test",
    //})
    //
    //transitServices.SendResourcesWithMetrics(resources)

    // Retrieve Metrics List with Transit
    //transit.Disconnect(transitServices)
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
    return metricSamples;
}
