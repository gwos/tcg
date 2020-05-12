package main

import (
	"github.com/gwos/tcg/cache"
	"github.com/gwos/tcg/connectors"
	"github.com/gwos/tcg/log"
	"github.com/gwos/tcg/milliseconds"
	"github.com/gwos/tcg/transit"
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

const (
	MB uint64 = 1048576
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
			Name:  connectors.Name(pr.Name, pr.CustomName),
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
				*function.(func(int, int, string, time.Duration) *transit.MonitoredService)(pr.WarningThreshold, pr.CriticalThreshold, pr.CustomName, timerSeconds))
		} else {
			notDefaultProcesses = append(notDefaultProcesses, pr)
		}
	}

	processesMap := collectMonitoredProcesses(notDefaultProcesses)
	interval := time.Now()

	for processName, processValues := range processesMap {
		value := transit.TypedValue{
			ValueType:   transit.DoubleType,
			DoubleValue: processValues.processCpu,
		}
		warningThreshold := transit.ThresholdValue{
			SampleType: transit.Warning,
			Label:      processName + "_wn",
			Value: &transit.TypedValue{
				ValueType:    transit.IntegerType,
				IntegerValue: int64(processValues.warningValue),
			}}
		errorThreshold := transit.ThresholdValue{
			SampleType: transit.Critical,
			Label:      processName + "_cr",
			Value: &transit.TypedValue{
				ValueType:    transit.IntegerType,
				IntegerValue: int64(processValues.criticalValue),
			}}
		monitoredService := transit.MonitoredService{
			Name: processName,
			Type: transit.Service,
			Status: connectors.CalculateStatus(&value,
				&transit.TypedValue{
					ValueType:    transit.IntegerType,
					IntegerValue: int64(processValues.warningValue),
				},
				&transit.TypedValue{
					ValueType:    transit.IntegerType,
					IntegerValue: int64(processValues.criticalValue),
				}),
			Owner:         hostName,
			LastCheckTime: milliseconds.MillisecondTimestamp{Time: interval},
			NextCheckTime: milliseconds.MillisecondTimestamp{Time: interval.Local().Add(time.Second * timerSeconds)},
			Metrics: []transit.TimeSeries{
				{
					MetricName:        processName,
					MetricComputeType: processValues.computeType,
					MetricExpression:  processValues.expression,
					SampleType:        transit.Value,
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
		if processValues.processCpu == -1 {
			monitoredService.Status = transit.ServicePending
		}
		monitoredResource.Services = append(monitoredResource.Services, monitoredService)
	}

	return &monitoredResource
}

func getTotalDiskUsageService(warningThresholdValue int, criticalThresholdValue int, customName string, timerSeconds time.Duration) *transit.MonitoredService {
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
		Label:      connectors.Name(TotalDiskAllocatedServiceName, customName) + "_wn",
		Value:      &warningValue,
	}
	criticalThreshold := transit.ThresholdValue{
		SampleType: transit.Critical,
		Label:      connectors.Name(TotalDiskAllocatedServiceName, customName) + "_cr",
		Value:      &criticalValue,
	}

	return &transit.MonitoredService{
		Name:          connectors.Name(TotalDiskAllocatedServiceName, customName),
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

func getDiskUsedService(warningThresholdValue int, criticalThresholdValue int, customName string, timerSeconds time.Duration) *transit.MonitoredService {
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
		Label:      connectors.Name(DiskUsedServiceName, customName) + "_wn",
		Value:      &warningValue,
	}
	criticalThreshold := transit.ThresholdValue{
		SampleType: transit.Critical,
		Label:      connectors.Name(DiskUsedServiceName, customName) + "_cr",
		Value:      &criticalValue,
	}

	return &transit.MonitoredService{
		Name:          connectors.Name(DiskUsedServiceName, customName),
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

func getDiskFreeService(warningThresholdValue int, criticalThresholdValue int, customName string, timerSeconds time.Duration) *transit.MonitoredService {
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
		Label:      connectors.Name(DiskFreeServiceName, customName) + "_wn",
		Value:      &warningValue,
	}
	criticalThreshold := transit.ThresholdValue{
		SampleType: transit.Critical,
		Label:      connectors.Name(DiskFreeServiceName, customName) + "_cr",
		Value:      &criticalValue,
	}

	return &transit.MonitoredService{
		Name:          connectors.Name(DiskFreeServiceName, customName),
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

func getTotalMemoryUsageService(warningThresholdValue int, criticalThresholdValue int, customName string, timerSeconds time.Duration) *transit.MonitoredService {
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
		Label:      connectors.Name(TotalMemoryUsageAllocatedName, customName) + "_wn",
		Value:      &warningValue,
	}
	criticalThreshold := transit.ThresholdValue{
		SampleType: transit.Critical,
		Label:      connectors.Name(TotalMemoryUsageAllocatedName, customName) + "_cr",
		Value:      &criticalValue,
	}

	return &transit.MonitoredService{
		Name:          connectors.Name(TotalMemoryUsageAllocatedName, customName),
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

func getMemoryUsedService(warningThresholdValue int, criticalThresholdValue int, customName string, timerSeconds time.Duration) *transit.MonitoredService {
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
		Label:      connectors.Name(MemoryUsedServiceName, customName) + "_wn",
		Value:      &warningValue,
	}
	criticalThreshold := transit.ThresholdValue{
		SampleType: transit.Critical,
		Label:      connectors.Name(MemoryUsedServiceName, customName) + "_cr",
		Value:      &criticalValue,
	}

	return &transit.MonitoredService{
		Name:          connectors.Name(MemoryUsedServiceName, customName),
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

func getMemoryFreeService(warningThresholdValue int, criticalThresholdValue int, customName string, timerSeconds time.Duration) *transit.MonitoredService {
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
		Label:      connectors.Name(MemoryFreeServiceName, customName) + "_wn",
		Value:      &warningValue,
	}
	criticalThreshold := transit.ThresholdValue{
		SampleType: transit.Critical,
		Label:      connectors.Name(MemoryFreeServiceName, customName) + "_cr",
		Value:      &criticalValue,
	}

	return &transit.MonitoredService{
		Name:          connectors.Name(MemoryFreeServiceName, customName),
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

func getNumberOfProcessesService(warningThresholdValue int, criticalThresholdValue int, customName string, timerSeconds time.Duration) *transit.MonitoredService {
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
		Label:      connectors.Name(ProcessesNumberServiceName, customName) + "_wn",
		Value:      &warningValue,
	}
	criticalThreshold := transit.ThresholdValue{
		SampleType: transit.Critical,
		Label:      connectors.Name(ProcessesNumberServiceName, customName) + "_cr",
		Value:      &criticalValue,
	}

	return &transit.MonitoredService{
		Name:          connectors.Name(ProcessesNumberServiceName, customName),
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

func getTotalCPUUsage(warningThresholdValue int, criticalThresholdValue int, customName string, timerSeconds time.Duration) *transit.MonitoredService {
	interval := time.Now()
	value := transit.TypedValue{
		ValueType:    transit.IntegerType,
		IntegerValue: getCPUUsage(),
	}

	warningValue := transit.TypedValue{ValueType: transit.IntegerType, IntegerValue: int64(warningThresholdValue)}
	criticalValue := transit.TypedValue{ValueType: transit.IntegerType, IntegerValue: int64(criticalThresholdValue)}

	warningThreshold := transit.ThresholdValue{
		SampleType: transit.Warning,
		Label:      connectors.Name(TotalCPUUsageServiceName, customName) + "_wn",
		Value:      &warningValue,
	}
	criticalThreshold := transit.ThresholdValue{
		SampleType: transit.Critical,
		Label:      connectors.Name(TotalCPUUsageServiceName, customName) + "_cr",
		Value:      &criticalValue,
	}

	return &transit.MonitoredService{
		Name:          connectors.Name(TotalCPUUsageServiceName, customName),
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
}

func getCPUUsage() int64 {
	percentages, _ := cpu.Percent(0, false)
	return int64(percentages[0])
}

type localProcess struct {
	name string
	cpu  float64
}

type values struct {
	processCpu    float64
	computeType   transit.ComputeType
	expression    string
	criticalValue int
	warningValue  int
}

// Collects a map of process names to cpu usage, given a list of processes to be monitored
func collectMonitoredProcesses(monitoredProcesses []transit.MetricDefinition) map[string]values {
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
		if _, exists := m[p.name]; exists {
			m[p.name] = m[p.name] + p.cpu
		} else {
			m[p.name] = p.cpu
		}
	}

	processesMap := make(map[string]values)
	for _, pr := range monitoredProcesses {
		name := pr.Name
		if pr.CustomName != "" {
			name = pr.CustomName
		}
		if _, exists := m[pr.Name]; exists && pr.ComputeType != transit.Synthetic {
			processesMap[name] = values{
				processCpu:    m[pr.Name],
				criticalValue: pr.CriticalThreshold,
				warningValue:  pr.WarningThreshold,
				computeType:   pr.ComputeType,
				expression:    "",
			}
		} else {
			processesMap[name] = values{
				processCpu:    -1,
				criticalValue: -1,
				warningValue:  -1,
				computeType:   transit.Synthetic,
				expression:    pr.Expression,
			}
		}
	}

	return processesMap
}

func listSuggestions(name string) []string {
	hostProcesses, _ := cache.ProcessesCache.Get("processes")

	var processes []string
	for _, hostProcess := range hostProcesses.([]string) {
		if strings.Contains(hostProcess, name) {
			processes = append(processes, hostProcess)
		}
	}

	return processes
}

func collectProcessesNames() []string {
	var processes []string
	hostProcesses, _ := process.Processes()

	for _, hostProcess := range hostProcesses {
		if name, err := hostProcess.Name(); err == nil {
			processes = append(processes, name)
		} else {
			log.Error(err)
		}
	}
	return processes
}
