package main

import (
	"github.com/gwos/tcg/connectors"
	"github.com/gwos/tcg/connectors/snmp-connector/clients"
	"github.com/gwos/tcg/connectors/snmp-connector/utils"
	"github.com/gwos/tcg/log"
	"github.com/gwos/tcg/milliseconds"
	"github.com/gwos/tcg/transit"
	"time"
)

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

func (state *MonitoringState) retrieveMonitoredResources(metricDefinitions map[string]transit.MetricDefinition) []transit.DynamicMonitoredResource {
	mResources := make([]transit.DynamicMonitoredResource, len(state.devices))
	i := 0
	for _, device := range state.devices {
		mServices := device.retrieveMonitoredServices(metricDefinitions)
		mResource, err := connectors.CreateResource(device.Name, mServices)
		if err != nil {
			log.Error("|snmpConnectorModel.go| : [retrieveMonitoredResources] : Error when create monitored resource '", device.Name,
				"'. Reason: ", err)
		}
		if mResource != nil {
			mResources[i] = *mResource
		}
		i++
	}
	return mResources
}

func (device *DeviceExt) retrieveMonitoredServices(metricDefinitions map[string]transit.MetricDefinition) []transit.DynamicMonitoredService {
	mServices := make([]transit.DynamicMonitoredService, len(device.Interfaces))

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
					log.Warn("|snmpConnectorModel.go| : [retrieveMonitoredServices] : Unable to process metric '",
						metricName, "' for interface '", iFace.Name, "' of device '", device.Name,
						"': unsupported unit type '", metric.UnitType, "'. Skipping.")
					continue
				}

				metricBuilder := connectors.MetricBuilder{
					Name:           metricName,
					CustomName:     metricDefinition.CustomName,
					ComputeType:    metricDefinition.ComputeType,
					Expression:     metricDefinition.Expression,
					Value:          value,
					UnitType:       unitType,
					Warning:        metricDefinition.WarningThreshold,
					Critical:       metricDefinition.CriticalThreshold,
					StartTimestamp: &milliseconds.MillisecondTimestamp{Time: interval},
					EndTimestamp:   &milliseconds.MillisecondTimestamp{Time: interval},
					Graphed:        metricDefinition.Graphed,
				}
				metricsBuilder = append(metricsBuilder, metricBuilder)
			}
		}

		mService, err := connectors.BuildServiceForMetrics(iFace.Name, device.Name, metricsBuilder)
		if err != nil {
			log.Error("|snmpConnectorModel.go| : [retrieveMonitoredServices] : Error when create monitored service '",
				device.Name, ":", iFace.Name, "'. Reason: ", err)
		}
		if mService != nil {
			mServices[i] = *mService
		}
		i++
	}

	return mServices
}
