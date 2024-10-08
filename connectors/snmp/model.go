package snmp

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/gwos/tcg/connectors"
	"github.com/gwos/tcg/connectors/snmp/clients"
	"github.com/gwos/tcg/connectors/snmp/utils"
	"github.com/gwos/tcg/sdk/transit"
	"github.com/patrickmn/go-cache"
	"github.com/rs/zerolog/log"
)

// FiveMinutes NeDi interval in seconds
const FiveMinutes = 300

// PreviousValueCache cache to handle "Delta" metrics
var previousValueCache = cache.New(-1, -1)

type MonitoringState struct {
	sync.Mutex
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
	Key   string
	Mib   string
	Value int64
}

func (state *MonitoringState) Init() {
	state.devices = make(map[string]DeviceExt)
}

func (state *MonitoringState) retrieveMonitoredResources(metricDefinitions map[string]transit.MetricDefinition) []transit.MonitoredResource {
	mResources := make([]transit.MonitoredResource, 0, len(state.devices))
	for _, device := range state.devices {
		mServices := device.retrieveMonitoredServices(metricDefinitions)
		mResource, err := connectors.CreateResource(device.Name, mServices)
		if err != nil {
			log.Err(err).Msgf("could not create monitored resource '%s'", device.Name)
			continue
		}
		mResource.Status = calculateHostStatus(device.LastOK)
		mResources = append(mResources, *mResource)
	}

	// log.Debug().
	// 	Interface("devices", state.devices).
	// 	Interface("metricDefinitions", metricDefinitions).
	// 	Interface("mResources", mResources).
	// 	Msg("__ retrieveMonitoredResources")

	return mResources
}

func (device *DeviceExt) retrieveMonitoredServices(metricDefinitions map[string]transit.MetricDefinition) []transit.MonitoredService {
	mServices := make([]transit.MonitoredService, 0, len(device.Interfaces))

	if metricDefinitions == nil {
		return mServices
	}

	timestamp := transit.NewTimestamp()
	for _, iFace := range device.Interfaces {
		var bytesInPrev, bytesOutPrev, bytesInX64Prev, bytesOutX64Prev int64 = -1, -1, -1, -1
		if val, ok := previousValueCache.Get(fmt.Sprintf("%s:%s:%s", device.Name, iFace.Name, clients.IfInOctets)); ok {
			bytesInPrev = val.(int64)
		}
		if val, ok := previousValueCache.Get(fmt.Sprintf("%s:%s:%s", device.Name, iFace.Name, clients.IfOutOctets)); ok {
			bytesOutPrev = val.(int64)
		}
		if val, ok := previousValueCache.Get(fmt.Sprintf("%s:%s:%s", device.Name, iFace.Name, clients.IfHCInOctets)); ok {
			bytesInX64Prev = val.(int64)
		}
		if val, ok := previousValueCache.Get(fmt.Sprintf("%s:%s:%s", device.Name, iFace.Name, clients.IfHCOutOctets)); ok {
			bytesOutX64Prev = val.(int64)
		}

		var metricsBuilder []connectors.MetricBuilder
		for mib, metric := range iFace.Metrics {
			if metricDefinition, has := metricDefinitions[metric.Mib]; has {
				var unitType transit.UnitType
				var value interface{}

				unitType = transit.UnitCounter
				value = metric.Value

				switch mib {
				case clients.IfInOctets, clients.IfOutOctets, clients.IfHCInOctets, clients.IfHCOutOctets:
					value = metric.Value * 8
				}

				metricBuilder := connectors.MetricBuilder{
					Name:           metric.Key,
					CustomName:     metricDefinition.CustomName,
					ComputeType:    metricDefinition.ComputeType,
					Expression:     metricDefinition.Expression,
					UnitType:       unitType,
					Warning:        metricDefinition.WarningThreshold,
					Critical:       metricDefinition.CriticalThreshold,
					StartTimestamp: timestamp,
					EndTimestamp:   timestamp,
					Graphed:        metricDefinition.Graphed,

					Value: nil,
				}

				ck := fmt.Sprintf("%s:%s:%s", device.Name, iFace.Name, mib)
				isDelta, isPreviousPresent, valueToSet := calculateValue(metricDefinition.MetricType, unitType,
					ck, value)

				if !isDelta || (isDelta && isPreviousPresent) {
					metricBuilder.Value = valueToSet
					metricsBuilder = append(metricsBuilder, metricBuilder)
				}

				previousValueCache.SetDefault(ck, metric.Value)

				// log.Debug().
				// 	Interface("_ck", ck).
				// 	Interface("_isDelta", isDelta).
				// 	Interface("_isPreviousPresent", isPreviousPresent).
				// 	Interface("_valueToSet", valueToSet).
				// 	Interface("metricsBuilder", metricsBuilder).
				// 	Msg("__ ck")
			}
		}

		for key := range clients.NonMibMetrics {
			if metricDefinition, has := metricDefinitions[key]; has {
				switch key {
				case clients.BytesPerSecondIn:
					metricBuilder := calculateBytesPerSecond(key, metricDefinition,
						iFace.Metrics[clients.IfInOctets].Value*8, iFace.Metrics[clients.IfHCInOctets].Value*8, bytesInPrev, bytesInX64Prev, timestamp)
					metricsBuilder = append(metricsBuilder, metricBuilder)
				case clients.BytesPerSecondOut:
					metricBuilder := calculateBytesPerSecond(key, metricDefinition,
						iFace.Metrics[clients.IfOutOctets].Value*8, iFace.Metrics[clients.IfHCOutOctets].Value*8, bytesOutPrev, bytesOutX64Prev, timestamp)
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
				mService.LastPluginOutput = "Interface Operational State is DOWN, Administrative state is DOWN"
			case 1:
				mService.Status = transit.ServiceUnscheduledCritical
				mService.LastPluginOutput = "Interface Operational State is DOWN, Administrative state is UP"
			case 2:
				mService.Status = transit.ServiceWarning
				mService.LastPluginOutput = "Interface Operational State is UP, Administrative state is DOWN"
			case 3:
				mService.Status = transit.ServiceOk
				mService.LastPluginOutput = "Interface Operational State is UP, Administrative state is UP"
			case -1:
			}
			mServices = append(mServices, *mService)
		}
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
				previousValueCache.SetDefault(metricName, currentValue.(int64))
				currentValue = currentValue.(int64) - previousValue.(int64)
			}
			return true, true, currentValue.(int64)
		}
		return true, false, currentValue.(int64)
	}
	return false, false, currentValue.(int64)
}

func calculateBytesPerSecond(metricName string, metricDefinition transit.MetricDefinition, current, currentX64, previous,
	previousX64 int64, timestamp *transit.Timestamp) connectors.MetricBuilder {
	seconds := int(connectors.CheckInterval.Seconds())
	result := (current - previous) / int64(seconds)
	if currentX64 > 0 && previousX64 > 0 {
		result = (currentX64 - previousX64) / int64(seconds)
	}

	return connectors.MetricBuilder{
		Name:           metricName,
		CustomName:     metricDefinition.CustomName,
		ComputeType:    metricDefinition.ComputeType,
		Expression:     metricDefinition.Expression,
		UnitType:       transit.UnitCounter,
		Warning:        metricDefinition.WarningThreshold,
		Critical:       metricDefinition.CriticalThreshold,
		StartTimestamp: timestamp,
		EndTimestamp:   timestamp,
		Graphed:        metricDefinition.Graphed,

		Value: result,
	}
}
