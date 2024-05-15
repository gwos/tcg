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
	// 2nd call for deltas
	monitoredResources := state.retrieveMonitoredResources(metricDefinitions)
	ts := transit.NewTimestamp()
	normalizeData(monitoredResources, ts)
	normalizeData(monitoredResourcesExp, ts)
	// assert.Equal(t, monitoredResourcesExp, monitoredResources)
	monitoredResourcesJSON, _ := json.Marshal(monitoredResources)
	monitoredResourcesExpJSON, _ := json.Marshal(monitoredResourcesExp)
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
	metricDefinitionsJSON := []byte(`{
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

	devicesJSON := []byte(`{
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

	mResourcesJSON := []byte(`[
	{
		"lastCheckTime": "1696286942511",
		"name": "c2801",
		"nextCheckTime": "1696287002511",
		"services": [
			{
				"lastCheckTime": "1696286942511",
				"lastPluginOutput": "Interface Operational State is UP, Administrative state is UP",
				"metrics": [
					{
						"interval": {
							"endTime": "1696286942511",
							"startTime": "1696286942511"
						},
						"metricName": "Outbound Octets",
						"sampleType": "Value",
						"thresholds": [
							{
								"label": "Outbound Octets_wn",
								"sampleType": "Warning",
								"value": {
									"integerValue": -1,
									"valueType": "IntegerType"
								}
							},
							{
								"label": "Outbound Octets_cr",
								"sampleType": "Critical",
								"value": {
									"integerValue": -1,
									"valueType": "IntegerType"
								}
							}
						],
						"unit": "1",
						"value": {
							"integerValue": 4571497819,
							"valueType": "IntegerType"
						}
					},
					{
						"interval": {
							"endTime": "1696286942511",
							"startTime": "1696286942511"
						},
						"metricName": "ifHCInOctets",
						"sampleType": "Value",
						"thresholds": [
							{
								"label": "ifHCInOctets_wn",
								"sampleType": "Warning",
								"value": {
									"integerValue": -1,
									"valueType": "IntegerType"
								}
							},
							{
								"label": "ifHCInOctets_cr",
								"sampleType": "Critical",
								"value": {
									"integerValue": -1,
									"valueType": "IntegerType"
								}
							}
						],
						"unit": "1",
						"value": {
							"integerValue": 1117565447404,
							"valueType": "IntegerType"
						}
					},
					{
						"interval": {
							"endTime": "1696286942511",
							"startTime": "1696286942511"
						},
						"metricName": "ifHCOutOctets",
						"sampleType": "Value",
						"thresholds": [
							{
								"label": "ifHCOutOctets_wn",
								"sampleType": "Warning",
								"value": {
									"integerValue": -1,
									"valueType": "IntegerType"
								}
							},
							{
								"label": "ifHCOutOctets_cr",
								"sampleType": "Critical",
								"value": {
									"integerValue": -1,
									"valueType": "IntegerType"
								}
							}
						],
						"unit": "1",
						"value": {
							"integerValue": 1056838668599,
							"valueType": "IntegerType"
						}
					},
					{
						"interval": {
							"endTime": "1696286942511",
							"startTime": "1696286942511"
						},
						"metricName": "Interface Speed",
						"sampleType": "Value",
						"thresholds": [
							{
								"label": "Interface Speed_wn",
								"sampleType": "Warning",
								"value": {
									"integerValue": -1,
									"valueType": "IntegerType"
								}
							},
							{
								"label": "Interface Speed_cr",
								"sampleType": "Critical",
								"value": {
									"integerValue": -1,
									"valueType": "IntegerType"
								}
							}
						],
						"unit": "1",
						"value": {
							"integerValue": 100000000,
							"valueType": "IntegerType"
						}
					},
					{
						"interval": {
							"endTime": "1696286942511",
							"startTime": "1696286942511"
						},
						"metricName": "Inbound Octets",
						"sampleType": "Value",
						"thresholds": [
							{
								"label": "Inbound Octets_wn",
								"sampleType": "Warning",
								"value": {
									"integerValue": -1,
									"valueType": "IntegerType"
								}
							},
							{
								"label": "Inbound Octets_cr",
								"sampleType": "Critical",
								"value": {
									"integerValue": -1,
									"valueType": "IntegerType"
								}
							}
						],
						"unit": "1",
						"value": {
							"integerValue": 5168872940,
							"valueType": "IntegerType"
						}
					}
				],
				"name": "Fa0/0.1003",
				"nextCheckTime": "1696287002511",
				"owner": "c2801",
				"status": "SERVICE_OK",
				"type": "service"
			},
			{
				"lastCheckTime": "1696286942511",
				"lastPluginOutput": "Interface Operational State is UP, Administrative state is UP",
				"metrics": [
					{
						"interval": {
							"endTime": "1696286942511",
							"startTime": "1696286942511"
						},
						"metricName": "ifHCOutOctets",
						"sampleType": "Value",
						"thresholds": [
							{
								"label": "ifHCOutOctets_wn",
								"sampleType": "Warning",
								"value": {
									"integerValue": -1,
									"valueType": "IntegerType"
								}
							},
							{
								"label": "ifHCOutOctets_cr",
								"sampleType": "Critical",
								"value": {
									"integerValue": -1,
									"valueType": "IntegerType"
								}
							}
						],
						"unit": "1",
						"value": {
							"integerValue": 701607891810,
							"valueType": "IntegerType"
						}
					},
					{
						"interval": {
							"endTime": "1696286942511",
							"startTime": "1696286942511"
						},
						"metricName": "Interface Speed",
						"sampleType": "Value",
						"thresholds": [
							{
								"label": "Interface Speed_wn",
								"sampleType": "Warning",
								"value": {
									"integerValue": -1,
									"valueType": "IntegerType"
								}
							},
							{
								"label": "Interface Speed_cr",
								"sampleType": "Critical",
								"value": {
									"integerValue": -1,
									"valueType": "IntegerType"
								}
							}
						],
						"unit": "1",
						"value": {
							"integerValue": 100000000,
							"valueType": "IntegerType"
						}
					},
					{
						"interval": {
							"endTime": "1696286942511",
							"startTime": "1696286942511"
						},
						"metricName": "Inbound Octets",
						"sampleType": "Value",
						"thresholds": [
							{
								"label": "Inbound Octets_wn",
								"sampleType": "Warning",
								"value": {
									"integerValue": -1,
									"valueType": "IntegerType"
								}
							},
							{
								"label": "Inbound Octets_cr",
								"sampleType": "Critical",
								"value": {
									"integerValue": -1,
									"valueType": "IntegerType"
								}
							}
						],
						"unit": "1",
						"value": {
							"integerValue": 12516361641,
							"valueType": "IntegerType"
						}
					},
					{
						"interval": {
							"endTime": "1696286942511",
							"startTime": "1696286942511"
						},
						"metricName": "Outbound Octets",
						"sampleType": "Value",
						"thresholds": [
							{
								"label": "Outbound Octets_wn",
								"sampleType": "Warning",
								"value": {
									"integerValue": -1,
									"valueType": "IntegerType"
								}
							},
							{
								"label": "Outbound Octets_cr",
								"sampleType": "Critical",
								"value": {
									"integerValue": -1,
									"valueType": "IntegerType"
								}
							}
						],
						"unit": "1",
						"value": {
							"integerValue": 10118157154,
							"valueType": "IntegerType"
						}
					},
					{
						"interval": {
							"endTime": "1696286942511",
							"startTime": "1696286942511"
						},
						"metricName": "ifHCInOctets",
						"sampleType": "Value",
						"thresholds": [
							{
								"label": "ifHCInOctets_wn",
								"sampleType": "Warning",
								"value": {
									"integerValue": -1,
									"valueType": "IntegerType"
								}
							},
							{
								"label": "ifHCInOctets_cr",
								"sampleType": "Critical",
								"value": {
									"integerValue": -1,
									"valueType": "IntegerType"
								}
							}
						],
						"unit": "1",
						"value": {
							"integerValue": 433423156649,
							"valueType": "IntegerType"
						}
					}
				],
				"name": "Fa0/1.1002",
				"nextCheckTime": "1696287002511",
				"owner": "c2801",
				"status": "SERVICE_OK",
				"type": "service"
			},
			{
				"lastCheckTime": "1696286942511",
				"lastPluginOutput": "Interface Operational State is UP, Administrative state is UP",
				"metrics": [
					{
						"interval": {
							"endTime": "1696286942511",
							"startTime": "1696286942511"
						},
						"metricName": "ifHCOutOctets",
						"sampleType": "Value",
						"thresholds": [
							{
								"label": "ifHCOutOctets_wn",
								"sampleType": "Warning",
								"value": {
									"integerValue": -1,
									"valueType": "IntegerType"
								}
							},
							{
								"label": "ifHCOutOctets_cr",
								"sampleType": "Critical",
								"value": {
									"integerValue": -1,
									"valueType": "IntegerType"
								}
							}
						],
						"unit": "1",
						"value": {
							"integerValue": 1060101949781,
							"valueType": "IntegerType"
						}
					},
					{
						"interval": {
							"endTime": "1696286942511",
							"startTime": "1696286942511"
						},
						"metricName": "Interface Speed",
						"sampleType": "Value",
						"thresholds": [
							{
								"label": "Interface Speed_wn",
								"sampleType": "Warning",
								"value": {
									"integerValue": -1,
									"valueType": "IntegerType"
								}
							},
							{
								"label": "Interface Speed_cr",
								"sampleType": "Critical",
								"value": {
									"integerValue": -1,
									"valueType": "IntegerType"
								}
							}
						],
						"unit": "1",
						"value": {
							"integerValue": 100000000,
							"valueType": "IntegerType"
						}
					},
					{
						"interval": {
							"endTime": "1696286942511",
							"startTime": "1696286942511"
						},
						"metricName": "Outbound Errors",
						"sampleType": "Value",
						"thresholds": [
							{
								"label": "Outbound Errors_wn",
								"sampleType": "Warning",
								"value": {
									"integerValue": -1,
									"valueType": "IntegerType"
								}
							},
							{
								"label": "Outbound Errors_cr",
								"sampleType": "Critical",
								"value": {
									"integerValue": -1,
									"valueType": "IntegerType"
								}
							}
						],
						"unit": "1",
						"value": {
							"integerValue": 0,
							"valueType": "IntegerType"
						}
					},
					{
						"interval": {
							"endTime": "1696286942511",
							"startTime": "1696286942511"
						},
						"metricName": "Inbound Discards",
						"sampleType": "Value",
						"thresholds": [
							{
								"label": "Inbound Discards_wn",
								"sampleType": "Warning",
								"value": {
									"integerValue": -1,
									"valueType": "IntegerType"
								}
							},
							{
								"label": "Inbound Discards_cr",
								"sampleType": "Critical",
								"value": {
									"integerValue": -1,
									"valueType": "IntegerType"
								}
							}
						],
						"unit": "1",
						"value": {
							"integerValue": 0,
							"valueType": "IntegerType"
						}
					},
					{
						"interval": {
							"endTime": "1696286942511",
							"startTime": "1696286942511"
						},
						"metricName": "Outbound Discards",
						"sampleType": "Value",
						"thresholds": [
							{
								"label": "Outbound Discards_wn",
								"sampleType": "Warning",
								"value": {
									"integerValue": -1,
									"valueType": "IntegerType"
								}
							},
							{
								"label": "Outbound Discards_cr",
								"sampleType": "Critical",
								"value": {
									"integerValue": -1,
									"valueType": "IntegerType"
								}
							}
						],
						"unit": "1",
						"value": {
							"integerValue": 0,
							"valueType": "IntegerType"
						}
					},
					{
						"interval": {
							"endTime": "1696286942511",
							"startTime": "1696286942511"
						},
						"metricName": "Outbound Octets",
						"sampleType": "Value",
						"thresholds": [
							{
								"label": "Outbound Octets_wn",
								"sampleType": "Warning",
								"value": {
									"integerValue": -1,
									"valueType": "IntegerType"
								}
							},
							{
								"label": "Outbound Octets_cr",
								"sampleType": "Critical",
								"value": {
									"integerValue": -1,
									"valueType": "IntegerType"
								}
							}
						],
						"unit": "1",
						"value": {
							"integerValue": 7834935633,
							"valueType": "IntegerType"
						}
					},
					{
						"interval": {
							"endTime": "1696286942511",
							"startTime": "1696286942511"
						},
						"metricName": "ifHCInOctets",
						"sampleType": "Value",
						"thresholds": [
							{
								"label": "ifHCInOctets_wn",
								"sampleType": "Warning",
								"value": {
									"integerValue": -1,
									"valueType": "IntegerType"
								}
							},
							{
								"label": "ifHCInOctets_cr",
								"sampleType": "Critical",
								"value": {
									"integerValue": -1,
									"valueType": "IntegerType"
								}
							}
						],
						"unit": "1",
						"value": {
							"integerValue": 1127994685160,
							"valueType": "IntegerType"
						}
					},
					{
						"interval": {
							"endTime": "1696286942511",
							"startTime": "1696286942511"
						},
						"metricName": "Inbound Errors",
						"sampleType": "Value",
						"thresholds": [
							{
								"label": "Inbound Errors_wn",
								"sampleType": "Warning",
								"value": {
									"integerValue": -1,
									"valueType": "IntegerType"
								}
							},
							{
								"label": "Inbound Errors_cr",
								"sampleType": "Critical",
								"value": {
									"integerValue": -1,
									"valueType": "IntegerType"
								}
							}
						],
						"unit": "1",
						"value": {
							"integerValue": 0,
							"valueType": "IntegerType"
						}
					},
					{
						"interval": {
							"endTime": "1696286942511",
							"startTime": "1696286942511"
						},
						"metricName": "Inbound Octets",
						"sampleType": "Value",
						"thresholds": [
							{
								"label": "Inbound Octets_wn",
								"sampleType": "Warning",
								"value": {
									"integerValue": -1,
									"valueType": "IntegerType"
								}
							},
							{
								"label": "Inbound Octets_cr",
								"sampleType": "Critical",
								"value": {
									"integerValue": -1,
									"valueType": "IntegerType"
								}
							}
						],
						"unit": "1",
						"value": {
							"integerValue": 15598148020,
							"valueType": "IntegerType"
						}
					}
				],
				"name": "Fa0/0",
				"nextCheckTime": "1696287002511",
				"owner": "c2801",
				"status": "SERVICE_OK",
				"type": "service"
			},
			{
				"lastCheckTime": "1696286942511",
				"lastPluginOutput": "Interface Operational State is UP, Administrative state is UP",
				"metrics": [
					{
						"interval": {
							"endTime": "1696286942511",
							"startTime": "1696286942511"
						},
						"metricName": "Inbound Errors",
						"sampleType": "Value",
						"thresholds": [
							{
								"label": "Inbound Errors_wn",
								"sampleType": "Warning",
								"value": {
									"integerValue": -1,
									"valueType": "IntegerType"
								}
							},
							{
								"label": "Inbound Errors_cr",
								"sampleType": "Critical",
								"value": {
									"integerValue": -1,
									"valueType": "IntegerType"
								}
							}
						],
						"unit": "1",
						"value": {
							"integerValue": 0,
							"valueType": "IntegerType"
						}
					},
					{
						"interval": {
							"endTime": "1696286942511",
							"startTime": "1696286942511"
						},
						"metricName": "Inbound Octets",
						"sampleType": "Value",
						"thresholds": [
							{
								"label": "Inbound Octets_wn",
								"sampleType": "Warning",
								"value": {
									"integerValue": -1,
									"valueType": "IntegerType"
								}
							},
							{
								"label": "Inbound Octets_cr",
								"sampleType": "Critical",
								"value": {
									"integerValue": -1,
									"valueType": "IntegerType"
								}
							}
						],
						"unit": "1",
						"value": {
							"integerValue": 12516412475,
							"valueType": "IntegerType"
						}
					},
					{
						"interval": {
							"endTime": "1696286942511",
							"startTime": "1696286942511"
						},
						"metricName": "Inbound Discards",
						"sampleType": "Value",
						"thresholds": [
							{
								"label": "Inbound Discards_wn",
								"sampleType": "Warning",
								"value": {
									"integerValue": -1,
									"valueType": "IntegerType"
								}
							},
							{
								"label": "Inbound Discards_cr",
								"sampleType": "Critical",
								"value": {
									"integerValue": -1,
									"valueType": "IntegerType"
								}
							}
						],
						"unit": "1",
						"value": {
							"integerValue": 0,
							"valueType": "IntegerType"
						}
					},
					{
						"interval": {
							"endTime": "1696286942511",
							"startTime": "1696286942511"
						},
						"metricName": "Outbound Discards",
						"sampleType": "Value",
						"thresholds": [
							{
								"label": "Outbound Discards_wn",
								"sampleType": "Warning",
								"value": {
									"integerValue": -1,
									"valueType": "IntegerType"
								}
							},
							{
								"label": "Outbound Discards_cr",
								"sampleType": "Critical",
								"value": {
									"integerValue": -1,
									"valueType": "IntegerType"
								}
							}
						],
						"unit": "1",
						"value": {
							"integerValue": 0,
							"valueType": "IntegerType"
						}
					},
					{
						"interval": {
							"endTime": "1696286942511",
							"startTime": "1696286942511"
						},
						"metricName": "Outbound Octets",
						"sampleType": "Value",
						"thresholds": [
							{
								"label": "Outbound Octets_wn",
								"sampleType": "Warning",
								"value": {
									"integerValue": -1,
									"valueType": "IntegerType"
								}
							},
							{
								"label": "Outbound Octets_cr",
								"sampleType": "Critical",
								"value": {
									"integerValue": -1,
									"valueType": "IntegerType"
								}
							}
						],
						"unit": "1",
						"value": {
							"integerValue": 13599427499,
							"valueType": "IntegerType"
						}
					},
					{
						"interval": {
							"endTime": "1696286942511",
							"startTime": "1696286942511"
						},
						"metricName": "ifHCInOctets",
						"sampleType": "Value",
						"thresholds": [
							{
								"label": "ifHCInOctets_wn",
								"sampleType": "Warning",
								"value": {
									"integerValue": -1,
									"valueType": "IntegerType"
								}
							},
							{
								"label": "ifHCInOctets_cr",
								"sampleType": "Critical",
								"value": {
									"integerValue": -1,
									"valueType": "IntegerType"
								}
							}
						],
						"unit": "1",
						"value": {
							"integerValue": 433423207483,
							"valueType": "IntegerType"
						}
					},
					{
						"interval": {
							"endTime": "1696286942511",
							"startTime": "1696286942511"
						},
						"metricName": "ifHCOutOctets",
						"sampleType": "Value",
						"thresholds": [
							{
								"label": "ifHCOutOctets_wn",
								"sampleType": "Warning",
								"value": {
									"integerValue": -1,
									"valueType": "IntegerType"
								}
							},
							{
								"label": "ifHCOutOctets_cr",
								"sampleType": "Critical",
								"value": {
									"integerValue": -1,
									"valueType": "IntegerType"
								}
							}
						],
						"unit": "1",
						"value": {
							"integerValue": 705089162575,
							"valueType": "IntegerType"
						}
					},
					{
						"interval": {
							"endTime": "1696286942511",
							"startTime": "1696286942511"
						},
						"metricName": "Interface Speed",
						"sampleType": "Value",
						"thresholds": [
							{
								"label": "Interface Speed_wn",
								"sampleType": "Warning",
								"value": {
									"integerValue": -1,
									"valueType": "IntegerType"
								}
							},
							{
								"label": "Interface Speed_cr",
								"sampleType": "Critical",
								"value": {
									"integerValue": -1,
									"valueType": "IntegerType"
								}
							}
						],
						"unit": "1",
						"value": {
							"integerValue": 100000000,
							"valueType": "IntegerType"
						}
					},
					{
						"interval": {
							"endTime": "1696286942511",
							"startTime": "1696286942511"
						},
						"metricName": "Outbound Errors",
						"sampleType": "Value",
						"thresholds": [
							{
								"label": "Outbound Errors_wn",
								"sampleType": "Warning",
								"value": {
									"integerValue": -1,
									"valueType": "IntegerType"
								}
							},
							{
								"label": "Outbound Errors_cr",
								"sampleType": "Critical",
								"value": {
									"integerValue": -1,
									"valueType": "IntegerType"
								}
							}
						],
						"unit": "1",
						"value": {
							"integerValue": 0,
							"valueType": "IntegerType"
						}
					}
				],
				"name": "Fa0/1",
				"nextCheckTime": "1696287002511",
				"owner": "c2801",
				"status": "SERVICE_OK",
				"type": "service"
			},
			{
				"lastCheckTime": "1696286942511",
				"lastPluginOutput": "Interface Operational State is UP, Administrative state is UP",
				"metrics": [
					{
						"interval": {
							"endTime": "1696286942511",
							"startTime": "1696286942511"
						},
						"metricName": "Inbound Discards",
						"sampleType": "Value",
						"thresholds": [
							{
								"label": "Inbound Discards_wn",
								"sampleType": "Warning",
								"value": {
									"integerValue": -1,
									"valueType": "IntegerType"
								}
							},
							{
								"label": "Inbound Discards_cr",
								"sampleType": "Critical",
								"value": {
									"integerValue": -1,
									"valueType": "IntegerType"
								}
							}
						],
						"unit": "1",
						"value": {
							"integerValue": 0,
							"valueType": "IntegerType"
						}
					},
					{
						"interval": {
							"endTime": "1696286942511",
							"startTime": "1696286942511"
						},
						"metricName": "Inbound Errors",
						"sampleType": "Value",
						"thresholds": [
							{
								"label": "Inbound Errors_wn",
								"sampleType": "Warning",
								"value": {
									"integerValue": -1,
									"valueType": "IntegerType"
								}
							},
							{
								"label": "Inbound Errors_cr",
								"sampleType": "Critical",
								"value": {
									"integerValue": -1,
									"valueType": "IntegerType"
								}
							}
						],
						"unit": "1",
						"value": {
							"integerValue": 0,
							"valueType": "IntegerType"
						}
					},
					{
						"interval": {
							"endTime": "1696286942511",
							"startTime": "1696286942511"
						},
						"metricName": "Outbound Discards",
						"sampleType": "Value",
						"thresholds": [
							{
								"label": "Outbound Discards_wn",
								"sampleType": "Warning",
								"value": {
									"integerValue": -1,
									"valueType": "IntegerType"
								}
							},
							{
								"label": "Outbound Discards_cr",
								"sampleType": "Critical",
								"value": {
									"integerValue": -1,
									"valueType": "IntegerType"
								}
							}
						],
						"unit": "1",
						"value": {
							"integerValue": 0,
							"valueType": "IntegerType"
						}
					},
					{
						"interval": {
							"endTime": "1696286942511",
							"startTime": "1696286942511"
						},
						"metricName": "Inbound Octets",
						"sampleType": "Value",
						"thresholds": [
							{
								"label": "Inbound Octets_wn",
								"sampleType": "Warning",
								"value": {
									"integerValue": -1,
									"valueType": "IntegerType"
								}
							},
							{
								"label": "Inbound Octets_cr",
								"sampleType": "Critical",
								"value": {
									"integerValue": -1,
									"valueType": "IntegerType"
								}
							}
						],
						"unit": "1",
						"value": {
							"integerValue": 18957178352,
							"valueType": "IntegerType"
						}
					},
					{
						"interval": {
							"endTime": "1696286942511",
							"startTime": "1696286942511"
						},
						"metricName": "Outbound Octets",
						"sampleType": "Value",
						"thresholds": [
							{
								"label": "Outbound Octets_wn",
								"sampleType": "Warning",
								"value": {
									"integerValue": -1,
									"valueType": "IntegerType"
								}
							},
							{
								"label": "Outbound Octets_cr",
								"sampleType": "Critical",
								"value": {
									"integerValue": -1,
									"valueType": "IntegerType"
								}
							}
						],
						"unit": "1",
						"value": {
							"integerValue": 0,
							"valueType": "IntegerType"
						}
					},
					{
						"interval": {
							"endTime": "1696286942511",
							"startTime": "1696286942511"
						},
						"metricName": "Interface Speed",
						"sampleType": "Value",
						"thresholds": [
							{
								"label": "Interface Speed_wn",
								"sampleType": "Warning",
								"value": {
									"integerValue": -1,
									"valueType": "IntegerType"
								}
							},
							{
								"label": "Interface Speed_cr",
								"sampleType": "Critical",
								"value": {
									"integerValue": -1,
									"valueType": "IntegerType"
								}
							}
						],
						"unit": "1",
						"value": {
							"integerValue": 4294967295,
							"valueType": "IntegerType"
						}
					},
					{
						"interval": {
							"endTime": "1696286942511",
							"startTime": "1696286942511"
						},
						"metricName": "Outbound Errors",
						"sampleType": "Value",
						"thresholds": [
							{
								"label": "Outbound Errors_wn",
								"sampleType": "Warning",
								"value": {
									"integerValue": -1,
									"valueType": "IntegerType"
								}
							},
							{
								"label": "Outbound Errors_cr",
								"sampleType": "Critical",
								"value": {
									"integerValue": -1,
									"valueType": "IntegerType"
								}
							}
						],
						"unit": "1",
						"value": {
							"integerValue": 0,
							"valueType": "IntegerType"
						}
					}
				],
				"name": "Nu0",
				"nextCheckTime": "1696287002511",
				"owner": "c2801",
				"status": "SERVICE_OK",
				"type": "service"
			},
			{
				"lastCheckTime": "1696286942511",
				"lastPluginOutput": "Interface Operational State is UP, Administrative state is UP",
				"metrics": [
					{
						"interval": {
							"endTime": "1696286942511",
							"startTime": "1696286942511"
						},
						"metricName": "Outbound Octets",
						"sampleType": "Value",
						"thresholds": [
							{
								"label": "Outbound Octets_wn",
								"sampleType": "Warning",
								"value": {
									"integerValue": -1,
									"valueType": "IntegerType"
								}
							},
							{
								"label": "Outbound Octets_cr",
								"sampleType": "Critical",
								"value": {
									"integerValue": -1,
									"valueType": "IntegerType"
								}
							}
						],
						"unit": "1",
						"value": {
							"integerValue": 0,
							"valueType": "IntegerType"
						}
					},
					{
						"interval": {
							"endTime": "1696286942511",
							"startTime": "1696286942511"
						},
						"metricName": "ifHCInOctets",
						"sampleType": "Value",
						"thresholds": [
							{
								"label": "ifHCInOctets_wn",
								"sampleType": "Warning",
								"value": {
									"integerValue": -1,
									"valueType": "IntegerType"
								}
							},
							{
								"label": "ifHCInOctets_cr",
								"sampleType": "Critical",
								"value": {
									"integerValue": -1,
									"valueType": "IntegerType"
								}
							}
						],
						"unit": "1",
						"value": {
							"integerValue": 0,
							"valueType": "IntegerType"
						}
					},
					{
						"interval": {
							"endTime": "1696286942511",
							"startTime": "1696286942511"
						},
						"metricName": "ifHCOutOctets",
						"sampleType": "Value",
						"thresholds": [
							{
								"label": "ifHCOutOctets_wn",
								"sampleType": "Warning",
								"value": {
									"integerValue": -1,
									"valueType": "IntegerType"
								}
							},
							{
								"label": "ifHCOutOctets_cr",
								"sampleType": "Critical",
								"value": {
									"integerValue": -1,
									"valueType": "IntegerType"
								}
							}
						],
						"unit": "1",
						"value": {
							"integerValue": 0,
							"valueType": "IntegerType"
						}
					},
					{
						"interval": {
							"endTime": "1696286942511",
							"startTime": "1696286942511"
						},
						"metricName": "Interface Speed",
						"sampleType": "Value",
						"thresholds": [
							{
								"label": "Interface Speed_wn",
								"sampleType": "Warning",
								"value": {
									"integerValue": -1,
									"valueType": "IntegerType"
								}
							},
							{
								"label": "Interface Speed_cr",
								"sampleType": "Critical",
								"value": {
									"integerValue": -1,
									"valueType": "IntegerType"
								}
							}
						],
						"unit": "1",
						"value": {
							"integerValue": 4294967295,
							"valueType": "IntegerType"
						}
					},
					{
						"interval": {
							"endTime": "1696286942511",
							"startTime": "1696286942511"
						},
						"metricName": "Outbound Errors",
						"sampleType": "Value",
						"thresholds": [
							{
								"label": "Outbound Errors_wn",
								"sampleType": "Warning",
								"value": {
									"integerValue": -1,
									"valueType": "IntegerType"
								}
							},
							{
								"label": "Outbound Errors_cr",
								"sampleType": "Critical",
								"value": {
									"integerValue": -1,
									"valueType": "IntegerType"
								}
							}
						],
						"unit": "1",
						"value": {
							"integerValue": 0,
							"valueType": "IntegerType"
						}
					},
					{
						"interval": {
							"endTime": "1696286942511",
							"startTime": "1696286942511"
						},
						"metricName": "Inbound Discards",
						"sampleType": "Value",
						"thresholds": [
							{
								"label": "Inbound Discards_wn",
								"sampleType": "Warning",
								"value": {
									"integerValue": -1,
									"valueType": "IntegerType"
								}
							},
							{
								"label": "Inbound Discards_cr",
								"sampleType": "Critical",
								"value": {
									"integerValue": -1,
									"valueType": "IntegerType"
								}
							}
						],
						"unit": "1",
						"value": {
							"integerValue": 0,
							"valueType": "IntegerType"
						}
					},
					{
						"interval": {
							"endTime": "1696286942511",
							"startTime": "1696286942511"
						},
						"metricName": "Inbound Errors",
						"sampleType": "Value",
						"thresholds": [
							{
								"label": "Inbound Errors_wn",
								"sampleType": "Warning",
								"value": {
									"integerValue": -1,
									"valueType": "IntegerType"
								}
							},
							{
								"label": "Inbound Errors_cr",
								"sampleType": "Critical",
								"value": {
									"integerValue": -1,
									"valueType": "IntegerType"
								}
							}
						],
						"unit": "1",
						"value": {
							"integerValue": 0,
							"valueType": "IntegerType"
						}
					},
					{
						"interval": {
							"endTime": "1696286942511",
							"startTime": "1696286942511"
						},
						"metricName": "Outbound Discards",
						"sampleType": "Value",
						"thresholds": [
							{
								"label": "Outbound Discards_wn",
								"sampleType": "Warning",
								"value": {
									"integerValue": -1,
									"valueType": "IntegerType"
								}
							},
							{
								"label": "Outbound Discards_cr",
								"sampleType": "Critical",
								"value": {
									"integerValue": -1,
									"valueType": "IntegerType"
								}
							}
						],
						"unit": "1",
						"value": {
							"integerValue": 0,
							"valueType": "IntegerType"
						}
					},
					{
						"interval": {
							"endTime": "1696286942511",
							"startTime": "1696286942511"
						},
						"metricName": "Inbound Octets",
						"sampleType": "Value",
						"thresholds": [
							{
								"label": "Inbound Octets_wn",
								"sampleType": "Warning",
								"value": {
									"integerValue": -1,
									"valueType": "IntegerType"
								}
							},
							{
								"label": "Inbound Octets_cr",
								"sampleType": "Critical",
								"value": {
									"integerValue": -1,
									"valueType": "IntegerType"
								}
							}
						],
						"unit": "1",
						"value": {
							"integerValue": 0,
							"valueType": "IntegerType"
						}
					}
				],
				"name": "Lo6774",
				"nextCheckTime": "1696287002511",
				"owner": "c2801",
				"status": "SERVICE_OK",
				"type": "service"
			},
			{
				"lastCheckTime": "1696286942511",
				"lastPluginOutput": "Interface Operational State is UP, Administrative state is UP",
				"metrics": [
					{
						"interval": {
							"endTime": "1696286942511",
							"startTime": "1696286942511"
						},
						"metricName": "Interface Speed",
						"sampleType": "Value",
						"thresholds": [
							{
								"label": "Interface Speed_wn",
								"sampleType": "Warning",
								"value": {
									"integerValue": -1,
									"valueType": "IntegerType"
								}
							},
							{
								"label": "Interface Speed_cr",
								"sampleType": "Critical",
								"value": {
									"integerValue": -1,
									"valueType": "IntegerType"
								}
							}
						],
						"unit": "1",
						"value": {
							"integerValue": 9000,
							"valueType": "IntegerType"
						}
					},
					{
						"interval": {
							"endTime": "1696286942511",
							"startTime": "1696286942511"
						},
						"metricName": "Inbound Errors",
						"sampleType": "Value",
						"thresholds": [
							{
								"label": "Inbound Errors_wn",
								"sampleType": "Warning",
								"value": {
									"integerValue": -1,
									"valueType": "IntegerType"
								}
							},
							{
								"label": "Inbound Errors_cr",
								"sampleType": "Critical",
								"value": {
									"integerValue": -1,
									"valueType": "IntegerType"
								}
							}
						],
						"unit": "1",
						"value": {
							"integerValue": 0,
							"valueType": "IntegerType"
						}
					},
					{
						"interval": {
							"endTime": "1696286942511",
							"startTime": "1696286942511"
						},
						"metricName": "Outbound Discards",
						"sampleType": "Value",
						"thresholds": [
							{
								"label": "Outbound Discards_wn",
								"sampleType": "Warning",
								"value": {
									"integerValue": -1,
									"valueType": "IntegerType"
								}
							},
							{
								"label": "Outbound Discards_cr",
								"sampleType": "Critical",
								"value": {
									"integerValue": -1,
									"valueType": "IntegerType"
								}
							}
						],
						"unit": "1",
						"value": {
							"integerValue": 0,
							"valueType": "IntegerType"
						}
					},
					{
						"interval": {
							"endTime": "1696286942511",
							"startTime": "1696286942511"
						},
						"metricName": "Outbound Octets",
						"sampleType": "Value",
						"thresholds": [
							{
								"label": "Outbound Octets_wn",
								"sampleType": "Warning",
								"value": {
									"integerValue": -1,
									"valueType": "IntegerType"
								}
							},
							{
								"label": "Outbound Octets_cr",
								"sampleType": "Critical",
								"value": {
									"integerValue": -1,
									"valueType": "IntegerType"
								}
							}
						],
						"unit": "1",
						"value": {
							"integerValue": 3422103643,
							"valueType": "IntegerType"
						}
					},
					{
						"interval": {
							"endTime": "1696286942511",
							"startTime": "1696286942511"
						},
						"metricName": "ifHCOutOctets",
						"sampleType": "Value",
						"thresholds": [
							{
								"label": "ifHCOutOctets_wn",
								"sampleType": "Warning",
								"value": {
									"integerValue": -1,
									"valueType": "IntegerType"
								}
							},
							{
								"label": "ifHCOutOctets_cr",
								"sampleType": "Critical",
								"value": {
									"integerValue": -1,
									"valueType": "IntegerType"
								}
							}
						],
						"unit": "1",
						"value": {
							"integerValue": 514523244431,
							"valueType": "IntegerType"
						}
					},
					{
						"interval": {
							"endTime": "1696286942511",
							"startTime": "1696286942511"
						},
						"metricName": "Outbound Errors",
						"sampleType": "Value",
						"thresholds": [
							{
								"label": "Outbound Errors_wn",
								"sampleType": "Warning",
								"value": {
									"integerValue": -1,
									"valueType": "IntegerType"
								}
							},
							{
								"label": "Outbound Errors_cr",
								"sampleType": "Critical",
								"value": {
									"integerValue": -1,
									"valueType": "IntegerType"
								}
							}
						],
						"unit": "1",
						"value": {
							"integerValue": 0,
							"valueType": "IntegerType"
						}
					},
					{
						"interval": {
							"endTime": "1696286942511",
							"startTime": "1696286942511"
						},
						"metricName": "Inbound Discards",
						"sampleType": "Value",
						"thresholds": [
							{
								"label": "Inbound Discards_wn",
								"sampleType": "Warning",
								"value": {
									"integerValue": -1,
									"valueType": "IntegerType"
								}
							},
							{
								"label": "Inbound Discards_cr",
								"sampleType": "Critical",
								"value": {
									"integerValue": -1,
									"valueType": "IntegerType"
								}
							}
						],
						"unit": "1",
						"value": {
							"integerValue": 0,
							"valueType": "IntegerType"
						}
					},
					{
						"interval": {
							"endTime": "1696286942511",
							"startTime": "1696286942511"
						},
						"metricName": "Inbound Octets",
						"sampleType": "Value",
						"thresholds": [
							{
								"label": "Inbound Octets_wn",
								"sampleType": "Warning",
								"value": {
									"integerValue": -1,
									"valueType": "IntegerType"
								}
							},
							{
								"label": "Inbound Octets_cr",
								"sampleType": "Critical",
								"value": {
									"integerValue": -1,
									"valueType": "IntegerType"
								}
							}
						],
						"unit": "1",
						"value": {
							"integerValue": 24994843692,
							"valueType": "IntegerType"
						}
					},
					{
						"interval": {
							"endTime": "1696286942511",
							"startTime": "1696286942511"
						},
						"metricName": "ifHCInOctets",
						"sampleType": "Value",
						"thresholds": [
							{
								"label": "ifHCInOctets_wn",
								"sampleType": "Warning",
								"value": {
									"integerValue": -1,
									"valueType": "IntegerType"
								}
							},
							{
								"label": "ifHCInOctets_cr",
								"sampleType": "Critical",
								"value": {
									"integerValue": -1,
									"valueType": "IntegerType"
								}
							}
						],
						"unit": "1",
						"value": {
							"integerValue": 836743667074,
							"valueType": "IntegerType"
						}
					}
				],
				"name": "Tu0",
				"nextCheckTime": "1696287002511",
				"owner": "c2801",
				"status": "SERVICE_OK",
				"type": "service"
			}
		],
		"status": "HOST_UP",
		"type": "host"
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
	lastOK := float64(time.Now().Unix())
	for k, d := range state.devices {
		d.LastOK = lastOK
		state.devices[k] = d
	}

	mResources := make([]transit.MonitoredResource, 0)
	if err := json.Unmarshal(mResourcesJSON, &mResources); err != nil {
		return nil, nil, nil, err
	}

	return state, metricDefinitions, mResources, nil
}
