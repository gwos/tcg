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

type cachedMetric struct {
	InterfaceMetric
	ts int64
}

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
	ts := timestamp.Unix()
	for _, iFace := range device.Interfaces {
		var metricsBuilder []connectors.MetricBuilder

		for key := range clients.NonMibMetrics {
			if metricDefinition, has := metricDefinitions[key]; has {
				metricBuilder := makeMetricBuilder(metricDefinition, key, timestamp)

				switch key {
				case clients.BytesPerSecondIn:
					if prev, ok := previousValueCache.Get(makeCK(device.Name, iFace.Name, clients.IfHCInOctets)); ok {
						prev := prev.(cachedMetric)
						v := (iFace.Metrics[clients.IfHCInOctets].Value - prev.Value) / (ts - prev.ts)
						metricBuilder.Value = v
						metricsBuilder = append(metricsBuilder, metricBuilder)
					} else if prev, ok := previousValueCache.Get(makeCK(device.Name, iFace.Name, clients.IfInOctets)); ok {
						prev := prev.(cachedMetric)
						v := (iFace.Metrics[clients.IfInOctets].Value - prev.Value) / (ts - prev.ts)
						metricBuilder.Value = v
						metricsBuilder = append(metricsBuilder, metricBuilder)
					}

				case clients.BytesPerSecondOut:
					if prev, ok := previousValueCache.Get(makeCK(device.Name, iFace.Name, clients.IfHCOutOctets)); ok {
						prev := prev.(cachedMetric)
						v := (iFace.Metrics[clients.IfHCOutOctets].Value - prev.Value) / (ts - prev.ts)
						metricBuilder.Value = v
						metricsBuilder = append(metricsBuilder, metricBuilder)
					} else if prev, ok := previousValueCache.Get(makeCK(device.Name, iFace.Name, clients.IfOutOctets)); ok {
						prev := prev.(cachedMetric)
						v := (iFace.Metrics[clients.IfOutOctets].Value - prev.Value) / (ts - prev.ts)
						metricBuilder.Value = v
						metricsBuilder = append(metricsBuilder, metricBuilder)
					}
				}
			}
		}

		for mib, metric := range iFace.Metrics {
			if metricDefinition, has := metricDefinitions[metric.Mib]; has {
				metricBuilder := makeMetricBuilder(metricDefinition, metric.Key, timestamp)

				ck := makeCK(device.Name, iFace.Name, mib)
				if isDelta(metricDefinition.MetricType) {
					if prev, ok := previousValueCache.Get(ck); ok {
						metricBuilder.Value = metric.Value - prev.(cachedMetric).Value
						metricsBuilder = append(metricsBuilder, metricBuilder)
					}
				} else {
					metricBuilder.Value = metric.Value
					metricsBuilder = append(metricsBuilder, metricBuilder)
				}

				previousValueCache.SetDefault(ck, cachedMetric{metric, ts})

				// log.Debug().
				// 	Interface("_ck", ck).
				// 	Interface("_isDelta", isDelta).
				// 	Interface("_isPreviousPresent", isPreviousPresent).
				// 	Interface("_valueToSet", valueToSet).
				// 	Interface("metricsBuilder", metricsBuilder).
				// 	Msg("__ ck")
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

func calculateHostStatus(lastOk int64) transit.MonitorStatus {
	now := time.Now().Unix() // in seconds
	if (now - lastOk) < FiveMinutes {
		return transit.HostUp
	}
	return transit.HostUnreachable
}

func isDelta(k transit.MetricKind) bool {
	return strings.EqualFold(string(k), string(transit.Delta))
}

func makeCK(deviceName, iFaceName, mib string) string {
	return fmt.Sprintf("%s:%s:%s", deviceName, iFaceName, mib)
}

func makeMetricBuilder(metricDefinition transit.MetricDefinition, metricName string, timestamp *transit.Timestamp) connectors.MetricBuilder {
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
	}
}

/**
## [Consider SNMP Counters: Frequently Asked Questions](https://www.cisco.com/c/en/us/support/docs/ip/simple-network-management-protocol-snmp/26007-faq-snmpcounter.html)

Q. When do 64-bit counters be used?
	A. [RFC 2233](https://www.ietf.org/rfc/rfc2233.txt)  adopted expanded 64-bit counters for high capacity interfaces in which 32-bit counters do not provide enough capacity and wrap too fast.

> [!Note]
> Cisco IOS Software does not support 64-bit counters for interface speeds of less than 20 Mbps.
> This means that 64-bit counters are not supported on 10 Mb Ethernet ports. Only 100 Mb Fast-Ethernet and other high speed ports support 64-bit counters.
*/
