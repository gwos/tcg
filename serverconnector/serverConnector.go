package serverconnector

import (
	"github.com/gwos/tng/log"
	"github.com/gwos/tng/milliseconds"
	"github.com/gwos/tng/transit"
	"github.com/shirou/gopsutil/cpu"
	"github.com/shirou/gopsutil/disk"
	"github.com/shirou/gopsutil/host"
	"github.com/shirou/gopsutil/mem"
	"github.com/shirou/gopsutil/process"
	"time"
)

const (
	TotalDiskUsageServiceName   = "total.disk.usage"
	TotalMemoryUsageServiceName = "total.memory.usage"
	TotalCpuUsageServiceName    = "total.cpu.usage"
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
	ProcessesNumberCriticalValue         = 800
	ProcessesNumberWarningValue          = 700
)

var hostName string // TODO: Vlad why use global? - Because we need to set owner in all functions but owner always the same

var LastCheck milliseconds.MillisecondTimestamp

func Synchronize() *transit.InventoryResource {
	hostStat, err := host.Info()
	if err != nil {
		log.Error(err)
		return nil
	}

	hostName = hostStat.Hostname

	LastCheck = milliseconds.MillisecondTimestamp{Time: time.Now()}

	inventoryResource := transit.InventoryResource{
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
				Name:  "cpu.usage.total",
				Type:  "network-device",
				Owner: hostName,
			},
		},
	}

	processesMap := collectProcesses()

	for processName, _ := range processesMap {
		inventoryResource.Services = append(inventoryResource.Services, transit.InventoryService{
			Name:  processName,
			Type:  transit.NetworkDevice,
			Owner: hostName,
		})
	}

	return &inventoryResource
}

func CollectMetrics() *transit.MonitoredResource {
	hostStat, err := host.Info()
	if err != nil {
		log.Error(err)
		return nil
	}

	hostName = hostStat.Hostname

	monitoredResource := transit.MonitoredResource{
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

	processesMap := collectProcesses()

	for processName, processCpu := range processesMap {
		monitoredResource.Services = append(monitoredResource.Services, transit.MonitoredService{
			Name:          processName,
			Type:          transit.Service,
			Status:        transit.ServiceOk,
			Owner:         hostName,
			LastCheckTime: milliseconds.MillisecondTimestamp{Time: time.Now()},
			NextCheckTime: milliseconds.MillisecondTimestamp{Time: time.Now().Local().Add(time.Second * 5)},
			Metrics: []transit.TimeSeries{
				{
					MetricName: processName,
					SampleType: transit.Value,
					Interval: &transit.TimeInterval{
						EndTime:   milliseconds.MillisecondTimestamp{Time: time.Now()},
						StartTime: milliseconds.MillisecondTimestamp{Time: time.Now()},
					},
					Value: &transit.TypedValue{
						ValueType:   transit.DoubleType,
						DoubleValue: processCpu,
					},
					Unit: transit.PercentCPU,
				},
				{
					MetricName: processName + "_cr",
					SampleType: transit.Value,
					Interval: &transit.TimeInterval{
						EndTime:   milliseconds.MillisecondTimestamp{Time: time.Now()},
						StartTime: milliseconds.MillisecondTimestamp{Time: time.Now()},
					},
					Value: &transit.TypedValue{
						ValueType:   transit.DoubleType,
						DoubleValue: TotalCpuUsageCriticalValue,
					},
					Unit: transit.PercentCPU,
				},
				{
					MetricName: processName + "_wn",
					SampleType: transit.Value,
					Interval: &transit.TimeInterval{
						EndTime:   milliseconds.MillisecondTimestamp{Time: time.Now()},
						StartTime: milliseconds.MillisecondTimestamp{Time: time.Now()},
					},
					Value: &transit.TypedValue{
						ValueType:    transit.IntegerType,
						IntegerValue: TotalCpuUsageWarningValue,
					},
					Unit: transit.PercentCPU,
				},
			},
		})
	}

	return &monitoredResource
}

func getTotalDiskUsageService() *transit.MonitoredService {
	diskStats, err := disk.Usage("/")
	if err != nil {
		log.Error(err)
		return nil
	}

	return &transit.MonitoredService{
		Name:          TotalDiskUsageServiceName,
		Type:          transit.Service,
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
		log.Error(err)
		return nil
	}

	return &transit.MonitoredService{
		Name:          DiskUsedServiceName,
		Type:          transit.Service,
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
		log.Error(err.Error())
		return nil
	}

	return &transit.MonitoredService{
		Name:          DiskFreeServiceName,
		Type:          transit.Service,
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
		log.Error(err.Error())
		return nil
	}

	return &transit.MonitoredService{
		Name:          TotalMemoryUsageServiceName,
		Type:          transit.Service,
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
		log.Error(err.Error())
		return nil
	}
	return &transit.MonitoredService{
		Name:          MemoryUsedServiceName,
		Type:          transit.Service,
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
		log.Error(err.Error())
		return nil
	}

	return &transit.MonitoredService{
		Name:          MemoryFreeServiceName,
		Type:          transit.Service,
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
		log.Error(err)
		return nil
	}

	return &transit.MonitoredService{
		Name:          ProcessesNumberServiceName,
		Type:          transit.Service,
		Status:        transit.ServiceOk,
		Owner:         hostName,
		LastCheckTime: milliseconds.MillisecondTimestamp{Time: time.Now()},
		NextCheckTime: milliseconds.MillisecondTimestamp{Time: time.Now().Local().Add(time.Second * 5)},
		Metrics: []transit.TimeSeries{
			{
				MetricName: "processes.number",
				SampleType: transit.Value,
				Interval: &transit.TimeInterval{
					EndTime:   milliseconds.MillisecondTimestamp{Time: time.Now()},
					StartTime: milliseconds.MillisecondTimestamp{Time: time.Now()},
				},
				Value: &transit.TypedValue{
					ValueType:    transit.IntegerType,
					IntegerValue: int64(hostStat.Procs),
				},
				Unit: transit.UnitCounter,
			},
			{
				MetricName: "processes.number_cr",
				SampleType: transit.Value,
				Interval: &transit.TimeInterval{
					EndTime:   milliseconds.MillisecondTimestamp{Time: time.Now()},
					StartTime: milliseconds.MillisecondTimestamp{Time: time.Now()},
				},
				Value: &transit.TypedValue{
					ValueType:    transit.IntegerType,
					IntegerValue: ProcessesNumberCriticalValue,
				},
				Unit: transit.UnitCounter,
			},
			{
				MetricName: "processes.number_wn",
				SampleType: transit.Value,
				Interval: &transit.TimeInterval{
					EndTime:   milliseconds.MillisecondTimestamp{Time: time.Now()},
					StartTime: milliseconds.MillisecondTimestamp{Time: time.Now()},
				},
				Value: &transit.TypedValue{
					ValueType:    transit.IntegerType,
					IntegerValue: ProcessesNumberWarningValue,
				},
				Unit: transit.UnitCounter,
			},
		},
	}
}

func getTotalCpuUsage() *transit.MonitoredService {
	interval := time.Now()

	service := transit.MonitoredService{
		Name:             TotalCpuUsageServiceName,
		Type:             transit.Service,
		Status:           transit.ServiceOk,
		Owner:            hostName,
		LastPlugInOutput: "CPU OK",
		LastCheckTime:    milliseconds.MillisecondTimestamp{Time: interval},
		NextCheckTime:    milliseconds.MillisecondTimestamp{Time: interval.Local().Add(time.Second * 5)},
		Metrics: []transit.TimeSeries{
			{
				MetricName: "cpu.usage.total",
				SampleType: transit.Value,
				Interval: &transit.TimeInterval{
					EndTime:   milliseconds.MillisecondTimestamp{Time: interval},
					StartTime: milliseconds.MillisecondTimestamp{Time: interval},
				},
				Value: &transit.TypedValue{
					ValueType:    transit.IntegerType,
					IntegerValue: getCpuUsage(),
				},
				Unit: transit.PercentCPU,
			},
			{
				MetricName: "cpu.usage.total_cr",
				SampleType: transit.Value,
				Interval: &transit.TimeInterval{
					EndTime:   milliseconds.MillisecondTimestamp{Time: interval},
					StartTime: milliseconds.MillisecondTimestamp{Time: interval},
				},
				Value: &transit.TypedValue{
					ValueType:    transit.IntegerType,
					IntegerValue: TotalCpuUsageCriticalValue,
				},
				Unit: transit.PercentCPU,
			},
			{
				MetricName: "cpu.usage.total_wn",
				SampleType: transit.Value,
				Interval: &transit.TimeInterval{
					EndTime:   milliseconds.MillisecondTimestamp{Time: interval},
					StartTime: milliseconds.MillisecondTimestamp{Time: interval},
				},
				Value: &transit.TypedValue{
					ValueType:    transit.IntegerType,
					IntegerValue: TotalCpuUsageWarningValue,
				},
				Unit: transit.PercentCPU,
			},
		},
	}

	//processesMap := collectProcesses()
	//
	//for processName, processCpu := range processesMap {
	//	service.Metrics = append(service.Metrics, transit.TimeSeries{
	//		MetricName: processName,
	//		SampleType: transit.Value,
	//		Interval: &transit.TimeInterval{
	//			EndTime:   milliseconds.MillisecondTimestamp{Time: interval},
	//			StartTime: milliseconds.MillisecondTimestamp{Time: interval},
	//		},
	//		Value: &transit.TypedValue{
	//			ValueType:   transit.DoubleType,
	//			DoubleValue: processCpu,
	//		},
	//		Unit: transit.PercentCPU,
	//	})
	//	break
	//}

	return &service
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
	pr, _ := process.Processes()

	processes := make([]*Process, 0)
	for _, proc := range pr {

		cpuUsed, err := proc.CPUPercent()
		if err != nil {
			log.Error(err)
		}

		name, err := proc.Name()
		if err != nil {
			log.Error(err)
		}

		processes = append(processes, &Process{name, cpuUsed})

	}

	m := make(map[string]float64)
	for _, p := range processes {
		_, exists := m[p.name]
		if exists {
			m[p.name] = m[p.name] + p.cpu
		} else {
			m[p.name] = p.cpu
		}
	}

	return m
}
