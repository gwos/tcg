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
	TotalDiskUsageServiceName   = "total.disk.usage"
	TotalMemoryUsageServiceName = "total.memory.usage"
	TotalCpuUsageServiceName    = "cpu.usage.total"
	DiskUsedServiceName         = "disk.used"
	MemoryUsedServiceName       = "memory.used"
	DiskFreeServiceName         = "disk.free"
	MemoryFreeServiceName       = "memory.free"
	ProcessesNumberServiceName  = "processes.number"
)

const (
	MB                            uint64 = 1048576
	TotalDiskUsageCriticalValue          = 500000
	TotalDiskUsageWarningValue           = 350000
	TotalMemoryUsageCriticalValue        = 50000
	TotalMemoryUsageWarningValue         = 35000
	TotalCpuUsageCriticalValue           = 90
	TotalCpuUsageWarningValue            = 70
	DiskUsedCriticalValue                = 400000
	DiskUsedWarningValue                 = 300000
	MemoryUsedCriticalValue              = 400000
	MemoryUsedWarningValue               = 300000
	DiskFreeCriticalValue                = 10000
	DiskFreeWarningValue                 = 30000
	MemoryFreeCriticalValue              = 100
	MemoryFreeWarningValue               = 300
	ProcessesNumberCriticalValue         = 120
	ProcessesNumberWarningValue          = 80
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
		Type: transit.Host,
		Services: []transit.InventoryService{
			{
				Name:  TotalDiskUsageServiceName,
				Type:  transit.NetworkDevice,
				Owner: hostName,
			},
			{
				Name:  DiskUsedServiceName,
				Type:  transit.NetworkDevice,
				Owner: hostName,
			},
			{
				Name:  DiskFreeServiceName,
				Type:  transit.NetworkDevice,
				Owner: hostName,
			},
			{
				Name:  TotalMemoryUsageServiceName,
				Type:  transit.NetworkDevice,
				Owner: hostName,
			},
			{
				Name:  MemoryUsedServiceName,
				Type:  transit.NetworkDevice,
				Owner: hostName,
			},
			{
				Name:  MemoryFreeServiceName,
				Type:  transit.NetworkDevice,
				Owner: hostName,
			},
			{
				Name:  ProcessesNumberServiceName,
				Type:  transit.NetworkDevice,
				Owner: hostName,
			},
			{
				Name:  TotalCpuUsageServiceName,
				Type:  transit.NetworkDevice,
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
		Name:          TotalDiskUsageServiceName,
		Status:        transit.ServiceOk,
		Owner:         hostName,
		LastCheckTime: milliseconds.MillisecondTimestamp{Time: time.Now()},
		NextCheckTime: milliseconds.MillisecondTimestamp{Time: time.Now().Local().Add(time.Second * 5)},
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
			{
				MetricName: "totalDiskUsage_cr",
				SampleType: transit.Value,
				Interval: &transit.TimeInterval{
					EndTime:   milliseconds.MillisecondTimestamp{Time: time.Now()},
					StartTime: milliseconds.MillisecondTimestamp{Time: time.Now()},
				},
				Value: &transit.TypedValue{
					ValueType:    transit.IntegerType,
					IntegerValue: TotalDiskUsageCriticalValue,
				},
				Unit: transit.MB,
			},
			{
				MetricName: "totalDiskUsage_wn",
				SampleType: transit.Value,
				Interval: &transit.TimeInterval{
					EndTime:   milliseconds.MillisecondTimestamp{Time: time.Now()},
					StartTime: milliseconds.MillisecondTimestamp{Time: time.Now()},
				},
				Value: &transit.TypedValue{
					ValueType:    transit.IntegerType,
					IntegerValue: TotalDiskUsageWarningValue,
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
		Name:          DiskUsedServiceName,
		Status:        transit.ServiceOk,
		Owner:         hostName,
		LastCheckTime: milliseconds.MillisecondTimestamp{Time: time.Now()},
		NextCheckTime: milliseconds.MillisecondTimestamp{Time: time.Now().Local().Add(time.Minute * 5)},
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
			{
				MetricName: "diskUsed_cr",
				SampleType: transit.Value,
				Interval: &transit.TimeInterval{
					EndTime:   milliseconds.MillisecondTimestamp{Time: time.Now()},
					StartTime: milliseconds.MillisecondTimestamp{Time: time.Now()},
				},
				Value: &transit.TypedValue{
					ValueType:    transit.IntegerType,
					IntegerValue: DiskUsedCriticalValue,
				},
				Unit: transit.MB,
			},
			{
				MetricName: "diskUsed_wn",
				SampleType: transit.Value,
				Interval: &transit.TimeInterval{
					EndTime:   milliseconds.MillisecondTimestamp{Time: time.Now()},
					StartTime: milliseconds.MillisecondTimestamp{Time: time.Now()},
				},
				Value: &transit.TypedValue{
					ValueType:    transit.IntegerType,
					IntegerValue: DiskUsedWarningValue,
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
		Name:          DiskFreeServiceName,
		Status:        transit.ServiceOk,
		Owner:         hostName,
		LastCheckTime: milliseconds.MillisecondTimestamp{Time: time.Now()},
		NextCheckTime: milliseconds.MillisecondTimestamp{Time: time.Now().Local().Add(time.Second * 5)},
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
			{
				MetricName: "diskFree_cr",
				SampleType: transit.Value,
				Interval: &transit.TimeInterval{
					EndTime:   milliseconds.MillisecondTimestamp{Time: time.Now()},
					StartTime: milliseconds.MillisecondTimestamp{Time: time.Now()},
				},
				Value: &transit.TypedValue{
					ValueType:    transit.IntegerType,
					IntegerValue: DiskFreeCriticalValue,
				},
				Unit: transit.MB,
			},
			{
				MetricName: "diskFree_wn",
				SampleType: transit.Value,
				Interval: &transit.TimeInterval{
					EndTime:   milliseconds.MillisecondTimestamp{Time: time.Now()},
					StartTime: milliseconds.MillisecondTimestamp{Time: time.Now()},
				},
				Value: &transit.TypedValue{
					ValueType:    transit.IntegerType,
					IntegerValue: DiskFreeWarningValue,
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
		Name:          TotalMemoryUsageServiceName,
		Status:        transit.ServiceOk,
		Owner:         hostName,
		LastCheckTime: milliseconds.MillisecondTimestamp{Time: time.Now()},
		NextCheckTime: milliseconds.MillisecondTimestamp{Time: time.Now().Local().Add(time.Second * 5)},
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
			{
				MetricName: "totalMemoryUsage_cr",
				SampleType: transit.Value,
				Interval: &transit.TimeInterval{
					EndTime:   milliseconds.MillisecondTimestamp{Time: time.Now()},
					StartTime: milliseconds.MillisecondTimestamp{Time: time.Now()},
				},
				Value: &transit.TypedValue{
					ValueType:    transit.IntegerType,
					IntegerValue: TotalMemoryUsageCriticalValue,
				},
				Unit: transit.MB,
			},
			{
				MetricName: "totalMemoryUsage_wn",
				SampleType: transit.Value,
				Interval: &transit.TimeInterval{
					EndTime:   milliseconds.MillisecondTimestamp{Time: time.Now()},
					StartTime: milliseconds.MillisecondTimestamp{Time: time.Now()},
				},
				Value: &transit.TypedValue{
					ValueType:    transit.IntegerType,
					IntegerValue: TotalMemoryUsageWarningValue,
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
		Name:          MemoryUsedServiceName,
		Status:        transit.ServiceOk,
		Owner:         hostName,
		LastCheckTime: milliseconds.MillisecondTimestamp{Time: time.Now()},
		NextCheckTime: milliseconds.MillisecondTimestamp{Time: time.Now().Local().Add(time.Second * 5)},
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
			{
				MetricName: "memoryUsed_cr",
				SampleType: transit.Value,
				Interval: &transit.TimeInterval{
					EndTime:   milliseconds.MillisecondTimestamp{Time: time.Now()},
					StartTime: milliseconds.MillisecondTimestamp{Time: time.Now()},
				},
				Value: &transit.TypedValue{
					ValueType:    transit.IntegerType,
					IntegerValue: MemoryUsedCriticalValue,
				},
				Unit: transit.MB,
			},
			{
				MetricName: "memoryUsed_wn",
				SampleType: transit.Value,
				Interval: &transit.TimeInterval{
					EndTime:   milliseconds.MillisecondTimestamp{Time: time.Now()},
					StartTime: milliseconds.MillisecondTimestamp{Time: time.Now()},
				},
				Value: &transit.TypedValue{
					ValueType:    transit.IntegerType,
					IntegerValue: MemoryUsedWarningValue,
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
		Name:          MemoryFreeServiceName,
		Status:        transit.ServiceOk,
		Owner:         hostName,
		LastCheckTime: milliseconds.MillisecondTimestamp{Time: time.Now()},
		NextCheckTime: milliseconds.MillisecondTimestamp{Time: time.Now().Local().Add(time.Second * 5)},
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
			{
				MetricName: "memoryFree_cr",
				SampleType: transit.Value,
				Interval: &transit.TimeInterval{
					EndTime:   milliseconds.MillisecondTimestamp{Time: time.Now()},
					StartTime: milliseconds.MillisecondTimestamp{Time: time.Now()},
				},
				Value: &transit.TypedValue{
					ValueType:    transit.IntegerType,
					IntegerValue: MemoryFreeCriticalValue,
				},
				Unit: transit.MB,
			},
			{
				MetricName: "memoryFree_wn",
				SampleType: transit.Value,
				Interval: &transit.TimeInterval{
					EndTime:   milliseconds.MillisecondTimestamp{Time: time.Now()},
					StartTime: milliseconds.MillisecondTimestamp{Time: time.Now()},
				},
				Value: &transit.TypedValue{
					ValueType:    transit.IntegerType,
					IntegerValue: MemoryFreeWarningValue,
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
		Name:          ProcessesNumberServiceName,
		Status:        transit.ServiceOk,
		Owner:         hostName,
		LastCheckTime: milliseconds.MillisecondTimestamp{Time: time.Now()},
		NextCheckTime: milliseconds.MillisecondTimestamp{Time: time.Now().Local().Add(time.Second * 5)},
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
			{
				MetricName: "processesNumber_cr",
				SampleType: transit.Value,
				Interval: &transit.TimeInterval{
					EndTime:   milliseconds.MillisecondTimestamp{Time: time.Now()},
					StartTime: milliseconds.MillisecondTimestamp{Time: time.Now()},
				},
				Value: &transit.TypedValue{
					ValueType:    transit.IntegerType,
					IntegerValue: ProcessesNumberCriticalValue,
				},
			},
			{
				MetricName: "processesNumber_wn",
				SampleType: transit.Value,
				Interval: &transit.TimeInterval{
					EndTime:   milliseconds.MillisecondTimestamp{Time: time.Now()},
					StartTime: milliseconds.MillisecondTimestamp{Time: time.Now()},
				},
				Value: &transit.TypedValue{
					ValueType:    transit.IntegerType,
					IntegerValue: ProcessesNumberWarningValue,
				},
			},
		},
	}
}

func getTotalCpuUsage() *transit.MonitoredService {
	service := &transit.MonitoredService{
		Name:          TotalCpuUsageServiceName,
		Status:        transit.ServiceOk,
		Owner:         hostName,
		LastCheckTime: milliseconds.MillisecondTimestamp{Time: time.Now()},
		NextCheckTime: milliseconds.MillisecondTimestamp{Time: time.Now().Local().Add(time.Second * 5)},
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
					IntegerValue: getCpuUsage(), // see getCpuUsage
				},
			},
			{
				MetricName: "cpuUsageTotal_cr",
				SampleType: transit.Value,
				Interval: &transit.TimeInterval{
					EndTime:   milliseconds.MillisecondTimestamp{Time: time.Now()},
					StartTime: milliseconds.MillisecondTimestamp{Time: time.Now()},
				},
				Value: &transit.TypedValue{
					ValueType:    transit.IntegerType,
					IntegerValue: TotalCpuUsageCriticalValue, // see getCpuUsage
				},
			},
			{
				MetricName: "cpuUsageTotal_wn",
				SampleType: transit.Value,
				Interval: &transit.TimeInterval{
					EndTime:   milliseconds.MillisecondTimestamp{Time: time.Now()},
					StartTime: milliseconds.MillisecondTimestamp{Time: time.Now()},
				},
				Value: &transit.TypedValue{
					ValueType:    transit.IntegerType,
					IntegerValue: TotalCpuUsageWarningValue,
				},
			},
		},
	}

	processesMap := collectProcesses()

	for processName, cpuValue := range processesMap {
		tm := transit.TimeSeries{
			MetricName: processName,
			SampleType: transit.Value,
			Interval: &transit.TimeInterval{
				EndTime:   milliseconds.MillisecondTimestamp{Time: time.Now()},
				StartTime: milliseconds.MillisecondTimestamp{Time: time.Now()},
			},
			Value: &transit.TypedValue{
				ValueType:   transit.DoubleType,
				DoubleValue: cpuValue,
			},
		}
		service.Metrics = append(service.Metrics, tm)
	}

	return service
}

func getCpuUsage() int64 {
	percentages, _ := cpu.Percent(0, false)
	return int64(percentages[0])
}

type Process struct {
	name string
	cpu  float64
}

func collectProcesses() map[string]float64 {
	//TODO: Make ps to print full name
	cmd := exec.Command("ps", "-Ao", "fname,pcpu", "--no-headers")
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
		cpuUsed, err := strconv.ParseFloat(ft[1][:len(ft[1])-1], 64)
		if err != nil {
			log.Fatal(err)
		}
		if cpuUsed == 0 {
			cpuUsed = 0.1
		}

		processes = append(processes, &Process{ft[0], cpuUsed})

	}

	m := make(map[string]float64)
	for _, process := range processes {
		_, exists := m[process.name]
		if exists {
			m[process.name] = m[process.name] + process.cpu
		} else {
			m[process.name] = process.cpu
		}
	}

	return m
}
