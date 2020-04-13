package main

import (
	"encoding/json"
	"github.com/gwos/tng/connectors"
	"github.com/gwos/tng/log"
	"github.com/gwos/tng/milliseconds"
	"github.com/gwos/tng/transit"
	"github.com/shirou/gopsutil/cpu"
	"github.com/shirou/gopsutil/disk"
	"github.com/shirou/gopsutil/host"
	"github.com/shirou/gopsutil/mem"
	"github.com/shirou/gopsutil/process"
	"strings"
	"time"
)

// Default processes names
const (
	TotalDiskAllocatedServiceName = "total.disk.allocated"
	TotalMemoryUsageAllocatedName = "total.memory.allocated"
	TotalCPUUsageServiceName      = "total.cpu.usage"
	DiskUsedServiceName           = "disk.used"
	MemoryUsedServiceName         = "memory.used"
	DiskFreeServiceName           = "disk.free"
	MemoryFreeServiceName         = "memory.free"
	ProcessesNumberServiceName    = "processes.number"
)

// Default 'Critical' and 'Warning' values for monitored processes(in MB)
// TODO: remove these when thresholds ready in database and ui
const (
	MB                           uint64 = 1048576
	ProcessCPUUsageCriticalValue        = 0.90
	ProcessCPUUsageWarningValue         = 0.50
)

var processToFuncMap = map[string]interface{}{
	TotalDiskAllocatedServiceName: getTotalDiskUsageService,
	TotalMemoryUsageAllocatedName: getTotalMemoryUsageService,
	TotalCPUUsageServiceName:      getTotalCPUUsage,
	DiskUsedServiceName:           getDiskUsedService,
	MemoryUsedServiceName:         getMemoryUsedService,
	DiskFreeServiceName:           getDiskFreeService,
	MemoryFreeServiceName:         getMemoryFreeService,
	ProcessesNumberServiceName:    getNumberOfProcessesService,
}

var hostName string

// LastCheck provide time of last processes state check
var LastCheck milliseconds.MillisecondTimestamp

// Synchronize inventory for necessary processes
func Synchronize(processes []transit.MetricDefinition) *transit.InventoryResource {
	hostStat, err := host.Info()
	if err != nil {
		log.Error(err)
		return nil
	}

	hostName = hostStat.Hostname

	LastCheck = milliseconds.MillisecondTimestamp{Time: time.Now()}

	inventoryResource := transit.InventoryResource{
		Name:     hostName,
		Device:   "SomeDevice",
		Type:     transit.Host,
		Services: []transit.InventoryService{},
	}

	for _, pr := range processes {
		inventoryResource.Services = append(inventoryResource.Services, transit.InventoryService{
			Name:  pr.Name,
			Type:  transit.Service,
			Owner: hostName,
		})
	}

	return &inventoryResource
}

// CollectMetrics method gather metrics data for necessary processes
func CollectMetrics(processes []transit.MetricDefinition, timerSeconds time.Duration) *transit.MonitoredResource {
	hostStat, err := host.Info()
	if err != nil {
		log.Error(err)
		return nil
	}

	hostName = hostStat.Hostname

	LastCheck = milliseconds.MillisecondTimestamp{Time: time.Now()}

	monitoredResource := transit.MonitoredResource{
		Name:          hostStat.Hostname,
		Type:          transit.Host,
		Status:        transit.HostUp,
		LastCheckTime: LastCheck,
		NextCheckTime: milliseconds.MillisecondTimestamp{Time: LastCheck.Local().Add(time.Second * timerSeconds)},
		Services:      []transit.MonitoredService{},
	}

	var notDefaultProcesses []transit.MetricDefinition
	for _, pr := range processes {
		if function, exists := processToFuncMap[pr.Name]; exists {
			monitoredResource.Services = append(monitoredResource.Services,
				*function.(func(int, int, time.Duration) *transit.MonitoredService)(pr.WarningThreshold, pr.CriticalThreshold, timerSeconds))
		} else {
			notDefaultProcesses = append(notDefaultProcesses, pr)
		}
	}

	processesMap := collectProcesses(notDefaultProcesses)
	interval := time.Now()

	warningValue := transit.TypedValue{ValueType: transit.DoubleType, DoubleValue: ProcessCPUUsageWarningValue}
	criticalValue := transit.TypedValue{ValueType: transit.DoubleType, DoubleValue: ProcessCPUUsageCriticalValue}

	for processName, processCPU := range processesMap {
		value := transit.TypedValue{
			ValueType:   transit.DoubleType,
			DoubleValue: processCPU,
		}
		warningThreshold := transit.ThresholdValue{
			SampleType: transit.Warning,
			Label:      processName + "_wn",
			Value:      &warningValue}
		errorThreshold := transit.ThresholdValue{
			SampleType: transit.Critical,
			Label:      processName + "_cr",
			Value:      &criticalValue}
		monitoredService := transit.MonitoredService{
			Name:          processName,
			Type:          transit.Service,
			Status:        connectors.CalculateStatus(&value, &warningValue, &criticalValue),
			Owner:         hostName,
			LastCheckTime: milliseconds.MillisecondTimestamp{Time: interval},
			NextCheckTime: milliseconds.MillisecondTimestamp{Time: interval.Local().Add(time.Second * timerSeconds)},
			Metrics: []transit.TimeSeries{
				{
					MetricName: processName,
					SampleType: transit.Value,
					Interval: &transit.TimeInterval{
						EndTime:   milliseconds.MillisecondTimestamp{Time: interval},
						StartTime: milliseconds.MillisecondTimestamp{Time: interval},
					},
					Value:      &value,
					Unit:       transit.PercentCPU,
					Thresholds: &[]transit.ThresholdValue{warningThreshold, errorThreshold},
				},
			},
		}
		if processCPU == -1 {
			monitoredService.Status = transit.ServicePending
		}
		monitoredResource.Services = append(monitoredResource.Services, monitoredService)
	}

	return &monitoredResource
}

func getTotalDiskUsageService(warningThresholdValue int, criticalThresholdValue int, timerSeconds time.Duration) *transit.MonitoredService {
	interval := time.Now()
	diskStats, err := disk.Usage("/")
	if err != nil {
		log.Error(err)
		return nil
	}

	value := transit.TypedValue{
		ValueType:    transit.IntegerType,
		IntegerValue: int64(diskStats.Total / MB),
	}

	warningValue := transit.TypedValue{ValueType: transit.DoubleType, DoubleValue: float64(warningThresholdValue)}
	criticalValue := transit.TypedValue{ValueType: transit.DoubleType, DoubleValue: float64(criticalThresholdValue)}

	warningThreshold := transit.ThresholdValue{
		SampleType: transit.Warning,
		Label:      TotalDiskAllocatedServiceName + "_wn",
		Value:      &warningValue,
	}
	criticalThreshold := transit.ThresholdValue{
		SampleType: transit.Critical,
		Label:      TotalDiskAllocatedServiceName + "_cr",
		Value:      &criticalValue,
	}

	return &transit.MonitoredService{
		Name:          TotalDiskAllocatedServiceName,
		Type:          transit.Service,
		Status:        connectors.CalculateStatus(&value, &warningValue, &criticalValue),
		Owner:         hostName,
		LastCheckTime: milliseconds.MillisecondTimestamp{Time: interval},
		NextCheckTime: milliseconds.MillisecondTimestamp{Time: interval.Local().Add(time.Second * timerSeconds)},
		Metrics: []transit.TimeSeries{
			{
				MetricName: TotalDiskAllocatedServiceName,
				SampleType: transit.Value,
				Interval: &transit.TimeInterval{
					EndTime:   milliseconds.MillisecondTimestamp{Time: interval},
					StartTime: milliseconds.MillisecondTimestamp{Time: interval},
				},
				Value:      &value,
				Unit:       transit.MB,
				Thresholds: &[]transit.ThresholdValue{warningThreshold, criticalThreshold},
			},
		},
	}
}

func getDiskUsedService(warningThresholdValue int, criticalThresholdValue int, timerSeconds time.Duration) *transit.MonitoredService {
	interval := time.Now()
	diskStats, err := disk.Usage("/")
	if err != nil {
		log.Error(err)
		return nil
	}

	value := transit.TypedValue{
		ValueType:    transit.IntegerType,
		IntegerValue: int64(diskStats.Used / MB),
	}

	warningValue := transit.TypedValue{ValueType: transit.IntegerType, IntegerValue: int64(warningThresholdValue)}
	criticalValue := transit.TypedValue{ValueType: transit.IntegerType, IntegerValue: int64(criticalThresholdValue)}

	warningThreshold := transit.ThresholdValue{
		SampleType: transit.Warning,
		Label:      DiskUsedServiceName + "_wn",
		Value:      &warningValue,
	}
	criticalThreshold := transit.ThresholdValue{
		SampleType: transit.Critical,
		Label:      DiskUsedServiceName + "_cr",
		Value:      &criticalValue,
	}

	return &transit.MonitoredService{
		Name:          DiskUsedServiceName,
		Type:          transit.Service,
		Status:        connectors.CalculateStatus(&value, &warningValue, &criticalValue),
		Owner:         hostName,
		LastCheckTime: milliseconds.MillisecondTimestamp{Time: interval},
		NextCheckTime: milliseconds.MillisecondTimestamp{Time: interval.Local().Add(time.Second * timerSeconds)},
		Metrics: []transit.TimeSeries{
			{
				MetricName: DiskUsedServiceName,
				SampleType: transit.Value,
				Interval: &transit.TimeInterval{
					EndTime:   milliseconds.MillisecondTimestamp{Time: interval},
					StartTime: milliseconds.MillisecondTimestamp{Time: interval},
				},
				Value:      &value,
				Unit:       transit.MB,
				Thresholds: &[]transit.ThresholdValue{warningThreshold, criticalThreshold},
			},
		},
	}
}

func getDiskFreeService(warningThresholdValue int, criticalThresholdValue int, timerSeconds time.Duration) *transit.MonitoredService {
	interval := time.Now()
	diskStats, err := disk.Usage("/")
	if err != nil {
		log.Error(err.Error())
		return nil
	}
	value := transit.TypedValue{
		ValueType:    transit.IntegerType,
		IntegerValue: int64(diskStats.Free / MB),
	}

	warningValue := transit.TypedValue{ValueType: transit.IntegerType, IntegerValue: int64(warningThresholdValue)}
	criticalValue := transit.TypedValue{ValueType: transit.IntegerType, IntegerValue: int64(criticalThresholdValue)}

	warningThreshold := transit.ThresholdValue{
		SampleType: transit.Warning,
		Label:      DiskFreeServiceName + "_wn",
		Value:      &warningValue,
	}
	criticalThreshold := transit.ThresholdValue{
		SampleType: transit.Critical,
		Label:      DiskFreeServiceName + "_cr",
		Value:      &criticalValue,
	}

	return &transit.MonitoredService{
		Name:          DiskFreeServiceName,
		Type:          transit.Service,
		Status:        connectors.CalculateStatus(&value, &warningValue, &criticalValue),
		Owner:         hostName,
		LastCheckTime: milliseconds.MillisecondTimestamp{Time: interval},
		NextCheckTime: milliseconds.MillisecondTimestamp{Time: interval.Local().Add(time.Second * timerSeconds)},
		Metrics: []transit.TimeSeries{
			{
				MetricName: DiskFreeServiceName,
				SampleType: transit.Value,
				Interval: &transit.TimeInterval{
					EndTime:   milliseconds.MillisecondTimestamp{Time: interval},
					StartTime: milliseconds.MillisecondTimestamp{Time: interval},
				},
				Value:      &value,
				Unit:       transit.MB,
				Thresholds: &[]transit.ThresholdValue{warningThreshold, criticalThreshold},
			},
		},
	}
}

func getTotalMemoryUsageService(warningThresholdValue int, criticalThresholdValue int, timerSeconds time.Duration) *transit.MonitoredService {
	interval := time.Now()
	vmStats, err := mem.VirtualMemory()
	if err != nil {
		log.Error(err.Error())
		return nil
	}

	value := transit.TypedValue{
		ValueType:    transit.IntegerType,
		IntegerValue: int64(vmStats.Total / MB),
	}

	warningValue := transit.TypedValue{ValueType: transit.IntegerType, IntegerValue: int64(warningThresholdValue)}
	criticalValue := transit.TypedValue{ValueType: transit.IntegerType, IntegerValue: int64(criticalThresholdValue)}

	warningThreshold := transit.ThresholdValue{
		SampleType: transit.Warning,
		Label:      TotalMemoryUsageAllocatedName + "_wn",
		Value:      &warningValue}
	criticalThreshold := transit.ThresholdValue{
		SampleType: transit.Critical,
		Label:      TotalMemoryUsageAllocatedName + "_cr",
		Value:      &criticalValue}

	return &transit.MonitoredService{
		Name:          TotalMemoryUsageAllocatedName,
		Type:          transit.Service,
		Status:        connectors.CalculateStatus(&value, &warningValue, &criticalValue),
		Owner:         hostName,
		LastCheckTime: milliseconds.MillisecondTimestamp{Time: interval},
		NextCheckTime: milliseconds.MillisecondTimestamp{Time: interval.Local().Add(time.Second * timerSeconds)},
		Metrics: []transit.TimeSeries{
			{
				MetricName: TotalMemoryUsageAllocatedName,
				SampleType: transit.Value,
				Interval: &transit.TimeInterval{
					EndTime:   milliseconds.MillisecondTimestamp{Time: interval},
					StartTime: milliseconds.MillisecondTimestamp{Time: interval},
				},
				Value:      &value,
				Unit:       transit.MB,
				Thresholds: &[]transit.ThresholdValue{warningThreshold, criticalThreshold},
			},
		},
	}
}

func getMemoryUsedService(warningThresholdValue int, criticalThresholdValue int, timerSeconds time.Duration) *transit.MonitoredService {
	interval := time.Now()
	vmStats, err := mem.VirtualMemory()
	if err != nil {
		log.Error(err.Error())
		return nil
	}

	value := transit.TypedValue{
		ValueType:    transit.IntegerType,
		IntegerValue: int64(vmStats.Used / MB),
	}

	warningValue := transit.TypedValue{ValueType: transit.IntegerType, IntegerValue: int64(warningThresholdValue)}
	criticalValue := transit.TypedValue{ValueType: transit.IntegerType, IntegerValue: int64(criticalThresholdValue)}

	warningThreshold := transit.ThresholdValue{
		SampleType: transit.Warning,
		Label:      MemoryUsedServiceName + "_wn",
		Value:      &warningValue,
	}
	criticalThreshold := transit.ThresholdValue{
		SampleType: transit.Critical,
		Label:      MemoryUsedServiceName + "_cr",
		Value:      &criticalValue,
	}

	return &transit.MonitoredService{
		Name:          MemoryUsedServiceName,
		Type:          transit.Service,
		Status:        connectors.CalculateStatus(&value, &warningValue, &criticalValue),
		Owner:         hostName,
		LastCheckTime: milliseconds.MillisecondTimestamp{Time: interval},
		NextCheckTime: milliseconds.MillisecondTimestamp{Time: interval.Local().Add(time.Second * timerSeconds)},
		Metrics: []transit.TimeSeries{
			{
				MetricName: MemoryUsedServiceName,
				SampleType: transit.Value,
				Interval: &transit.TimeInterval{
					EndTime:   milliseconds.MillisecondTimestamp{Time: interval},
					StartTime: milliseconds.MillisecondTimestamp{Time: interval},
				},
				Value:      &value,
				Unit:       transit.MB,
				Thresholds: &[]transit.ThresholdValue{warningThreshold, criticalThreshold},
			},
		},
	}
}

func getMemoryFreeService(warningThresholdValue int, criticalThresholdValue int, timerSeconds time.Duration) *transit.MonitoredService {
	interval := time.Now()
	vmStats, err := mem.VirtualMemory()
	if err != nil {
		log.Error(err.Error())
		return nil
	}

	value := transit.TypedValue{
		ValueType:    transit.IntegerType,
		IntegerValue: int64(vmStats.Free / MB),
	}

	warningValue := transit.TypedValue{ValueType: transit.IntegerType, IntegerValue: int64(warningThresholdValue)}
	criticalValue := transit.TypedValue{ValueType: transit.IntegerType, IntegerValue: int64(criticalThresholdValue)}

	warningThreshold := transit.ThresholdValue{
		SampleType: transit.Warning,
		Label:      MemoryFreeServiceName + "_wn",
		Value:      &warningValue,
	}
	criticalThreshold := transit.ThresholdValue{
		SampleType: transit.Critical,
		Label:      MemoryFreeServiceName + "_cr",
		Value:      &criticalValue,
	}

	return &transit.MonitoredService{
		Name:          MemoryFreeServiceName,
		Type:          transit.Service,
		Status:        connectors.CalculateStatus(&value, &warningValue, &criticalValue),
		Owner:         hostName,
		LastCheckTime: milliseconds.MillisecondTimestamp{Time: interval},
		NextCheckTime: milliseconds.MillisecondTimestamp{Time: interval.Local().Add(time.Second * timerSeconds)},
		Metrics: []transit.TimeSeries{
			{
				MetricName: MemoryFreeServiceName,
				SampleType: transit.Value,
				Interval: &transit.TimeInterval{
					EndTime:   milliseconds.MillisecondTimestamp{Time: interval},
					StartTime: milliseconds.MillisecondTimestamp{Time: interval},
				},
				Value:      &value,
				Unit:       transit.MB,
				Thresholds: &[]transit.ThresholdValue{warningThreshold, criticalThreshold},
			},
		},
	}
}

func getNumberOfProcessesService(warningThresholdValue int, criticalThresholdValue int, timerSeconds time.Duration) *transit.MonitoredService {
	interval := time.Now()
	hostStat, err := host.Info()
	if err != nil {
		log.Error(err)
		return nil
	}

	value := transit.TypedValue{
		ValueType:    transit.IntegerType,
		IntegerValue: int64(hostStat.Procs),
	}

	warningValue := transit.TypedValue{ValueType: transit.IntegerType, IntegerValue: int64(warningThresholdValue)}
	criticalValue := transit.TypedValue{ValueType: transit.IntegerType, IntegerValue: int64(criticalThresholdValue)}

	warningThreshold := transit.ThresholdValue{
		SampleType: transit.Warning,
		Label:      ProcessesNumberServiceName + "_wn",
		Value:      &warningValue,
	}
	criticalThreshold := transit.ThresholdValue{
		SampleType: transit.Critical,
		Label:      ProcessesNumberServiceName + "_cr",
		Value:      &criticalValue,
	}

	return &transit.MonitoredService{
		Name:          ProcessesNumberServiceName,
		Type:          transit.Service,
		Status:        connectors.CalculateStatus(&value, &warningValue, &criticalValue),
		Owner:         hostName,
		LastCheckTime: milliseconds.MillisecondTimestamp{Time: interval},
		NextCheckTime: milliseconds.MillisecondTimestamp{Time: interval.Local().Add(time.Second * timerSeconds)},
		Metrics: []transit.TimeSeries{
			{
				MetricName: ProcessesNumberServiceName,
				SampleType: transit.Value,
				Interval: &transit.TimeInterval{
					EndTime:   milliseconds.MillisecondTimestamp{Time: interval},
					StartTime: milliseconds.MillisecondTimestamp{Time: interval},
				},
				Value:      &value,
				Unit:       transit.UnitCounter,
				Thresholds: &[]transit.ThresholdValue{warningThreshold, criticalThreshold},
			},
		},
	}
}

func getTotalCPUUsage(warningThresholdValue int, criticalThresholdValue int, timerSeconds time.Duration) *transit.MonitoredService {
	interval := time.Now()
	value := transit.TypedValue{
		ValueType:    transit.IntegerType,
		IntegerValue: getCPUUsage(),
	}

	warningValue := transit.TypedValue{ValueType: transit.IntegerType, IntegerValue: int64(warningThresholdValue)}
	criticalValue := transit.TypedValue{ValueType: transit.IntegerType, IntegerValue: int64(criticalThresholdValue)}

	warningThreshold := transit.ThresholdValue{
		SampleType: transit.Warning,
		Label:      TotalCPUUsageServiceName + "_wn",
		Value:      &warningValue,
	}
	criticalThreshold := transit.ThresholdValue{
		SampleType: transit.Critical,
		Label:      TotalCPUUsageServiceName + "_cr",
		Value:      &criticalValue,
	}

	service := transit.MonitoredService{
		Name:          TotalCPUUsageServiceName,
		Type:          transit.Service,
		Status:        connectors.CalculateStatus(&value, &warningValue, &criticalValue),
		Owner:         hostName,
		LastCheckTime: milliseconds.MillisecondTimestamp{Time: interval},
		NextCheckTime: milliseconds.MillisecondTimestamp{Time: interval.Local().Add(time.Second * timerSeconds)},
		Metrics: []transit.TimeSeries{
			{
				MetricName: TotalCPUUsageServiceName,
				SampleType: transit.Value,
				Interval: &transit.TimeInterval{
					EndTime:   milliseconds.MillisecondTimestamp{Time: interval},
					StartTime: milliseconds.MillisecondTimestamp{Time: interval},
				},
				Value:      &value,
				Unit:       transit.PercentCPU,
				Thresholds: &[]transit.ThresholdValue{warningThreshold, criticalThreshold},
			},
		},
	}
	return &service
}

func getCPUUsage() int64 {
	percentages, _ := cpu.Percent(0, false)
	return int64(percentages[0])
}

type localProcess struct {
	name string
	cpu  float64
}

// Collects a map of process names to cpu usage, given a list of processes to be monitored
func collectProcesses(monitoredProcesses []transit.MetricDefinition) map[string]float64 {
	hostProcesses, _ := process.Processes()

	processes := make([]*localProcess, 0)
	for _, hostProcess := range hostProcesses {
		cpuUsed, err := hostProcess.CPUPercent()
		if err != nil {
			log.Error(err)
		}

		name, err := hostProcess.Name()
		if err != nil {
			log.Error(err)
		}

		processes = append(processes, &localProcess{name, cpuUsed})
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

	processesMap := make(map[string]float64)
	for _, pr := range monitoredProcesses {
		_, exists := m[pr.Name]
		if exists {
			processesMap[pr.Name] = m[pr.Name]
		} else {
			processesMap[pr.Name] = -1
		}
	}

	return processesMap
}

func listSuggestions(name string) string {
	hostProcesses, _ := process.Processes()

	processes := make([]*transit.MetricDefinition, 0)
	for _, hostProcess := range hostProcesses {
		processName, err := hostProcess.Name()
		if err != nil {
			log.Error(err)
		}

		if strings.Contains(processName, name) {
			processes = append(processes, &transit.MetricDefinition{
				Name:              processName,
				MetricType:        "Gauge",
				ComputeType:       "Query",
				ServiceType:       "Process",
				AggregateType:     "average",
				WarningThreshold:  -1,
				CriticalThreshold: -1,
			})
		}
	}
	jsonBytes, _ := json.Marshal(processes)
	return string(jsonBytes)
}
