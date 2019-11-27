package serverconnector

import (
    "bytes"
    "fmt"
    "github.com/gwos/tng/milliseconds"
    "github.com/gwos/tng/transit"
    "github.com/shirou/gopsutil/cpu"
    "github.com/shirou/gopsutil/disk"
    "github.com/shirou/gopsutil/host"
    "github.com/shirou/gopsutil/mem"
    "log"
    "os/exec"
    "strconv"
    "strings"
    "time"
)

const (
    MB  uint64 = 1048576
)

var hostName string // TODO: Vlad why use global?

var LastCheck milliseconds.MillisecondTimestamp

func Synchronize() *transit.InventoryResource {
    hostStat, err := host.Info()
    if err != nil {
        log.Println(err)
        return nil
    }

    hostName = hostStat.Hostname

    LastCheck = milliseconds.MillisecondTimestamp{Time: time.Now()}

    return &transit.InventoryResource{
        Name: hostName,
        Type: "HOST",
        Services: []transit.InventoryService{
            {
                Name:  "total.disk.usage",
                Type:  "network-device",
                Owner: hostName,
            },
            {
                Name:  "disk.used",
                Type:  "network-device",
                Owner: hostName,
            },
            {
                Name:  "disk.free",
                Type:  "network-device",
                Owner: hostName,
            },
            {
                Name:  "total.memory.usage",
                Type:  "network-device",
                Owner: hostName,
            },
            {
                Name:  "memory.used",
                Type:  "network-device",
                Owner: hostName,
            },
            {
                Name:  "memory.free",
                Type:  "network-device",
                Owner: hostName,
            },
            {
                Name:  "processes.number",
                Type:  "network-device",
                Owner: hostName,
            },
            {
                Name:  "cpu.usage.total",
                Type:  "network-device",
                Owner: hostName,
            },
        },
    }
}

func CollectMetrics() *transit.MonitoredResource {
    hostStat, err := host.Info()
    if err != nil {
        log.Println(err)
        return nil
    }

    hostName = hostStat.Hostname

    return &transit.MonitoredResource{
        Name:          hostStat.Hostname,
        Type:          transit.Host,
        Status:        transit.HostUp,
        LastCheckTime: LastCheck,
        NextCheckTime: milliseconds.MillisecondTimestamp{Time: time.Now()},
        Services: []transit.MonitoredService{
            *getDiskFreeService(),
            *getTotalDiskUsageService(),
            *getDiskUsedService(),
            *getMemoryFreeService(),
            *getTotalMemoryUsageService(),
            *getMemoryUsedService(),
            *getNumberOfProcessesService(),
            *getTotalCpuUsage(),
        },
    }
}

func getTotalDiskUsageService() *transit.MonitoredService {
    diskStats, err := disk.Usage("/")
    if err != nil {
        fmt.Println(err.Error())
        return nil
    }

    return &transit.MonitoredService{
        Name:          "total.disk.usage",
        Status:        transit.ServiceOk,
        Owner:         hostName,
        LastCheckTime: milliseconds.MillisecondTimestamp{Time: time.Now()},
        NextCheckTime: milliseconds.MillisecondTimestamp{Time: time.Now()},  // TODO: VLAD - NEXT SHOULD NOT EQUAL LAST
        Metrics: []transit.TimeSeries{
            {
                MetricName: "totalDiskUsage",
                SampleType: transit.Value,
                Interval: &transit.TimeInterval{
                    EndTime:   milliseconds.MillisecondTimestamp{Time: time.Now()},
                    StartTime: milliseconds.MillisecondTimestamp{Time: time.Now()},
                },
                Value: &transit.TypedValue{
                    ValueType:    transit.IntegerType,
                    IntegerValue: int64(diskStats.Total / MB),
                },
                Unit: transit.MB,
            },
        },
    }
}

func getDiskUsedService() *transit.MonitoredService {
    diskStats, err := disk.Usage("/")
    if err != nil {
        fmt.Println(err.Error())
        return nil
    }

    return &transit.MonitoredService{
        Name:          "disk.used",
        Status:        transit.ServiceOk,
        Owner:         hostName,
        LastCheckTime: milliseconds.MillisecondTimestamp{Time: time.Now()},
        NextCheckTime: milliseconds.MillisecondTimestamp{Time: time.Now()}, // TODO: VLAD - NEXT SHOULD NOT EQUAL LAST	EEEE
        Metrics: []transit.TimeSeries{
            {
                MetricName: "diskUsed",
                SampleType: transit.Value,
                Interval: &transit.TimeInterval{
                    EndTime:   milliseconds.MillisecondTimestamp{Time: time.Now()},
                    StartTime: milliseconds.MillisecondTimestamp{Time: time.Now()},
                },
                Value: &transit.TypedValue{
                    ValueType:    transit.IntegerType,
                    IntegerValue: int64(diskStats.Used / MB),
                },
                Unit: transit.MB,
            },
        },
    }

}

func getDiskFreeService() *transit.MonitoredService {
    diskStats, err := disk.Usage("/")
    if err != nil {
        log.Println(err.Error())
        return nil
    }

    return &transit.MonitoredService{
        Name:          "disk.free",
        Status:        transit.ServiceOk,
        Owner:         hostName,
        LastCheckTime: milliseconds.MillisecondTimestamp{Time: time.Now()},
        NextCheckTime: milliseconds.MillisecondTimestamp{Time: time.Now()},
        Metrics: []transit.TimeSeries{
            {
                MetricName: "diskFree",
                SampleType: transit.Value,
                Interval: &transit.TimeInterval{
                    EndTime:   milliseconds.MillisecondTimestamp{Time: time.Now()},
                    StartTime: milliseconds.MillisecondTimestamp{Time: time.Now()},
                },
                Value: &transit.TypedValue{
                    ValueType:    transit.IntegerType,
                    IntegerValue: int64(diskStats.Free / MB),
                },
                Unit: transit.MB,
            },
        },
    }
}

func getTotalMemoryUsageService() *transit.MonitoredService {
    vmStats, err := mem.VirtualMemory()
    if err != nil {
        log.Println(err.Error())
        return nil
    }

    return &transit.MonitoredService{
        Name:          "total.memory.usage",
        Status:        transit.ServiceOk,
        Owner:         hostName,
        LastCheckTime: milliseconds.MillisecondTimestamp{Time: time.Now()},
        NextCheckTime: milliseconds.MillisecondTimestamp{Time: time.Now()},
        Metrics: []transit.TimeSeries{
            {
                MetricName: "totalMemoryUsage",
                SampleType: transit.Value,
                Interval: &transit.TimeInterval{
                    EndTime:   milliseconds.MillisecondTimestamp{Time: time.Now()},
                    StartTime: milliseconds.MillisecondTimestamp{Time: time.Now()},
                },
                Value: &transit.TypedValue{
                    ValueType:    transit.IntegerType,
                    IntegerValue: int64(vmStats.Total / MB),
                },
                Unit: transit.MB,
            },
        },
    }
}

func getMemoryUsedService() *transit.MonitoredService {
    vmStats, err := mem.VirtualMemory()
    if err != nil {
        log.Println(err.Error())
        return nil
    }
    return &transit.MonitoredService{
        Name:          "memory.used",
        Status:        transit.ServiceOk,
        Owner:         hostName,
        LastCheckTime: milliseconds.MillisecondTimestamp{Time: time.Now()},
        NextCheckTime: milliseconds.MillisecondTimestamp{Time: time.Now()},
        Metrics: []transit.TimeSeries{
            {
                MetricName: "memoryUsed",
                SampleType: transit.Value,
                Interval: &transit.TimeInterval{
                    EndTime:   milliseconds.MillisecondTimestamp{Time: time.Now()},
                    StartTime: milliseconds.MillisecondTimestamp{Time: time.Now()},
                },
                Value: &transit.TypedValue{
                    ValueType:    transit.IntegerType,
                    IntegerValue: int64(vmStats.Used / MB),
                },
                Unit: transit.MB,
            },
        },
    }
}

func getMemoryFreeService() *transit.MonitoredService {
    vmStats, err := mem.VirtualMemory()
    if err != nil {
        log.Println(err.Error())
        return nil
    }

    return &transit.MonitoredService{
        Name:          "memory.free",
        Status:        transit.ServiceOk,
        Owner:         hostName,
        LastCheckTime: milliseconds.MillisecondTimestamp{Time: time.Now()},
        NextCheckTime: milliseconds.MillisecondTimestamp{Time: time.Now()},
        Metrics: []transit.TimeSeries{
            {
                MetricName: "memoryFree",
                SampleType: transit.Value,
                Interval: &transit.TimeInterval{
                    EndTime:   milliseconds.MillisecondTimestamp{Time: time.Now()},
                    StartTime: milliseconds.MillisecondTimestamp{Time: time.Now()},
                },
                Value: &transit.TypedValue{
                    ValueType:    transit.IntegerType,
                    IntegerValue: int64(vmStats.Free / MB),
                },
                Unit: transit.MB,
            },
        },
    }
}

func getNumberOfProcessesService() *transit.MonitoredService {
    hostStat, err := host.Info()
    if err != nil {
        log.Println(err)
        return nil
    }

    return &transit.MonitoredService{
        Name:          "processes.number",
        Status:        transit.ServiceOk,
        Owner:         hostName,
        LastCheckTime: milliseconds.MillisecondTimestamp{Time: time.Now()},
        NextCheckTime: milliseconds.MillisecondTimestamp{Time: time.Now()},
        Metrics: []transit.TimeSeries{
            {
                MetricName: "processesNumber",
                SampleType: transit.Value,
                Interval: &transit.TimeInterval{
                    EndTime:   milliseconds.MillisecondTimestamp{Time: time.Now()},
                    StartTime: milliseconds.MillisecondTimestamp{Time: time.Now()},
                },
                Value: &transit.TypedValue{
                    ValueType:    transit.IntegerType,
                    IntegerValue: int64(hostStat.Procs),
                },
            },
        },
    }
}

func getTotalCpuUsage() *transit.MonitoredService {
    return &transit.MonitoredService{
        Name:          "cpu.usage.total",
        Status:        transit.ServiceOk,
        Owner:         hostName,
        LastCheckTime: milliseconds.MillisecondTimestamp{Time: time.Now()},
        NextCheckTime: milliseconds.MillisecondTimestamp{Time: time.Now()},
        Metrics: []transit.TimeSeries{
            {
                MetricName: "cpuUsageTotal",
                SampleType: transit.Value,
                Interval: &transit.TimeInterval{
                    EndTime:   milliseconds.MillisecondTimestamp{Time: time.Now()},
                    StartTime: milliseconds.MillisecondTimestamp{Time: time.Now()},
                },
                Value: &transit.TypedValue{
                    ValueType:    transit.IntegerType,
                    IntegerValue: int64(collectProcessMetrics()), // see getCpuUsage
                },
            },
        },
    }
}

// this uses gopsutil
func getCpuUsage() {
    // TODO implement this
    percentages, _ := cpu.Percent(0, true)
    for _, cpupercent := range percentages {
        strconv.FormatFloat(cpupercent, 'f', 2, 64);
    }

}
// TODO: this function should support named process matching
func collectProcessMetrics() float64 {
    cmd := exec.Command("ps", "aux")
    var out bytes.Buffer
    cmd.Stdout = &out
    err := cmd.Run()
    if err != nil {
        log.Fatal(err)
    }

    var totalCpu float64
    var totalCount int
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
        _, err = strconv.Atoi(ft[1])
        if err != nil {
            continue
        }
        cpuUsed, err := strconv.ParseFloat(ft[2], 64)
        if err != nil {
            log.Fatal(err)
        }
        totalCpu += cpuUsed
        totalCount++
    }

    return totalCpu / float64(totalCount) * 100
}
