package snmp

import (
	"cmp"
	"encoding/json"
	"slices"
	"testing"
	"time"

	"github.com/gwos/tcg/sdk/transit"
	"github.com/stretchr/testify/assert"
)

func TestRetrieveMonitoredResources(t *testing.T) {
	state, metricDefinitions, monitoredResourcesExp, err := seedTest()
	if err != nil {
		t.FailNow()
	}

	// 1st call for set previousValueCache
	_ = state.retrieveMonitoredResources(metricDefinitions)

	for k, v := range previousValueCache.Items() {
		prev := v.Object.(cachedMetric)
		prev.ts, prev.Value = prev.ts-60, prev.Value-10
		previousValueCache.SetDefault(k, prev)
	}

	// 2nd call for deltas
	monitoredResources := state.retrieveMonitoredResources(metricDefinitions)
	ts := transit.NewTimestamp()
	normalizeData(monitoredResources, ts)
	normalizeData(monitoredResourcesExp, ts)
	// assert.Equal(t, monitoredResourcesExp, monitoredResources)
	monitoredResourcesJSON, _ := json.Marshal(monitoredResources)
	monitoredResourcesExpJSON, _ := json.Marshal(monitoredResourcesExp)

	println(`--`)
	println(string(monitoredResourcesJSON))
	println(`--`)
	assert.JSONEq(t, string(monitoredResourcesExpJSON), string(monitoredResourcesJSON))
}

func normalizeData(mr []transit.MonitoredResource, ts *transit.Timestamp) {
	for i := range mr {
		for j := range mr[i].Services {
			for k := range mr[i].Services[j].Metrics {
				mr[i].Services[j].Metrics[k].Interval.StartTime = ts
				mr[i].Services[j].Metrics[k].Interval.EndTime = ts
			}
			slices.SortFunc(mr[i].Services[j].Metrics, func(a, b transit.TimeSeries) int {
				return cmp.Compare(a.MetricName, b.MetricName)
			})
			mr[i].Services[j].LastCheckTime = ts
			mr[i].Services[j].NextCheckTime = ts
		}
		slices.SortFunc(mr[i].Services, func(a, b transit.MonitoredService) int {
			return cmp.Compare(a.Name, b.Name)
		})
		mr[i].LastCheckTime = ts
		mr[i].NextCheckTime = ts
	}
	slices.SortFunc(mr, func(a, b transit.MonitoredResource) int {
		return cmp.Compare(a.Name, b.Name)
	})
}

// data from 8.7.0
func seedTest() (*MonitoringState, map[string]transit.MetricDefinition, []transit.MonitoredResource, error) {
	metricDefinitionsJSON := []byte(`
{
	"ifHCInOctets": {
		"computeType": "Query",
		"criticalThreshold": -1,
		"customName": "ifHCInOctets",
		"description": "ifHCInOctets",
		"graphed": true,
		"metricType": "Delta",
		"monitored": true,
		"name": "ifHCInOctets",
		"serviceType": "interfaces",
		"warningThreshold": -1
	},
	"ifHCOutOctets": {
		"computeType": "Query",
		"criticalThreshold": -1,
		"customName": "ifHCOutOctets",
		"graphed": true,
		"metricType": "Delta",
		"monitored": true,
		"name": "ifHCOutOctets",
		"serviceType": "interfaces",
		"warningThreshold": -1
	},
	"ifInDiscards": {
		"computeType": "Query",
		"criticalThreshold": -1,
		"customName": "Inbound Discards",
		"description": "The number of inbound packets which were chosen to be discarded even though no errors had been detected to prevent their being deliverable to a higher-layer protocol.",
		"graphed": true,
		"metricType": "Delta",
		"monitored": true,
		"name": "ifInDiscards",
		"serviceType": "interfaces",
		"sourceType": "interfaces",
		"warningThreshold": -1
	},
	"ifInErrors": {
		"computeType": "Query",
		"criticalThreshold": -1,
		"customName": "Inbound Errors",
		"description": "For packet-oriented interfaces, the number of outbound packets that could not be transmitted because of errors. For character-oriented or fixed-length interfaces, the number of outbound transmission units that could not be transmitted because of errors.",
		"graphed": true,
		"metricType": "Delta",
		"monitored": true,
		"name": "ifInErrors",
		"serviceType": "interfaces",
		"sourceType": "interfaces",
		"warningThreshold": -1
	},
	"ifInOctets": {
		"computeType": "Query",
		"criticalThreshold": -1,
		"customName": "Inbound Octets",
		"description": "The total number of octets received on the interface, including framing characters.",
		"graphed": true,
		"metricType": "Delta",
		"monitored": true,
		"name": "ifInOctets",
		"serviceType": "interfaces",
		"sourceType": "interfaces",
		"warningThreshold": -1
	},
	"ifOutDiscards": {
		"computeType": "Query",
		"criticalThreshold": -1,
		"customName": "Outbound Discards",
		"description": "The number of outbound packets which were chosen to be discarded even though no errors had been detected to prevent their being transmitted.",
		"graphed": true,
		"metricType": "Delta",
		"monitored": true,
		"name": "ifOutDiscards",
		"serviceType": "interfaces",
		"sourceType": "interfaces",
		"warningThreshold": -1
	},
	"ifOutErrors": {
		"computeType": "Query",
		"criticalThreshold": -1,
		"customName": "Outbound Errors",
		"description": "For packet-oriented interfaces, the number of outbound packets that could not be transmitted because of errors. For character-oriented or fixed-length interfaces, the number of outbound transmission units that could not be transmitted because of errors.",
		"graphed": true,
		"metricType": "Delta",
		"monitored": true,
		"name": "ifOutErrors",
		"serviceType": "interfaces",
		"sourceType": "interfaces",
		"warningThreshold": -1
	},
	"ifOutOctets": {
		"computeType": "Query",
		"criticalThreshold": -1,
		"customName": "Outbound Octets",
		"description": "The total number of octets transmitted out of the interface, including framing characters.",
		"graphed": true,
		"metricType": "Delta",
		"monitored": true,
		"name": "ifOutOctets",
		"serviceType": "interfaces",
		"sourceType": "interfaces",
		"warningThreshold": -1
	},
	"ifSpeed": {
		"computeType": "Query",
		"criticalThreshold": -1,
		"customName": "Interface Speed",
		"description": "An estimate of the interface's current bandwidth in bits per second.",
		"graphed": true,
		"metricType": "Gauge",
		"monitored": true,
		"name": "ifSpeed",
		"serviceType": "interfaces",
		"sourceType": "interfaces",
		"warningThreshold": -1
	}
}`)

	devicesJSON := []byte(`
{
	"c2801": {
		"Community": "public",
		"IP": "172.21.0.1",
		"Interfaces": {
			"1": {
				"Device": "c2801",
				"Index": 1,
				"Metrics": {
					"ifHCInOctets": {
						"Key": "bytesInX64",
						"Mib": "ifHCInOctets",
						"Value": 161142097880
					},
					"ifHCOutOctets": {
						"Key": "bytesOutX64",
						"Mib": "ifHCOutOctets",
						"Value": 151443135683
					},
					"ifInDiscards": {
						"Key": "ifInDiscards",
						"Mib": "ifInDiscards",
						"Value": 1629
					},
					"ifInErrors": {
						"Key": "ifInErrors",
						"Mib": "ifInErrors",
						"Value": 226
					},
					"ifInOctets": {
						"Key": "bytesIn",
						"Mib": "ifInOctets",
						"Value": 2228306860
					},
					"ifOutDiscards": {
						"Key": "ifOutDiscards",
						"Mib": "ifOutDiscards",
						"Value": 0
					},
					"ifOutErrors": {
						"Key": "ifOutErrors",
						"Mib": "ifOutErrors",
						"Value": 0
					},
					"ifOutOctets": {
						"Key": "bytesOut",
						"Mib": "ifOutOctets",
						"Value": 1119276519
					},
					"ifSpeed": {
						"Key": "ifSpeed",
						"Mib": "ifSpeed",
						"Value": 100000000
					}
				},
				"Name": "Fa0/0",
				"Status": 3
			},
			"2": {
				"Device": "c2801",
				"Index": 2,
				"Metrics": {
					"ifHCInOctets": {
						"Key": "bytesInX64",
						"Mib": "ifHCInOctets",
						"Value": 61917601069
					},
					"ifHCOutOctets": {
						"Key": "bytesOutX64",
						"Mib": "ifHCOutOctets",
						"Value": 100727023225
					},
					"ifInDiscards": {
						"Key": "ifInDiscards",
						"Mib": "ifInDiscards",
						"Value": 142
					},
					"ifInErrors": {
						"Key": "ifInErrors",
						"Mib": "ifInErrors",
						"Value": 8
					},
					"ifInOctets": {
						"Key": "bytesIn",
						"Mib": "ifInOctets",
						"Value": 1788058925
					},
					"ifOutDiscards": {
						"Key": "ifOutDiscards",
						"Mib": "ifOutDiscards",
						"Value": 0
					},
					"ifOutErrors": {
						"Key": "ifOutErrors",
						"Mib": "ifOutErrors",
						"Value": 0
					},
					"ifOutOctets": {
						"Key": "bytesOut",
						"Mib": "ifOutOctets",
						"Value": 1942775357
					},
					"ifSpeed": {
						"Key": "ifSpeed",
						"Mib": "ifSpeed",
						"Value": 100000000
					}
				},
				"Name": "Fa0/1",
				"Status": 3
			},
			"4": {
				"Device": "c2801",
				"Index": 4,
				"Metrics": {
					"ifInDiscards": {
						"Key": "ifInDiscards",
						"Mib": "ifInDiscards",
						"Value": 0
					},
					"ifInErrors": {
						"Key": "ifInErrors",
						"Mib": "ifInErrors",
						"Value": 0
					},
					"ifInOctets": {
						"Key": "bytesIn",
						"Mib": "ifInOctets",
						"Value": 2708168336
					},
					"ifOutDiscards": {
						"Key": "ifOutDiscards",
						"Mib": "ifOutDiscards",
						"Value": 0
					},
					"ifOutErrors": {
						"Key": "ifOutErrors",
						"Mib": "ifOutErrors",
						"Value": 0
					},
					"ifOutOctets": {
						"Key": "bytesOut",
						"Mib": "ifOutOctets",
						"Value": 0
					},
					"ifSpeed": {
						"Key": "ifSpeed",
						"Mib": "ifSpeed",
						"Value": 4294967295
					}
				},
				"Name": "Nu0",
				"Status": 3
			},
			"5": {
				"Device": "c2801",
				"Index": 5,
				"Metrics": {
					"ifHCInOctets": {
						"Key": "bytesInX64",
						"Mib": "ifHCInOctets",
						"Value": 0
					},
					"ifHCOutOctets": {
						"Key": "bytesOutX64",
						"Mib": "ifHCOutOctets",
						"Value": 0
					},
					"ifInDiscards": {
						"Key": "ifInDiscards",
						"Mib": "ifInDiscards",
						"Value": 0
					},
					"ifInErrors": {
						"Key": "ifInErrors",
						"Mib": "ifInErrors",
						"Value": 0
					},
					"ifInOctets": {
						"Key": "bytesIn",
						"Mib": "ifInOctets",
						"Value": 0
					},
					"ifOutDiscards": {
						"Key": "ifOutDiscards",
						"Mib": "ifOutDiscards",
						"Value": 0
					},
					"ifOutErrors": {
						"Key": "ifOutErrors",
						"Mib": "ifOutErrors",
						"Value": 0
					},
					"ifOutOctets": {
						"Key": "bytesOut",
						"Mib": "ifOutOctets",
						"Value": 0
					},
					"ifSpeed": {
						"Key": "ifSpeed",
						"Mib": "ifSpeed",
						"Value": 4294967295
					}
				},
				"Name": "Lo6774",
				"Status": 3
			},
			"6": {
				"Device": "c2801",
				"Index": 6,
				"Metrics": {
					"ifHCInOctets": {
						"Key": "bytesInX64",
						"Mib": "ifHCInOctets",
						"Value": 119534809582
					},
					"ifHCOutOctets": {
						"Key": "bytesOutX64",
						"Mib": "ifHCOutOctets",
						"Value": 73503320633
					},
					"ifInDiscards": {
						"Key": "ifInDiscards",
						"Mib": "ifInDiscards",
						"Value": 10
					},
					"ifInErrors": {
						"Key": "ifInErrors",
						"Mib": "ifInErrors",
						"Value": 0
					},
					"ifInOctets": {
						"Key": "bytesIn",
						"Mib": "ifInOctets",
						"Value": 3570691956
					},
					"ifOutDiscards": {
						"Key": "ifOutDiscards",
						"Mib": "ifOutDiscards",
						"Value": 15419
					},
					"ifOutErrors": {
						"Key": "ifOutErrors",
						"Mib": "ifOutErrors",
						"Value": 0
					},
					"ifOutOctets": {
						"Key": "bytesOut",
						"Mib": "ifOutOctets",
						"Value": 488871949
					},
					"ifSpeed": {
						"Key": "ifSpeed",
						"Mib": "ifSpeed",
						"Value": 9000
					}
				},
				"Name": "Tu0",
				"Status": 3
			},
			"7": {
				"Device": "c2801",
				"Index": 7,
				"Metrics": {
					"ifHCInOctets": {
						"Key": "bytesInX64",
						"Mib": "ifHCInOctets",
						"Value": 159652206772
					},
					"ifHCOutOctets": {
						"Key": "bytesOutX64",
						"Mib": "ifHCOutOctets",
						"Value": 150976952657
					},
					"ifInOctets": {
						"Key": "bytesIn",
						"Mib": "ifInOctets",
						"Value": 738410420
					},
					"ifOutOctets": {
						"Key": "bytesOut",
						"Mib": "ifOutOctets",
						"Value": 653071117
					},
					"ifSpeed": {
						"Key": "ifSpeed",
						"Mib": "ifSpeed",
						"Value": 100000000
					}
				},
				"Name": "Fa0/0.1003",
				"Status": 3
			},
			"8": {
				"Device": "c2801",
				"Index": 8,
				"Metrics": {
					"ifHCInOctets": {
						"Key": "bytesInX64",
						"Mib": "ifHCInOctets",
						"Value": 61917593807
					},
					"ifHCOutOctets": {
						"Key": "bytesOutX64",
						"Mib": "ifHCOutOctets",
						"Value": 100229698830
					},
					"ifInOctets": {
						"Key": "bytesIn",
						"Mib": "ifInOctets",
						"Value": 1788051663
					},
					"ifOutOctets": {
						"Key": "bytesOut",
						"Mib": "ifOutOctets",
						"Value": 1445451022
					},
					"ifSpeed": {
						"Key": "ifSpeed",
						"Mib": "ifSpeed",
						"Value": 100000000
					}
				},
				"Name": "Fa0/1.1002",
				"Status": 3
			}
		},
		"LastOK": 1696286883,
		"Name": "c2801",
		"SecData": {
			"AuthPassword": "***",
			"AuthProtocol": "",
			"Name": "public",
			"PrivacyPassword": "***",
			"PrivacyProtocol": ""
		}
	}
}`)

	mResourcesJSON := []byte(`
[
	{
		"name": "c2801",
		"type": "host",
		"status": "HOST_UP",
		"lastCheckTime": "1732834159762",
		"nextCheckTime": "1732834159762",
		"services": [
			{
				"name": "Fa0/0",
				"type": "service",
				"owner": "c2801",
				"status": "SERVICE_OK",
				"lastCheckTime": "1732834159762",
				"nextCheckTime": "1732834159762",
				"lastPluginOutput": "Interface Operational State is UP, Administrative state is UP",
				"metrics": [
					{
						"metricName": "Inbound Discards",
						"sampleType": "Value",
						"interval": {
							"endTime": "1732834159762",
							"startTime": "1732834159762"
						},
						"value": {
							"valueType": "IntegerType",
							"integerValue": 10
						},
						"unit": "1",
						"thresholds": [
							{
								"sampleType": "Warning",
								"label": "Inbound Discards_wn",
								"value": {
									"valueType": "IntegerType",
									"integerValue": -1
								}
							},
							{
								"sampleType": "Critical",
								"label": "Inbound Discards_cr",
								"value": {
									"valueType": "IntegerType",
									"integerValue": -1
								}
							}
						]
					},
					{
						"metricName": "Inbound Errors",
						"sampleType": "Value",
						"interval": {
							"endTime": "1732834159762",
							"startTime": "1732834159762"
						},
						"value": {
							"valueType": "IntegerType",
							"integerValue": 10
						},
						"unit": "1",
						"thresholds": [
							{
								"sampleType": "Warning",
								"label": "Inbound Errors_wn",
								"value": {
									"valueType": "IntegerType",
									"integerValue": -1
								}
							},
							{
								"sampleType": "Critical",
								"label": "Inbound Errors_cr",
								"value": {
									"valueType": "IntegerType",
									"integerValue": -1
								}
							}
						]
					},
					{
						"metricName": "Inbound Octets",
						"sampleType": "Value",
						"interval": {
							"endTime": "1732834159762",
							"startTime": "1732834159762"
						},
						"value": {
							"valueType": "IntegerType",
							"integerValue": 10
						},
						"unit": "1",
						"thresholds": [
							{
								"sampleType": "Warning",
								"label": "Inbound Octets_wn",
								"value": {
									"valueType": "IntegerType",
									"integerValue": -1
								}
							},
							{
								"sampleType": "Critical",
								"label": "Inbound Octets_cr",
								"value": {
									"valueType": "IntegerType",
									"integerValue": -1
								}
							}
						]
					},
					{
						"metricName": "Interface Speed",
						"sampleType": "Value",
						"interval": {
							"endTime": "1732834159762",
							"startTime": "1732834159762"
						},
						"value": {
							"valueType": "IntegerType",
							"integerValue": 100000000
						},
						"unit": "1",
						"thresholds": [
							{
								"sampleType": "Warning",
								"label": "Interface Speed_wn",
								"value": {
									"valueType": "IntegerType",
									"integerValue": -1
								}
							},
							{
								"sampleType": "Critical",
								"label": "Interface Speed_cr",
								"value": {
									"valueType": "IntegerType",
									"integerValue": -1
								}
							}
						]
					},
					{
						"metricName": "Outbound Discards",
						"sampleType": "Value",
						"interval": {
							"endTime": "1732834159762",
							"startTime": "1732834159762"
						},
						"value": {
							"valueType": "IntegerType",
							"integerValue": 10
						},
						"unit": "1",
						"thresholds": [
							{
								"sampleType": "Warning",
								"label": "Outbound Discards_wn",
								"value": {
									"valueType": "IntegerType",
									"integerValue": -1
								}
							},
							{
								"sampleType": "Critical",
								"label": "Outbound Discards_cr",
								"value": {
									"valueType": "IntegerType",
									"integerValue": -1
								}
							}
						]
					},
					{
						"metricName": "Outbound Errors",
						"sampleType": "Value",
						"interval": {
							"endTime": "1732834159762",
							"startTime": "1732834159762"
						},
						"value": {
							"valueType": "IntegerType",
							"integerValue": 10
						},
						"unit": "1",
						"thresholds": [
							{
								"sampleType": "Warning",
								"label": "Outbound Errors_wn",
								"value": {
									"valueType": "IntegerType",
									"integerValue": -1
								}
							},
							{
								"sampleType": "Critical",
								"label": "Outbound Errors_cr",
								"value": {
									"valueType": "IntegerType",
									"integerValue": -1
								}
							}
						]
					},
					{
						"metricName": "Outbound Octets",
						"sampleType": "Value",
						"interval": {
							"endTime": "1732834159762",
							"startTime": "1732834159762"
						},
						"value": {
							"valueType": "IntegerType",
							"integerValue": 10
						},
						"unit": "1",
						"thresholds": [
							{
								"sampleType": "Warning",
								"label": "Outbound Octets_wn",
								"value": {
									"valueType": "IntegerType",
									"integerValue": -1
								}
							},
							{
								"sampleType": "Critical",
								"label": "Outbound Octets_cr",
								"value": {
									"valueType": "IntegerType",
									"integerValue": -1
								}
							}
						]
					},
					{
						"metricName": "ifHCInOctets",
						"sampleType": "Value",
						"interval": {
							"endTime": "1732834159762",
							"startTime": "1732834159762"
						},
						"value": {
							"valueType": "IntegerType",
							"integerValue": 10
						},
						"unit": "1",
						"thresholds": [
							{
								"sampleType": "Warning",
								"label": "ifHCInOctets_wn",
								"value": {
									"valueType": "IntegerType",
									"integerValue": -1
								}
							},
							{
								"sampleType": "Critical",
								"label": "ifHCInOctets_cr",
								"value": {
									"valueType": "IntegerType",
									"integerValue": -1
								}
							}
						]
					},
					{
						"metricName": "ifHCOutOctets",
						"sampleType": "Value",
						"interval": {
							"endTime": "1732834159762",
							"startTime": "1732834159762"
						},
						"value": {
							"valueType": "IntegerType",
							"integerValue": 10
						},
						"unit": "1",
						"thresholds": [
							{
								"sampleType": "Warning",
								"label": "ifHCOutOctets_wn",
								"value": {
									"valueType": "IntegerType",
									"integerValue": -1
								}
							},
							{
								"sampleType": "Critical",
								"label": "ifHCOutOctets_cr",
								"value": {
									"valueType": "IntegerType",
									"integerValue": -1
								}
							}
						]
					}
				]
			},
			{
				"name": "Fa0/0.1003",
				"type": "service",
				"owner": "c2801",
				"status": "SERVICE_OK",
				"lastCheckTime": "1732834159762",
				"nextCheckTime": "1732834159762",
				"lastPluginOutput": "Interface Operational State is UP, Administrative state is UP",
				"metrics": [
					{
						"metricName": "Inbound Octets",
						"sampleType": "Value",
						"interval": {
							"endTime": "1732834159762",
							"startTime": "1732834159762"
						},
						"value": {
							"valueType": "IntegerType",
							"integerValue": 10
						},
						"unit": "1",
						"thresholds": [
							{
								"sampleType": "Warning",
								"label": "Inbound Octets_wn",
								"value": {
									"valueType": "IntegerType",
									"integerValue": -1
								}
							},
							{
								"sampleType": "Critical",
								"label": "Inbound Octets_cr",
								"value": {
									"valueType": "IntegerType",
									"integerValue": -1
								}
							}
						]
					},
					{
						"metricName": "Interface Speed",
						"sampleType": "Value",
						"interval": {
							"endTime": "1732834159762",
							"startTime": "1732834159762"
						},
						"value": {
							"valueType": "IntegerType",
							"integerValue": 100000000
						},
						"unit": "1",
						"thresholds": [
							{
								"sampleType": "Warning",
								"label": "Interface Speed_wn",
								"value": {
									"valueType": "IntegerType",
									"integerValue": -1
								}
							},
							{
								"sampleType": "Critical",
								"label": "Interface Speed_cr",
								"value": {
									"valueType": "IntegerType",
									"integerValue": -1
								}
							}
						]
					},
					{
						"metricName": "Outbound Octets",
						"sampleType": "Value",
						"interval": {
							"endTime": "1732834159762",
							"startTime": "1732834159762"
						},
						"value": {
							"valueType": "IntegerType",
							"integerValue": 10
						},
						"unit": "1",
						"thresholds": [
							{
								"sampleType": "Warning",
								"label": "Outbound Octets_wn",
								"value": {
									"valueType": "IntegerType",
									"integerValue": -1
								}
							},
							{
								"sampleType": "Critical",
								"label": "Outbound Octets_cr",
								"value": {
									"valueType": "IntegerType",
									"integerValue": -1
								}
							}
						]
					},
					{
						"metricName": "ifHCInOctets",
						"sampleType": "Value",
						"interval": {
							"endTime": "1732834159762",
							"startTime": "1732834159762"
						},
						"value": {
							"valueType": "IntegerType",
							"integerValue": 10
						},
						"unit": "1",
						"thresholds": [
							{
								"sampleType": "Warning",
								"label": "ifHCInOctets_wn",
								"value": {
									"valueType": "IntegerType",
									"integerValue": -1
								}
							},
							{
								"sampleType": "Critical",
								"label": "ifHCInOctets_cr",
								"value": {
									"valueType": "IntegerType",
									"integerValue": -1
								}
							}
						]
					},
					{
						"metricName": "ifHCOutOctets",
						"sampleType": "Value",
						"interval": {
							"endTime": "1732834159762",
							"startTime": "1732834159762"
						},
						"value": {
							"valueType": "IntegerType",
							"integerValue": 10
						},
						"unit": "1",
						"thresholds": [
							{
								"sampleType": "Warning",
								"label": "ifHCOutOctets_wn",
								"value": {
									"valueType": "IntegerType",
									"integerValue": -1
								}
							},
							{
								"sampleType": "Critical",
								"label": "ifHCOutOctets_cr",
								"value": {
									"valueType": "IntegerType",
									"integerValue": -1
								}
							}
						]
					}
				]
			},
			{
				"name": "Fa0/1",
				"type": "service",
				"owner": "c2801",
				"status": "SERVICE_OK",
				"lastCheckTime": "1732834159762",
				"nextCheckTime": "1732834159762",
				"lastPluginOutput": "Interface Operational State is UP, Administrative state is UP",
				"metrics": [
					{
						"metricName": "Inbound Discards",
						"sampleType": "Value",
						"interval": {
							"endTime": "1732834159762",
							"startTime": "1732834159762"
						},
						"value": {
							"valueType": "IntegerType",
							"integerValue": 10
						},
						"unit": "1",
						"thresholds": [
							{
								"sampleType": "Warning",
								"label": "Inbound Discards_wn",
								"value": {
									"valueType": "IntegerType",
									"integerValue": -1
								}
							},
							{
								"sampleType": "Critical",
								"label": "Inbound Discards_cr",
								"value": {
									"valueType": "IntegerType",
									"integerValue": -1
								}
							}
						]
					},
					{
						"metricName": "Inbound Errors",
						"sampleType": "Value",
						"interval": {
							"endTime": "1732834159762",
							"startTime": "1732834159762"
						},
						"value": {
							"valueType": "IntegerType",
							"integerValue": 10
						},
						"unit": "1",
						"thresholds": [
							{
								"sampleType": "Warning",
								"label": "Inbound Errors_wn",
								"value": {
									"valueType": "IntegerType",
									"integerValue": -1
								}
							},
							{
								"sampleType": "Critical",
								"label": "Inbound Errors_cr",
								"value": {
									"valueType": "IntegerType",
									"integerValue": -1
								}
							}
						]
					},
					{
						"metricName": "Inbound Octets",
						"sampleType": "Value",
						"interval": {
							"endTime": "1732834159762",
							"startTime": "1732834159762"
						},
						"value": {
							"valueType": "IntegerType",
							"integerValue": 10
						},
						"unit": "1",
						"thresholds": [
							{
								"sampleType": "Warning",
								"label": "Inbound Octets_wn",
								"value": {
									"valueType": "IntegerType",
									"integerValue": -1
								}
							},
							{
								"sampleType": "Critical",
								"label": "Inbound Octets_cr",
								"value": {
									"valueType": "IntegerType",
									"integerValue": -1
								}
							}
						]
					},
					{
						"metricName": "Interface Speed",
						"sampleType": "Value",
						"interval": {
							"endTime": "1732834159762",
							"startTime": "1732834159762"
						},
						"value": {
							"valueType": "IntegerType",
							"integerValue": 100000000
						},
						"unit": "1",
						"thresholds": [
							{
								"sampleType": "Warning",
								"label": "Interface Speed_wn",
								"value": {
									"valueType": "IntegerType",
									"integerValue": -1
								}
							},
							{
								"sampleType": "Critical",
								"label": "Interface Speed_cr",
								"value": {
									"valueType": "IntegerType",
									"integerValue": -1
								}
							}
						]
					},
					{
						"metricName": "Outbound Discards",
						"sampleType": "Value",
						"interval": {
							"endTime": "1732834159762",
							"startTime": "1732834159762"
						},
						"value": {
							"valueType": "IntegerType",
							"integerValue": 10
						},
						"unit": "1",
						"thresholds": [
							{
								"sampleType": "Warning",
								"label": "Outbound Discards_wn",
								"value": {
									"valueType": "IntegerType",
									"integerValue": -1
								}
							},
							{
								"sampleType": "Critical",
								"label": "Outbound Discards_cr",
								"value": {
									"valueType": "IntegerType",
									"integerValue": -1
								}
							}
						]
					},
					{
						"metricName": "Outbound Errors",
						"sampleType": "Value",
						"interval": {
							"endTime": "1732834159762",
							"startTime": "1732834159762"
						},
						"value": {
							"valueType": "IntegerType",
							"integerValue": 10
						},
						"unit": "1",
						"thresholds": [
							{
								"sampleType": "Warning",
								"label": "Outbound Errors_wn",
								"value": {
									"valueType": "IntegerType",
									"integerValue": -1
								}
							},
							{
								"sampleType": "Critical",
								"label": "Outbound Errors_cr",
								"value": {
									"valueType": "IntegerType",
									"integerValue": -1
								}
							}
						]
					},
					{
						"metricName": "Outbound Octets",
						"sampleType": "Value",
						"interval": {
							"endTime": "1732834159762",
							"startTime": "1732834159762"
						},
						"value": {
							"valueType": "IntegerType",
							"integerValue": 10
						},
						"unit": "1",
						"thresholds": [
							{
								"sampleType": "Warning",
								"label": "Outbound Octets_wn",
								"value": {
									"valueType": "IntegerType",
									"integerValue": -1
								}
							},
							{
								"sampleType": "Critical",
								"label": "Outbound Octets_cr",
								"value": {
									"valueType": "IntegerType",
									"integerValue": -1
								}
							}
						]
					},
					{
						"metricName": "ifHCInOctets",
						"sampleType": "Value",
						"interval": {
							"endTime": "1732834159762",
							"startTime": "1732834159762"
						},
						"value": {
							"valueType": "IntegerType",
							"integerValue": 10
						},
						"unit": "1",
						"thresholds": [
							{
								"sampleType": "Warning",
								"label": "ifHCInOctets_wn",
								"value": {
									"valueType": "IntegerType",
									"integerValue": -1
								}
							},
							{
								"sampleType": "Critical",
								"label": "ifHCInOctets_cr",
								"value": {
									"valueType": "IntegerType",
									"integerValue": -1
								}
							}
						]
					},
					{
						"metricName": "ifHCOutOctets",
						"sampleType": "Value",
						"interval": {
							"endTime": "1732834159762",
							"startTime": "1732834159762"
						},
						"value": {
							"valueType": "IntegerType",
							"integerValue": 10
						},
						"unit": "1",
						"thresholds": [
							{
								"sampleType": "Warning",
								"label": "ifHCOutOctets_wn",
								"value": {
									"valueType": "IntegerType",
									"integerValue": -1
								}
							},
							{
								"sampleType": "Critical",
								"label": "ifHCOutOctets_cr",
								"value": {
									"valueType": "IntegerType",
									"integerValue": -1
								}
							}
						]
					}
				]
			},
			{
				"name": "Fa0/1.1002",
				"type": "service",
				"owner": "c2801",
				"status": "SERVICE_OK",
				"lastCheckTime": "1732834159762",
				"nextCheckTime": "1732834159762",
				"lastPluginOutput": "Interface Operational State is UP, Administrative state is UP",
				"metrics": [
					{
						"metricName": "Inbound Octets",
						"sampleType": "Value",
						"interval": {
							"endTime": "1732834159762",
							"startTime": "1732834159762"
						},
						"value": {
							"valueType": "IntegerType",
							"integerValue": 10
						},
						"unit": "1",
						"thresholds": [
							{
								"sampleType": "Warning",
								"label": "Inbound Octets_wn",
								"value": {
									"valueType": "IntegerType",
									"integerValue": -1
								}
							},
							{
								"sampleType": "Critical",
								"label": "Inbound Octets_cr",
								"value": {
									"valueType": "IntegerType",
									"integerValue": -1
								}
							}
						]
					},
					{
						"metricName": "Interface Speed",
						"sampleType": "Value",
						"interval": {
							"endTime": "1732834159762",
							"startTime": "1732834159762"
						},
						"value": {
							"valueType": "IntegerType",
							"integerValue": 100000000
						},
						"unit": "1",
						"thresholds": [
							{
								"sampleType": "Warning",
								"label": "Interface Speed_wn",
								"value": {
									"valueType": "IntegerType",
									"integerValue": -1
								}
							},
							{
								"sampleType": "Critical",
								"label": "Interface Speed_cr",
								"value": {
									"valueType": "IntegerType",
									"integerValue": -1
								}
							}
						]
					},
					{
						"metricName": "Outbound Octets",
						"sampleType": "Value",
						"interval": {
							"endTime": "1732834159762",
							"startTime": "1732834159762"
						},
						"value": {
							"valueType": "IntegerType",
							"integerValue": 10
						},
						"unit": "1",
						"thresholds": [
							{
								"sampleType": "Warning",
								"label": "Outbound Octets_wn",
								"value": {
									"valueType": "IntegerType",
									"integerValue": -1
								}
							},
							{
								"sampleType": "Critical",
								"label": "Outbound Octets_cr",
								"value": {
									"valueType": "IntegerType",
									"integerValue": -1
								}
							}
						]
					},
					{
						"metricName": "ifHCInOctets",
						"sampleType": "Value",
						"interval": {
							"endTime": "1732834159762",
							"startTime": "1732834159762"
						},
						"value": {
							"valueType": "IntegerType",
							"integerValue": 10
						},
						"unit": "1",
						"thresholds": [
							{
								"sampleType": "Warning",
								"label": "ifHCInOctets_wn",
								"value": {
									"valueType": "IntegerType",
									"integerValue": -1
								}
							},
							{
								"sampleType": "Critical",
								"label": "ifHCInOctets_cr",
								"value": {
									"valueType": "IntegerType",
									"integerValue": -1
								}
							}
						]
					},
					{
						"metricName": "ifHCOutOctets",
						"sampleType": "Value",
						"interval": {
							"endTime": "1732834159762",
							"startTime": "1732834159762"
						},
						"value": {
							"valueType": "IntegerType",
							"integerValue": 10
						},
						"unit": "1",
						"thresholds": [
							{
								"sampleType": "Warning",
								"label": "ifHCOutOctets_wn",
								"value": {
									"valueType": "IntegerType",
									"integerValue": -1
								}
							},
							{
								"sampleType": "Critical",
								"label": "ifHCOutOctets_cr",
								"value": {
									"valueType": "IntegerType",
									"integerValue": -1
								}
							}
						]
					}
				]
			},
			{
				"name": "Lo6774",
				"type": "service",
				"owner": "c2801",
				"status": "SERVICE_OK",
				"lastCheckTime": "1732834159762",
				"nextCheckTime": "1732834159762",
				"lastPluginOutput": "Interface Operational State is UP, Administrative state is UP",
				"metrics": [
					{
						"metricName": "Inbound Discards",
						"sampleType": "Value",
						"interval": {
							"endTime": "1732834159762",
							"startTime": "1732834159762"
						},
						"value": {
							"valueType": "IntegerType",
							"integerValue": 10
						},
						"unit": "1",
						"thresholds": [
							{
								"sampleType": "Warning",
								"label": "Inbound Discards_wn",
								"value": {
									"valueType": "IntegerType",
									"integerValue": -1
								}
							},
							{
								"sampleType": "Critical",
								"label": "Inbound Discards_cr",
								"value": {
									"valueType": "IntegerType",
									"integerValue": -1
								}
							}
						]
					},
					{
						"metricName": "Inbound Errors",
						"sampleType": "Value",
						"interval": {
							"endTime": "1732834159762",
							"startTime": "1732834159762"
						},
						"value": {
							"valueType": "IntegerType",
							"integerValue": 10
						},
						"unit": "1",
						"thresholds": [
							{
								"sampleType": "Warning",
								"label": "Inbound Errors_wn",
								"value": {
									"valueType": "IntegerType",
									"integerValue": -1
								}
							},
							{
								"sampleType": "Critical",
								"label": "Inbound Errors_cr",
								"value": {
									"valueType": "IntegerType",
									"integerValue": -1
								}
							}
						]
					},
					{
						"metricName": "Inbound Octets",
						"sampleType": "Value",
						"interval": {
							"endTime": "1732834159762",
							"startTime": "1732834159762"
						},
						"value": {
							"valueType": "IntegerType",
							"integerValue": 10
						},
						"unit": "1",
						"thresholds": [
							{
								"sampleType": "Warning",
								"label": "Inbound Octets_wn",
								"value": {
									"valueType": "IntegerType",
									"integerValue": -1
								}
							},
							{
								"sampleType": "Critical",
								"label": "Inbound Octets_cr",
								"value": {
									"valueType": "IntegerType",
									"integerValue": -1
								}
							}
						]
					},
					{
						"metricName": "Interface Speed",
						"sampleType": "Value",
						"interval": {
							"endTime": "1732834159762",
							"startTime": "1732834159762"
						},
						"value": {
							"valueType": "IntegerType",
							"integerValue": 4294967295
						},
						"unit": "1",
						"thresholds": [
							{
								"sampleType": "Warning",
								"label": "Interface Speed_wn",
								"value": {
									"valueType": "IntegerType",
									"integerValue": -1
								}
							},
							{
								"sampleType": "Critical",
								"label": "Interface Speed_cr",
								"value": {
									"valueType": "IntegerType",
									"integerValue": -1
								}
							}
						]
					},
					{
						"metricName": "Outbound Discards",
						"sampleType": "Value",
						"interval": {
							"endTime": "1732834159762",
							"startTime": "1732834159762"
						},
						"value": {
							"valueType": "IntegerType",
							"integerValue": 10
						},
						"unit": "1",
						"thresholds": [
							{
								"sampleType": "Warning",
								"label": "Outbound Discards_wn",
								"value": {
									"valueType": "IntegerType",
									"integerValue": -1
								}
							},
							{
								"sampleType": "Critical",
								"label": "Outbound Discards_cr",
								"value": {
									"valueType": "IntegerType",
									"integerValue": -1
								}
							}
						]
					},
					{
						"metricName": "Outbound Errors",
						"sampleType": "Value",
						"interval": {
							"endTime": "1732834159762",
							"startTime": "1732834159762"
						},
						"value": {
							"valueType": "IntegerType",
							"integerValue": 10
						},
						"unit": "1",
						"thresholds": [
							{
								"sampleType": "Warning",
								"label": "Outbound Errors_wn",
								"value": {
									"valueType": "IntegerType",
									"integerValue": -1
								}
							},
							{
								"sampleType": "Critical",
								"label": "Outbound Errors_cr",
								"value": {
									"valueType": "IntegerType",
									"integerValue": -1
								}
							}
						]
					},
					{
						"metricName": "Outbound Octets",
						"sampleType": "Value",
						"interval": {
							"endTime": "1732834159762",
							"startTime": "1732834159762"
						},
						"value": {
							"valueType": "IntegerType",
							"integerValue": 10
						},
						"unit": "1",
						"thresholds": [
							{
								"sampleType": "Warning",
								"label": "Outbound Octets_wn",
								"value": {
									"valueType": "IntegerType",
									"integerValue": -1
								}
							},
							{
								"sampleType": "Critical",
								"label": "Outbound Octets_cr",
								"value": {
									"valueType": "IntegerType",
									"integerValue": -1
								}
							}
						]
					},
					{
						"metricName": "ifHCInOctets",
						"sampleType": "Value",
						"interval": {
							"endTime": "1732834159762",
							"startTime": "1732834159762"
						},
						"value": {
							"valueType": "IntegerType",
							"integerValue": 10
						},
						"unit": "1",
						"thresholds": [
							{
								"sampleType": "Warning",
								"label": "ifHCInOctets_wn",
								"value": {
									"valueType": "IntegerType",
									"integerValue": -1
								}
							},
							{
								"sampleType": "Critical",
								"label": "ifHCInOctets_cr",
								"value": {
									"valueType": "IntegerType",
									"integerValue": -1
								}
							}
						]
					},
					{
						"metricName": "ifHCOutOctets",
						"sampleType": "Value",
						"interval": {
							"endTime": "1732834159762",
							"startTime": "1732834159762"
						},
						"value": {
							"valueType": "IntegerType",
							"integerValue": 10
						},
						"unit": "1",
						"thresholds": [
							{
								"sampleType": "Warning",
								"label": "ifHCOutOctets_wn",
								"value": {
									"valueType": "IntegerType",
									"integerValue": -1
								}
							},
							{
								"sampleType": "Critical",
								"label": "ifHCOutOctets_cr",
								"value": {
									"valueType": "IntegerType",
									"integerValue": -1
								}
							}
						]
					}
				]
			},
			{
				"name": "Nu0",
				"type": "service",
				"owner": "c2801",
				"status": "SERVICE_OK",
				"lastCheckTime": "1732834159762",
				"nextCheckTime": "1732834159762",
				"lastPluginOutput": "Interface Operational State is UP, Administrative state is UP",
				"metrics": [
					{
						"metricName": "Inbound Discards",
						"sampleType": "Value",
						"interval": {
							"endTime": "1732834159762",
							"startTime": "1732834159762"
						},
						"value": {
							"valueType": "IntegerType",
							"integerValue": 10
						},
						"unit": "1",
						"thresholds": [
							{
								"sampleType": "Warning",
								"label": "Inbound Discards_wn",
								"value": {
									"valueType": "IntegerType",
									"integerValue": -1
								}
							},
							{
								"sampleType": "Critical",
								"label": "Inbound Discards_cr",
								"value": {
									"valueType": "IntegerType",
									"integerValue": -1
								}
							}
						]
					},
					{
						"metricName": "Inbound Errors",
						"sampleType": "Value",
						"interval": {
							"endTime": "1732834159762",
							"startTime": "1732834159762"
						},
						"value": {
							"valueType": "IntegerType",
							"integerValue": 10
						},
						"unit": "1",
						"thresholds": [
							{
								"sampleType": "Warning",
								"label": "Inbound Errors_wn",
								"value": {
									"valueType": "IntegerType",
									"integerValue": -1
								}
							},
							{
								"sampleType": "Critical",
								"label": "Inbound Errors_cr",
								"value": {
									"valueType": "IntegerType",
									"integerValue": -1
								}
							}
						]
					},
					{
						"metricName": "Inbound Octets",
						"sampleType": "Value",
						"interval": {
							"endTime": "1732834159762",
							"startTime": "1732834159762"
						},
						"value": {
							"valueType": "IntegerType",
							"integerValue": 10
						},
						"unit": "1",
						"thresholds": [
							{
								"sampleType": "Warning",
								"label": "Inbound Octets_wn",
								"value": {
									"valueType": "IntegerType",
									"integerValue": -1
								}
							},
							{
								"sampleType": "Critical",
								"label": "Inbound Octets_cr",
								"value": {
									"valueType": "IntegerType",
									"integerValue": -1
								}
							}
						]
					},
					{
						"metricName": "Interface Speed",
						"sampleType": "Value",
						"interval": {
							"endTime": "1732834159762",
							"startTime": "1732834159762"
						},
						"value": {
							"valueType": "IntegerType",
							"integerValue": 4294967295
						},
						"unit": "1",
						"thresholds": [
							{
								"sampleType": "Warning",
								"label": "Interface Speed_wn",
								"value": {
									"valueType": "IntegerType",
									"integerValue": -1
								}
							},
							{
								"sampleType": "Critical",
								"label": "Interface Speed_cr",
								"value": {
									"valueType": "IntegerType",
									"integerValue": -1
								}
							}
						]
					},
					{
						"metricName": "Outbound Discards",
						"sampleType": "Value",
						"interval": {
							"endTime": "1732834159762",
							"startTime": "1732834159762"
						},
						"value": {
							"valueType": "IntegerType",
							"integerValue": 10
						},
						"unit": "1",
						"thresholds": [
							{
								"sampleType": "Warning",
								"label": "Outbound Discards_wn",
								"value": {
									"valueType": "IntegerType",
									"integerValue": -1
								}
							},
							{
								"sampleType": "Critical",
								"label": "Outbound Discards_cr",
								"value": {
									"valueType": "IntegerType",
									"integerValue": -1
								}
							}
						]
					},
					{
						"metricName": "Outbound Errors",
						"sampleType": "Value",
						"interval": {
							"endTime": "1732834159762",
							"startTime": "1732834159762"
						},
						"value": {
							"valueType": "IntegerType",
							"integerValue": 10
						},
						"unit": "1",
						"thresholds": [
							{
								"sampleType": "Warning",
								"label": "Outbound Errors_wn",
								"value": {
									"valueType": "IntegerType",
									"integerValue": -1
								}
							},
							{
								"sampleType": "Critical",
								"label": "Outbound Errors_cr",
								"value": {
									"valueType": "IntegerType",
									"integerValue": -1
								}
							}
						]
					},
					{
						"metricName": "Outbound Octets",
						"sampleType": "Value",
						"interval": {
							"endTime": "1732834159762",
							"startTime": "1732834159762"
						},
						"value": {
							"valueType": "IntegerType",
							"integerValue": 10
						},
						"unit": "1",
						"thresholds": [
							{
								"sampleType": "Warning",
								"label": "Outbound Octets_wn",
								"value": {
									"valueType": "IntegerType",
									"integerValue": -1
								}
							},
							{
								"sampleType": "Critical",
								"label": "Outbound Octets_cr",
								"value": {
									"valueType": "IntegerType",
									"integerValue": -1
								}
							}
						]
					}
				]
			},
			{
				"name": "Tu0",
				"type": "service",
				"owner": "c2801",
				"status": "SERVICE_OK",
				"lastCheckTime": "1732834159762",
				"nextCheckTime": "1732834159762",
				"lastPluginOutput": "Interface Operational State is UP, Administrative state is UP",
				"metrics": [
					{
						"metricName": "Inbound Discards",
						"sampleType": "Value",
						"interval": {
							"endTime": "1732834159762",
							"startTime": "1732834159762"
						},
						"value": {
							"valueType": "IntegerType",
							"integerValue": 10
						},
						"unit": "1",
						"thresholds": [
							{
								"sampleType": "Warning",
								"label": "Inbound Discards_wn",
								"value": {
									"valueType": "IntegerType",
									"integerValue": -1
								}
							},
							{
								"sampleType": "Critical",
								"label": "Inbound Discards_cr",
								"value": {
									"valueType": "IntegerType",
									"integerValue": -1
								}
							}
						]
					},
					{
						"metricName": "Inbound Errors",
						"sampleType": "Value",
						"interval": {
							"endTime": "1732834159762",
							"startTime": "1732834159762"
						},
						"value": {
							"valueType": "IntegerType",
							"integerValue": 10
						},
						"unit": "1",
						"thresholds": [
							{
								"sampleType": "Warning",
								"label": "Inbound Errors_wn",
								"value": {
									"valueType": "IntegerType",
									"integerValue": -1
								}
							},
							{
								"sampleType": "Critical",
								"label": "Inbound Errors_cr",
								"value": {
									"valueType": "IntegerType",
									"integerValue": -1
								}
							}
						]
					},
					{
						"metricName": "Inbound Octets",
						"sampleType": "Value",
						"interval": {
							"endTime": "1732834159762",
							"startTime": "1732834159762"
						},
						"value": {
							"valueType": "IntegerType",
							"integerValue": 10
						},
						"unit": "1",
						"thresholds": [
							{
								"sampleType": "Warning",
								"label": "Inbound Octets_wn",
								"value": {
									"valueType": "IntegerType",
									"integerValue": -1
								}
							},
							{
								"sampleType": "Critical",
								"label": "Inbound Octets_cr",
								"value": {
									"valueType": "IntegerType",
									"integerValue": -1
								}
							}
						]
					},
					{
						"metricName": "Interface Speed",
						"sampleType": "Value",
						"interval": {
							"endTime": "1732834159762",
							"startTime": "1732834159762"
						},
						"value": {
							"valueType": "IntegerType",
							"integerValue": 9000
						},
						"unit": "1",
						"thresholds": [
							{
								"sampleType": "Warning",
								"label": "Interface Speed_wn",
								"value": {
									"valueType": "IntegerType",
									"integerValue": -1
								}
							},
							{
								"sampleType": "Critical",
								"label": "Interface Speed_cr",
								"value": {
									"valueType": "IntegerType",
									"integerValue": -1
								}
							}
						]
					},
					{
						"metricName": "Outbound Discards",
						"sampleType": "Value",
						"interval": {
							"endTime": "1732834159762",
							"startTime": "1732834159762"
						},
						"value": {
							"valueType": "IntegerType",
							"integerValue": 10
						},
						"unit": "1",
						"thresholds": [
							{
								"sampleType": "Warning",
								"label": "Outbound Discards_wn",
								"value": {
									"valueType": "IntegerType",
									"integerValue": -1
								}
							},
							{
								"sampleType": "Critical",
								"label": "Outbound Discards_cr",
								"value": {
									"valueType": "IntegerType",
									"integerValue": -1
								}
							}
						]
					},
					{
						"metricName": "Outbound Errors",
						"sampleType": "Value",
						"interval": {
							"endTime": "1732834159762",
							"startTime": "1732834159762"
						},
						"value": {
							"valueType": "IntegerType",
							"integerValue": 10
						},
						"unit": "1",
						"thresholds": [
							{
								"sampleType": "Warning",
								"label": "Outbound Errors_wn",
								"value": {
									"valueType": "IntegerType",
									"integerValue": -1
								}
							},
							{
								"sampleType": "Critical",
								"label": "Outbound Errors_cr",
								"value": {
									"valueType": "IntegerType",
									"integerValue": -1
								}
							}
						]
					},
					{
						"metricName": "Outbound Octets",
						"sampleType": "Value",
						"interval": {
							"endTime": "1732834159762",
							"startTime": "1732834159762"
						},
						"value": {
							"valueType": "IntegerType",
							"integerValue": 10
						},
						"unit": "1",
						"thresholds": [
							{
								"sampleType": "Warning",
								"label": "Outbound Octets_wn",
								"value": {
									"valueType": "IntegerType",
									"integerValue": -1
								}
							},
							{
								"sampleType": "Critical",
								"label": "Outbound Octets_cr",
								"value": {
									"valueType": "IntegerType",
									"integerValue": -1
								}
							}
						]
					},
					{
						"metricName": "ifHCInOctets",
						"sampleType": "Value",
						"interval": {
							"endTime": "1732834159762",
							"startTime": "1732834159762"
						},
						"value": {
							"valueType": "IntegerType",
							"integerValue": 10
						},
						"unit": "1",
						"thresholds": [
							{
								"sampleType": "Warning",
								"label": "ifHCInOctets_wn",
								"value": {
									"valueType": "IntegerType",
									"integerValue": -1
								}
							},
							{
								"sampleType": "Critical",
								"label": "ifHCInOctets_cr",
								"value": {
									"valueType": "IntegerType",
									"integerValue": -1
								}
							}
						]
					},
					{
						"metricName": "ifHCOutOctets",
						"sampleType": "Value",
						"interval": {
							"endTime": "1732834159762",
							"startTime": "1732834159762"
						},
						"value": {
							"valueType": "IntegerType",
							"integerValue": 10
						},
						"unit": "1",
						"thresholds": [
							{
								"sampleType": "Warning",
								"label": "ifHCOutOctets_wn",
								"value": {
									"valueType": "IntegerType",
									"integerValue": -1
								}
							},
							{
								"sampleType": "Critical",
								"label": "ifHCOutOctets_cr",
								"value": {
									"valueType": "IntegerType",
									"integerValue": -1
								}
							}
						]
					}
				]
			}
		]
	}
]`)

	metricDefinitions := make(map[string]transit.MetricDefinition)
	if err := json.Unmarshal(metricDefinitionsJSON, &metricDefinitions); err != nil {
		return nil, nil, nil, err
	}

	state := new(MonitoringState)
	if err := json.Unmarshal(devicesJSON, &state.devices); err != nil {
		return nil, nil, nil, err
	}
	for k, d := range state.devices {
		d.LastOK = time.Now().Unix()
		state.devices[k] = d
	}

	mResources := make([]transit.MonitoredResource, 0)
	if err := json.Unmarshal(mResourcesJSON, &mResources); err != nil {
		return nil, nil, nil, err
	}

	return state, metricDefinitions, mResources, nil
}
