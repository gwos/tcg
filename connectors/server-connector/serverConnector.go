package main

import (
	"encoding/json"
	"github.com/gin-gonic/gin"
	"github.com/gwos/tcg/cache"
	"github.com/gwos/tcg/connectors"
	"github.com/gwos/tcg/log"
	"github.com/gwos/tcg/milliseconds"
	"github.com/gwos/tcg/services"
	"github.com/gwos/tcg/transit"
	"github.com/shirou/gopsutil/cpu"
	"github.com/shirou/gopsutil/disk"
	"github.com/shirou/gopsutil/host"
	"github.com/shirou/gopsutil/mem"
	"github.com/shirou/gopsutil/process"
	"net/http"
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

// temporary solution, will be removed
var templateMetricName = "$view_Template#"

// Synchronize inventory for necessary processes
func Synchronize(processes []transit.MetricDefinition) *transit.InventoryResource {
	hostStat, err := host.Info()
	if err != nil {
		log.Error(err)
		return nil
	}

	hostName = hostStat.Hostname

	var services []transit.InventoryService
	for _, pr := range processes {
		// temporary solution, will be removed
		if pr.Name == templateMetricName {
			continue
		}
		service := connectors.CreateInventoryService(connectors.Name(pr.Name, pr.CustomName),
			hostName)
		services = append(services, service)
	}

	inventoryResource := connectors.CreateInventoryResource(hostName, services)

	return &inventoryResource
}

// CollectMetrics method gather metrics data for necessary processes
func CollectMetrics(processes []transit.MetricDefinition) *transit.MonitoredResource {
	hostStat, err := host.Info()
	if err != nil {
		log.Error(err)
		return nil
	}

	hostName = hostStat.Hostname

	monitoredResource, _ := connectors.CreateResource(hostName)

	var notDefaultProcesses []transit.MetricDefinition
	for _, pr := range processes {
		// temporary solution, will be removed
		if pr.Name == templateMetricName {
			continue
		}
		if function, exists := processToFuncMap[pr.Name]; exists {
			monitoredService := function.(func(int, int, string) *transit.MonitoredService)(pr.WarningThreshold, pr.CriticalThreshold, pr.CustomName)
			if monitoredService != nil {
				monitoredResource.Services = append(monitoredResource.Services, *monitoredService)
			}
		} else {
			notDefaultProcesses = append(notDefaultProcesses, pr)
		}
	}

	processesMap := collectMonitoredProcesses(notDefaultProcesses)
	interval := time.Now()

	for processName, processValues := range processesMap {
		metricBuilder := connectors.MetricBuilder{
			Name:           processName,
			Value:          processValues.value,
			UnitType:       transit.PercentCPU,
			Warning:        int64(processValues.warningValue),
			Critical:       int64(processValues.criticalValue),
			StartTimestamp: &milliseconds.MillisecondTimestamp{Time: interval},
			EndTimestamp:   &milliseconds.MillisecondTimestamp{Time: interval},
		}
		monitoredService, err := connectors.BuildServiceForMetric(hostName, metricBuilder)
		if err != nil {
			log.Error("Error when creating service ", hostName, ":", processName)
			log.Error(err)
			continue
		}

		if processValues.value == -1 {
			monitoredService.Status = transit.ServicePending
		}

		monitoredResource.Services = append(monitoredResource.Services, *monitoredService)
	}

	updateCache()

	return monitoredResource
}

func getTotalDiskUsageService(warningThresholdValue int, criticalThresholdValue int, customName string) *transit.MonitoredService {
	interval := time.Now()
	diskStats, err := disk.Usage("/")
	if err != nil {
		log.Error(err)
		return nil
	}

	metricBuilder := connectors.MetricBuilder{
		Name:           TotalDiskAllocatedServiceName,
		CustomName:     customName,
		Value:          int64(diskStats.Total / MB),
		UnitType:       transit.MB,
		Warning:        int64(warningThresholdValue),
		Critical:       int64(criticalThresholdValue),
		StartTimestamp: &milliseconds.MillisecondTimestamp{Time: interval},
		EndTimestamp:   &milliseconds.MillisecondTimestamp{Time: interval},
	}

	service, err := connectors.BuildServiceForMetric(hostName, metricBuilder)
	if err != nil {
		log.Error("Error when creating service ", hostName, ":", connectors.Name(metricBuilder.Name, metricBuilder.CustomName))
		log.Error(err)
		return nil
	}
	return service
}

func getDiskUsedService(warningThresholdValue int, criticalThresholdValue int, customName string) *transit.MonitoredService {
	interval := time.Now()
	diskStats, err := disk.Usage("/")
	if err != nil {
		log.Error(err)
		return nil
	}

	metricBuilder := connectors.MetricBuilder{
		Name:           DiskUsedServiceName,
		CustomName:     customName,
		Value:          int64(diskStats.Used / MB),
		UnitType:       transit.MB,
		Warning:        int64(warningThresholdValue),
		Critical:       int64(criticalThresholdValue),
		StartTimestamp: &milliseconds.MillisecondTimestamp{Time: interval},
		EndTimestamp:   &milliseconds.MillisecondTimestamp{Time: interval},
	}

	service, err := connectors.BuildServiceForMetric(hostName, metricBuilder)
	if err != nil {
		log.Error("Error when creating service ", hostName, ":", connectors.Name(metricBuilder.Name, metricBuilder.CustomName))
		log.Error(err)
		return nil
	}
	return service
}

func getDiskFreeService(warningThresholdValue int, criticalThresholdValue int, customName string) *transit.MonitoredService {
	interval := time.Now()
	diskStats, err := disk.Usage("/")
	if err != nil {
		log.Error(err.Error())
		return nil
	}

	metricBuilder := connectors.MetricBuilder{
		Name:           DiskFreeServiceName,
		CustomName:     customName,
		Value:          int64(diskStats.Free / MB),
		UnitType:       transit.MB,
		Warning:        int64(warningThresholdValue),
		Critical:       int64(criticalThresholdValue),
		StartTimestamp: &milliseconds.MillisecondTimestamp{Time: interval},
		EndTimestamp:   &milliseconds.MillisecondTimestamp{Time: interval},
	}

	service, err := connectors.BuildServiceForMetric(hostName, metricBuilder)
	if err != nil {
		log.Error("Error when creating service ", hostName, ":", connectors.Name(metricBuilder.Name, metricBuilder.CustomName))
		log.Error(err)
		return nil
	}
	return service
}

func getTotalMemoryUsageService(warningThresholdValue int, criticalThresholdValue int, customName string) *transit.MonitoredService {
	interval := time.Now()
	vmStats, err := mem.VirtualMemory()
	if err != nil {
		log.Error(err.Error())
		return nil
	}

	metricBuilder := connectors.MetricBuilder{
		Name:           TotalMemoryUsageAllocatedName,
		CustomName:     customName,
		Value:          int64(vmStats.Total / MB),
		UnitType:       transit.MB,
		Warning:        int64(warningThresholdValue),
		Critical:       int64(criticalThresholdValue),
		StartTimestamp: &milliseconds.MillisecondTimestamp{Time: interval},
		EndTimestamp:   &milliseconds.MillisecondTimestamp{Time: interval},
	}

	service, err := connectors.BuildServiceForMetric(hostName, metricBuilder)
	if err != nil {
		log.Error("Error when creating service ", hostName, ":", connectors.Name(metricBuilder.Name, metricBuilder.CustomName))
		log.Error(err)
		return nil
	}
	return service
}

func getMemoryUsedService(warningThresholdValue int, criticalThresholdValue int, customName string) *transit.MonitoredService {
	interval := time.Now()
	vmStats, err := mem.VirtualMemory()
	if err != nil {
		log.Error(err.Error())
		return nil
	}

	metricBuilder := connectors.MetricBuilder{
		Name:           MemoryUsedServiceName,
		CustomName:     customName,
		Value:          int64(vmStats.Used / MB),
		UnitType:       transit.MB,
		Warning:        int64(warningThresholdValue),
		Critical:       int64(criticalThresholdValue),
		StartTimestamp: &milliseconds.MillisecondTimestamp{Time: interval},
		EndTimestamp:   &milliseconds.MillisecondTimestamp{Time: interval},
	}

	service, err := connectors.BuildServiceForMetric(hostName, metricBuilder)
	if err != nil {
		log.Error("Error when creating service ", hostName, ":", connectors.Name(metricBuilder.Name, metricBuilder.CustomName))
		log.Error(err)
		return nil
	}
	return service
}

func getMemoryFreeService(warningThresholdValue int, criticalThresholdValue int, customName string) *transit.MonitoredService {
	interval := time.Now()
	vmStats, err := mem.VirtualMemory()
	if err != nil {
		log.Error(err.Error())
		return nil
	}

	metricBuilder := connectors.MetricBuilder{
		Name:           MemoryFreeServiceName,
		CustomName:     customName,
		Value:          int64(vmStats.Free / MB),
		UnitType:       transit.MB,
		Warning:        int64(warningThresholdValue),
		Critical:       int64(criticalThresholdValue),
		StartTimestamp: &milliseconds.MillisecondTimestamp{Time: interval},
		EndTimestamp:   &milliseconds.MillisecondTimestamp{Time: interval},
	}

	service, err := connectors.BuildServiceForMetric(hostName, metricBuilder)
	if err != nil {
		log.Error("Error when creating service ", hostName, ":", connectors.Name(metricBuilder.Name, metricBuilder.CustomName))
		log.Error(err)
		return nil
	}
	return service
}

func getNumberOfProcessesService(warningThresholdValue int, criticalThresholdValue int, customName string) *transit.MonitoredService {
	interval := time.Now()
	hostStat, err := host.Info()
	if err != nil {
		log.Error(err)
		return nil
	}

	metricBuilder := connectors.MetricBuilder{
		Name:           ProcessesNumberServiceName,
		CustomName:     customName,
		Value:          int64(hostStat.Procs),
		UnitType:       transit.UnitCounter,
		Warning:        int64(warningThresholdValue),
		Critical:       int64(criticalThresholdValue),
		StartTimestamp: &milliseconds.MillisecondTimestamp{Time: interval},
		EndTimestamp:   &milliseconds.MillisecondTimestamp{Time: interval},
	}

	service, err := connectors.BuildServiceForMetric(hostName, metricBuilder)
	if err != nil {
		log.Error("Error when creating service ", hostName, ":", connectors.Name(metricBuilder.Name, metricBuilder.CustomName))
		log.Error(err)
		return nil
	}
	return service
}

func getTotalCPUUsage(warningThresholdValue int, criticalThresholdValue int, customName string) *transit.MonitoredService {
	interval := time.Now()
	metricBuilder := connectors.MetricBuilder{
		Name:           TotalCPUUsageServiceName,
		CustomName:     customName,
		Value:          getCPUUsage(),
		UnitType:       transit.PercentCPU,
		Warning:        int64(warningThresholdValue),
		Critical:       int64(criticalThresholdValue),
		StartTimestamp: &milliseconds.MillisecondTimestamp{Time: interval},
		EndTimestamp:   &milliseconds.MillisecondTimestamp{Time: interval},
	}

	service, err := connectors.BuildServiceForMetric(hostName, metricBuilder)
	if err != nil {
		log.Error("Error when creating service ", hostName, ":", connectors.Name(metricBuilder.Name, metricBuilder.CustomName))
		log.Error(err)
		return nil
	}
	return service
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
	value         float64
	computeType   transit.ComputeType
	expression    string
	criticalValue int
	warningValue  int
}

// Collects a map of process names to cpu usage, given a list of processes to be monitored
func collectMonitoredProcesses(monitoredProcesses []transit.MetricDefinition) map[string]values {
	if len(monitoredProcesses) == 0 {
		return make(map[string]values)
	}
	processes := make([]*localProcess, 0)
	hostProcesses, _ := process.Processes()
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
				value:         m[pr.Name],
				criticalValue: pr.CriticalThreshold,
				warningValue:  pr.WarningThreshold,
				computeType:   pr.ComputeType,
				expression:    "",
			}
		} else {
			processesMap[name] = values{
				value:         -1,
				criticalValue: pr.CriticalThreshold,
				warningValue:  pr.WarningThreshold,
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
	for n, _ := range hostProcesses.(map[string]float64) {
		if strings.Contains(n, name) {
			processes = append(processes, n)
		}
	}

	return processes
}

func collectProcesses() map[string]float64 {
	processes := make(map[string]float64)
	hostProcesses, _ := process.Processes()

	for _, hostProcess := range hostProcesses {
		cpuUsed, _ := hostProcess.CPUPercent()
		name, _ := hostProcess.Name()
		processes[strings.ReplaceAll(name, ".", "_")] = cpuUsed
	}

	for name, function := range processToFuncMap {
		monitoredService := function.(func(int, int, string) *transit.MonitoredService)(-1, -1, "")
		if monitoredService != nil {
			if monitoredService.Metrics[0].Value.ValueType == transit.DoubleType {
				processes[strings.ReplaceAll(name, ".", "_")] = monitoredService.Metrics[0].Value.DoubleValue
			}
			if monitoredService.Metrics[0].Value.ValueType == transit.IntegerType {
				processes[strings.ReplaceAll(name, ".", "_")] = float64(monitoredService.Metrics[0].Value.IntegerValue)
			}
		}
	}

	return processes
}

func updateCache() {
	cache.ProcessesCache.SetDefault("processes", collectProcesses())
}

// initializeEntrypoints - function for setting entrypoints,
// that will be available through the Server Connector API
func initializeEntrypoints() []services.Entrypoint {
	var entrypoints []services.Entrypoint

	entrypoints = append(entrypoints,
		services.Entrypoint{
			Url:    "/suggest/:viewName/:name",
			Method: "Get",
			Handler: func(c *gin.Context) {
				if c.Param("viewName") == string(transit.Process) {
					c.JSON(http.StatusOK, listSuggestions(c.Param("name")))
				} else {
					c.JSON(http.StatusOK, []transit.MetricDefinition{})
				}
			},
		},
		services.Entrypoint{
			Url:    "/expressions/suggest/:name",
			Method: "Get",
			Handler: func(c *gin.Context) {
				c.JSON(http.StatusOK, connectors.ListExpressions(c.Param("name")))
			},
		},
		services.Entrypoint{
			Url:    "/expressions/evaluate",
			Method: "Post",
			Handler: func(c *gin.Context) {
				var expression connectors.ExpressionToEvaluate
				body, err := c.GetRawData()
				if err != nil {
					c.JSON(http.StatusBadRequest, err.Error())
					return
				}
				err = json.Unmarshal(body, &expression)
				if err != nil {
					c.JSON(http.StatusBadRequest, err.Error())
					return
				}
				result, err := connectors.EvaluateExpression(expression, c.Request.URL.Query().Get("override") == "true")
				if err == nil {
					c.JSON(http.StatusOK, result)
					return
				}
				log.Error("[Server Connector]: " + err.Error())
				c.IndentedJSON(http.StatusBadRequest, err.Error())
			},
		},
		services.Entrypoint{
			Url:    "/version",
			Method: "Get",
			Handler: func(c *gin.Context) {
				c.JSON(http.StatusOK, connectors.Version{Number: serverConnectorVersion,
					BuildTimestamp: strings.ReplaceAll(buildTime, "_", " ")})
			},
		})

	return entrypoints
}
