package main

import (
	"fmt"
	"strings"
	"time"

	"github.com/gwos/tcg/connectors"
	"github.com/gwos/tcg/connectors/snmp-connector/clients"
	"github.com/gwos/tcg/connectors/snmp-connector/utils"
	"github.com/gwos/tcg/sdk/milliseconds"
	"github.com/gwos/tcg/sdk/transit"
	"github.com/patrickmn/go-cache"
	"github.com/rs/zerolog/log"
)

// FiveMinutes NeDi interval in seconds
const FiveMinutes = 300

// PreviousValueCache cache to handle "Delta" metrics
var previousValueCache = cache.New(-1, -1)

type MonitoringState struct {
	// [deviceName]Device
	devices map[string]DeviceExt
}

type DeviceExt struct {
	clients.Device
	SecData *utils.SecurityData
	// [ifIdx]Interface
	Interfaces map[int]InterfaceExt
}

type InterfaceExt struct {
	clients.Interface
	// [mib]InterfaceMetric
	Metrics map[string]InterfaceMetric
}

type InterfaceMetric struct {
	Mib      string
	Value    int
	UnitType clients.SnmpUnitType
}

func (state *MonitoringState) Init() {
	state.devices = make(map[string]DeviceExt)
}

func (state *MonitoringState) retrieveMonitoredResources(metricDefinitions map[string]transit.MetricDefinition) []transit.MonitoredResource {
	mResources := make([]transit.MonitoredResource, len(state.devices))
	i := 0
	for _, device := range state.devices {
		mServices := device.retrieveMonitoredServices(metricDefinitions)
		mResource, err := connectors.CreateResource(device.Name, mServices)
		if err != nil {
			log.Err(err).Msgf("could not create monitored resource '%s'", device.Name)
		}
		mResource.Status = calculateHostStatus(device.LastOK)
		if mResource != nil {
			mResources[i] = *mResource
		}
		i++
	}
	return mResources
}

func (device *DeviceExt) retrieveMonitoredServices(metricDefinitions map[string]transit.MetricDefinition) []transit.MonitoredService {
	mServices := make([]transit.MonitoredService, len(device.Interfaces))

	if metricDefinitions == nil {
		return mServices
	}

	interval := time.Now()
	i := 0
	for _, iFace := range device.Interfaces {
		var metricsBuilder []connectors.MetricBuilder

		for metricName, metric := range iFace.Metrics {
			if metricDefinition, has := metricDefinitions[metricName]; has {
				var unitType transit.UnitType
				var value interface{}

				switch metric.UnitType {
				case clients.Number:
					unitType = transit.UnitCounter
					value = metric.Value
					break
				case clients.Bit:
					unitType = transit.MB
					value = float64(metric.Value) / 8000000
					break
				default:
					log.Warn().Msgf("could not process metric '%s' for interface '%s' of device '%s': unsupported unit type '%s': skipping",
						metricName, iFace.Name, device.Name, metric.UnitType)
					continue
				}

				metricBuilder := connectors.MetricBuilder{
					Name:           metricName,
					CustomName:     metricDefinition.CustomName,
					ComputeType:    metricDefinition.ComputeType,
					Expression:     metricDefinition.Expression,
					UnitType:       unitType,
					Warning:        metricDefinition.WarningThreshold,
					Critical:       metricDefinition.CriticalThreshold,
					StartTimestamp: &milliseconds.MillisecondTimestamp{Time: interval},
					EndTimestamp:   &milliseconds.MillisecondTimestamp{Time: interval},
					Graphed:        metricDefinition.Graphed,

					Value: nil,
				}

				isDelta, isPreviousPresent, valueToSet := calculateValue(metricDefinition.MetricType, unitType,
					fmt.Sprintf("%s:%s:%s", device.Name, iFace.Name, metricName), value)

				if !isDelta || (isDelta && isPreviousPresent) {
					metricBuilder.Value = valueToSet
					metricsBuilder = append(metricsBuilder, metricBuilder)
				}
			}
		}

		mService, err := connectors.BuildServiceForMetrics(iFace.Name, device.Name, metricsBuilder)
		if err != nil {
			log.Err(err).Msgf("could not create monitored service '%s:%s'", device.Name, iFace.Name)
		}
		if mService != nil {
			switch iFace.Status {
			case 0:
				mService.Status = transit.ServiceWarning
				mService.LastPlugInOutput = "Interface Operational State is DOWN, Administrative state is DOWN"
			case 1:
				mService.Status = transit.ServiceUnscheduledCritical
				mService.LastPlugInOutput = "Interface Operational State is DOWN, Administrative state is UP"
			case 2:
				mService.Status = transit.ServiceWarning
				mService.LastPlugInOutput = "Interface Operational State is UP, Administrative state is DOWN"
			case 3:
				mService.Status = transit.ServiceOk
				mService.LastPlugInOutput = "Interface Operational State is UP, Administrative state is UP"
			case -1:
			}
			mServices[i] = *mService
		}
		i++
	}

	return mServices
}

func calculateHostStatus(lastOk float64) transit.MonitorStatus {
	now := time.Now().Unix() // in seconds
	if (float64(now) - lastOk) < FiveMinutes {
		return transit.HostUp
	}

	return transit.HostUnreachable
}

func calculateValue(metricKind transit.MetricKind, unitType transit.UnitType,
	metricName string, currentValue interface{}) (bool, bool, interface{}) {
	if strings.EqualFold(string(metricKind), string(transit.Delta)) {
		if previousValue, present := previousValueCache.Get(metricName); present {
			switch unitType {
			case transit.UnitCounter:
				previousValueCache.SetDefault(metricName, float64(currentValue.(int)))
				currentValue = int(float64(currentValue.(int)) - previousValue.(float64))
			case transit.MB:
				previousValueCache.SetDefault(metricName, currentValue.(float64))
				currentValue = currentValue.(float64) - previousValue.(float64)
			}
			return true, true, currentValue
		} else {
			switch unitType {
			case transit.UnitCounter:
				previousValueCache.SetDefault(metricName, float64(currentValue.(int)))
			case transit.MB:
				previousValueCache.SetDefault(metricName, currentValue.(float64))
			}
			return true, false, currentValue
		}
	}
	return false, false, currentValue
}
