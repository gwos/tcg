package main

import (
    "encoding/json"
    "fmt"
    "github.com/gwos/tng/milliseconds"
    "github.com/gwos/tng/serverconnector"
    "github.com/gwos/tng/services"
    "github.com/gwos/tng/transit"
    "log"
    "os/exec"
    "time"
)

var transitService = services.GetTransitService()

func main() {
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

    err = sendInventoryResources(*serverconnector.Synchronize())

    for {
        err := sendMonitoredResources(*serverconnector.CollectMetrics())
        if err != nil {
            log.Println(err.Error())
        }

        serverconnector.LastCheck = milliseconds.MillisecondTimestamp{Time: time.Now()}

        time.Sleep(5 * time.Second)
    }
}

func sendInventoryResources(resource transit.InventoryResource) error {
    inventoryRequest := transit.InventoryRequest{
        Context: transit.TracerContext{
            AppType:    "VEMA",
            AgentID:    "3939333393342",
            TraceToken: "token-99e93",
            TimeStamp:  milliseconds.MillisecondTimestamp{Time: time.Now()},
        },
        Resources: []transit.InventoryResource{resource},
        Groups: nil,
    }

    b, err := json.Marshal(inventoryRequest)
    if err != nil {
        return err
    }

    err = transitService.SynchronizeInventory(b)

    return err
}

func sendMonitoredResources(resource transit.MonitoredResource) error {
    request := transit.ResourcesWithServicesRequest{
        Context: transit.TracerContext{
            AppType:    "VEMA",
            AgentID:    "3939333393342",
            TraceToken: "token-99e93",
            TimeStamp:  milliseconds.MillisecondTimestamp{Time: time.Now()},
        },
        Resources: []transit.MonitoredResource{resource},
    }

    b, err := json.Marshal(request)
    if err != nil {
        return err
    }

    return transitService.SendResourceWithMetrics(b)
}

/*metricSample := makeMetricSample()
  sampleValue := transit.TimeSeries{
      MetricName: "local_load_5",
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

  var localLoadService = transit.MonitoredService{
      Name:             "local_load",
      Type:             transit.Service,
      Owner:            "geneva",
      Status:           transit.ServiceOk,
      LastCheckTime:    milliseconds.MillisecondTimestamp{Time: time.Now()},
      NextCheckTime:    milliseconds.MillisecondTimestamp{Time: time.Now().Add(time.Minute * 5)},
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
  }

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
          "InceptionTime":   {ValueType: "DateType", DateValue: milliseconds.MillisecondTimestamp{Time: time.Now()}},
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
  }*/

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
