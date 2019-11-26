package main

import (
    "bytes"
    "encoding/json"
    "flag"
    "fmt"
    "github.com/gwos/tng/milliseconds"
    "github.com/gwos/tng/services"
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

var transitService = services.GetTransitService()

// examples
func main() {
    flag.Parse()
    fmt.Printf("Starting Groundwork Agent on port %d\n", *argPort)

    err := transitService.StartNats()
    if err != nil {
        fmt.Printf("%s", err.Error())
        return
    }

    defer func() {
        err = transitService.StopNats()
        if err != nil {
            fmt.Printf("%s", err.Error())
        }
        cmd := exec.Command("rm", "-rf", "src")
        _, err = cmd.Output()
        if err != nil {
            fmt.Printf("%s", err.Error())
        }
    }()

    // VLAD - I think the gatherMetrics could be made into an advanced feature built into the ServerConnector:
    // 	1. From the PID, we can get the process name
    //  2. provide a list of process names that we want to monitor
    //  3. Look up the CPU usage for a process with a given name and turn it into a Service
    //  This way we can (a) get the status of processes running on a server (b) get their cpu usage with thresholds
    //  After finishing serverConnector.CollectMetrics(), please implement this in the ServerConnector
    //processes := gatherMetrics()

    //for _, p := range processes {
    //    log.Println("Process ", p.pid, " takes ", p.cpu, " % of the CPU")
    //}

    //server := serverConnector.CollectMetrics()
    //println(server.Name)

    err = sendInventoryResources()
    if err != nil {
        fmt.Printf("%s", err.Error())
        return
    }

    time.Sleep(5 * time.Second)

    err = sendMonitoredResources()
    if err != nil {
        fmt.Printf("%s", err.Error())
        return
    }

    time.Sleep(10 * time.Second)
}

func sendInventoryResources() error {
    inventoryRequest := transit.InventoryRequest{
        Context: transit.TracerContext{
            AppType:    "VEMA",
            AgentID:    "3939333393342",
            TraceToken: "token-99e93",
            TimeStamp:  milliseconds.MillisecondTimestamp{Time: time.Now()},
        },
        Resources: []transit.InventoryResource{
            {
                Name: "geneva",
                Type: "HOST",
                Services: []transit.InventoryService{
                    {
                        Name:  "local_load",
                        Type:  "network-device",
                        Owner: "geneva",
                    },
                },
            },
        },
        Groups: nil,
    }

    b, err := json.Marshal(inventoryRequest)
    if err != nil {
        return err
    }

    err = transitService.SynchronizeInventory(b)

    return err
}

func sendMonitoredResources() error {
    // Create a Metrics Sample
    metricSample := makeMetricSample()
    sampleValue := transit.TimeSeries{
        MetricName: "local_load_5",
        // Labels:      []*LabelDescriptor{&cores, &sampleTime},
        SampleType: transit.Value,
        Interval:   metricSample.Interval,
        Value:      metricSample.Value,
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
        SampleType: transit.Critical,
        Interval:   metricSample.Interval,
        Value:      metricSample.Value,
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
        SampleType: transit.Warning,
        Interval:   metricSample.Interval,
        Value:      metricSample.Value,
        Tags: map[string]string{
            "deviceTag":     "127.0.0.1",
            "httpMethodTag": "POST",
            "httpStatusTag": "200",
        },
        Unit: "%{cpu}",
    }

    // Example Service
    var localLoadService = transit.MonitoredService{
        Name:          "local_load",
        Type:          transit.Service,
        Owner:         "geneva",
        Status:        transit.ServiceOk,
        LastCheckTime: milliseconds.MillisecondTimestamp{Time: time.Now()},
        NextCheckTime: milliseconds.MillisecondTimestamp{Time: time.Now().Add(time.Minute * 5)},
        LastPlugInOutput: "foo | bar",
        Properties: map[string]transit.TypedValue{
            "stateType": {
                ValueType:   "StringType",
                StringValue: "SOFT",
            },
            "checkType": {
                ValueType:   "StringType",
                StringValue: "ACTIVE",
            },
            "PerformanceData": {
                ValueType:   "StringType",
                StringValue: "007-321 RAD",
            },
            "ExecutionTime": {
                ValueType:   "DoubleType",
                DoubleValue: 3.0,
            },
            "CurrentAttempt": {
                ValueType:    "IntegerType",
                IntegerValue: 2,
            },
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
           "stateType":       {ValueType: "StringType", StringValue: "SOFT"},
           "checkType":       {ValueType: "StringType", StringValue: "ACTIVE"},
           "PerformanceData": {ValueType: "StringType", StringValue: "007-321 RAD"},
           "ExecutionTime":   {ValueType: "DoubleType", DoubleValue: 3.0},
           "CurrentAttempt":  {ValueType: "IntegerType", IntegerValue: 2},
           //"InceptionTime":   {TimeValue: milliseconds.MillisecondTimestamp{Time: time.Now()}},
       },
       Services: []transit.MonitoredService{localLoadService},
    }

    request := transit.ResourcesWithServicesRequest{
        Context: transit.TracerContext{
            AppType:    "VEMA",
            AgentID:    "3939333393342",
            TraceToken: "token-99e93",
            TimeStamp:  milliseconds.MillisecondTimestamp{Time: time.Now()},
        },
        Resources: []transit.MonitoredResource{geneva},
    }

    b, err := json.Marshal(request)
    if err != nil {
        return err
    }

    return transitService.SendResourceWithMetrics(b)
}

func makeMetricSample() *transit.MetricSample {
    random := rand.Float64() * 100.0
    now := milliseconds.MillisecondTimestamp{Time: time.Now()}
    return &transit.MetricSample{
        SampleType: transit.Value,
        Interval:   &transit.TimeInterval{EndTime: now, StartTime: now},
        Value:      &transit.TypedValue{ValueType: "DoubleType", DoubleValue: random},
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

func gatherMetrics() []*Process {
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
