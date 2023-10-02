package main

import (
	"cmp"
	"encoding/json"
	"slices"
	"testing"

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
		"LastOK": 1696235042,
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
		"name": "c2801",
		"type": "host",
		"status": "HOST_UNREACHABLE",
		"lastCheckTime": "1696253136855",
		"nextCheckTime": "1696253256855",
		"services": [
			{
				"name": "Nu0",
				"type": "service",
				"owner": "c2801",
				"status": "SERVICE_OK",
				"lastCheckTime": "1696253136855",
				"nextCheckTime": "1696253256855",
				"lastPluginOutput": "Interface Operational State is UP, Administrative state is UP",
				"metrics": [
					{
						"metricName": "Outbound Errors",
						"sampleType": "Value",
						"interval": {
							"endTime": "1696253136855",
							"startTime": "1696253136855"
						},
						"value": {
							"valueType": "IntegerType",
							"integerValue": 0
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
						"metricName": "Interface Speed",
						"sampleType": "Value",
						"interval": {
							"endTime": "1696253136855",
							"startTime": "1696253136855"
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
						"metricName": "Inbound Discards",
						"sampleType": "Value",
						"interval": {
							"endTime": "1696253136855",
							"startTime": "1696253136855"
						},
						"value": {
							"valueType": "IntegerType",
							"integerValue": 0
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
							"endTime": "1696253136855",
							"startTime": "1696253136855"
						},
						"value": {
							"valueType": "IntegerType",
							"integerValue": 0
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
						"metricName": "Outbound Discards",
						"sampleType": "Value",
						"interval": {
							"endTime": "1696253136855",
							"startTime": "1696253136855"
						},
						"value": {
							"valueType": "IntegerType",
							"integerValue": 0
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
					}
				]
			},
			{
				"name": "Lo6774",
				"type": "service",
				"owner": "c2801",
				"status": "SERVICE_OK",
				"lastCheckTime": "1696253136855",
				"nextCheckTime": "1696253256855",
				"lastPluginOutput": "Interface Operational State is UP, Administrative state is UP",
				"metrics": [
					{
						"metricName": "Inbound Discards",
						"sampleType": "Value",
						"interval": {
							"endTime": "1696253136855",
							"startTime": "1696253136855"
						},
						"value": {
							"valueType": "IntegerType",
							"integerValue": 0
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
							"endTime": "1696253136855",
							"startTime": "1696253136855"
						},
						"value": {
							"valueType": "IntegerType",
							"integerValue": 0
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
						"metricName": "Outbound Discards",
						"sampleType": "Value",
						"interval": {
							"endTime": "1696253136855",
							"startTime": "1696253136855"
						},
						"value": {
							"valueType": "IntegerType",
							"integerValue": 0
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
							"endTime": "1696253136855",
							"startTime": "1696253136855"
						},
						"value": {
							"valueType": "IntegerType",
							"integerValue": 0
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
						"metricName": "Interface Speed",
						"sampleType": "Value",
						"interval": {
							"endTime": "1696253136855",
							"startTime": "1696253136855"
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
					}
				]
			},
			{
				"name": "Tu0",
				"type": "service",
				"owner": "c2801",
				"status": "SERVICE_OK",
				"lastCheckTime": "1696253136855",
				"nextCheckTime": "1696253256855",
				"lastPluginOutput": "Interface Operational State is UP, Administrative state is UP",
				"metrics": [
					{
						"metricName": "Inbound Discards",
						"sampleType": "Value",
						"interval": {
							"endTime": "1696253136855",
							"startTime": "1696253136855"
						},
						"value": {
							"valueType": "IntegerType",
							"integerValue": 0
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
							"endTime": "1696253136855",
							"startTime": "1696253136855"
						},
						"value": {
							"valueType": "IntegerType",
							"integerValue": 0
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
						"metricName": "Outbound Discards",
						"sampleType": "Value",
						"interval": {
							"endTime": "1696253136855",
							"startTime": "1696253136855"
						},
						"value": {
							"valueType": "IntegerType",
							"integerValue": 0
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
							"endTime": "1696253136855",
							"startTime": "1696253136855"
						},
						"value": {
							"valueType": "IntegerType",
							"integerValue": 0
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
						"metricName": "Interface Speed",
						"sampleType": "Value",
						"interval": {
							"endTime": "1696253136855",
							"startTime": "1696253136855"
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
					}
				]
			},
			{
				"name": "Fa0/0.1003",
				"type": "service",
				"owner": "c2801",
				"status": "SERVICE_OK",
				"lastCheckTime": "1696253136855",
				"nextCheckTime": "1696253256855",
				"lastPluginOutput": "Interface Operational State is UP, Administrative state is UP",
				"metrics": [
					{
						"metricName": "Interface Speed",
						"sampleType": "Value",
						"interval": {
							"endTime": "1696253136855",
							"startTime": "1696253136855"
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
					}
				]
			},
			{
				"name": "Fa0/1.1002",
				"type": "service",
				"owner": "c2801",
				"status": "SERVICE_OK",
				"lastCheckTime": "1696253136855",
				"nextCheckTime": "1696253256855",
				"lastPluginOutput": "Interface Operational State is UP, Administrative state is UP",
				"metrics": [
					{
						"metricName": "Interface Speed",
						"sampleType": "Value",
						"interval": {
							"endTime": "1696253136855",
							"startTime": "1696253136855"
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
					}
				]
			},
			{
				"name": "Fa0/0",
				"type": "service",
				"owner": "c2801",
				"status": "SERVICE_OK",
				"lastCheckTime": "1696253136855",
				"nextCheckTime": "1696253256855",
				"lastPluginOutput": "Interface Operational State is UP, Administrative state is UP",
				"metrics": [
					{
						"metricName": "Inbound Discards",
						"sampleType": "Value",
						"interval": {
							"endTime": "1696253136855",
							"startTime": "1696253136855"
						},
						"value": {
							"valueType": "IntegerType",
							"integerValue": 0
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
							"endTime": "1696253136855",
							"startTime": "1696253136855"
						},
						"value": {
							"valueType": "IntegerType",
							"integerValue": 0
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
						"metricName": "Outbound Discards",
						"sampleType": "Value",
						"interval": {
							"endTime": "1696253136855",
							"startTime": "1696253136855"
						},
						"value": {
							"valueType": "IntegerType",
							"integerValue": 0
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
							"endTime": "1696253136855",
							"startTime": "1696253136855"
						},
						"value": {
							"valueType": "IntegerType",
							"integerValue": 0
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
						"metricName": "Interface Speed",
						"sampleType": "Value",
						"interval": {
							"endTime": "1696253136855",
							"startTime": "1696253136855"
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
					}
				]
			},
			{
				"name": "Fa0/1",
				"type": "service",
				"owner": "c2801",
				"status": "SERVICE_OK",
				"lastCheckTime": "1696253136855",
				"nextCheckTime": "1696253256855",
				"lastPluginOutput": "Interface Operational State is UP, Administrative state is UP",
				"metrics": [
					{
						"metricName": "Inbound Discards",
						"sampleType": "Value",
						"interval": {
							"endTime": "1696253136855",
							"startTime": "1696253136855"
						},
						"value": {
							"valueType": "IntegerType",
							"integerValue": 0
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
							"endTime": "1696253136855",
							"startTime": "1696253136855"
						},
						"value": {
							"valueType": "IntegerType",
							"integerValue": 0
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
						"metricName": "Outbound Discards",
						"sampleType": "Value",
						"interval": {
							"endTime": "1696253136855",
							"startTime": "1696253136855"
						},
						"value": {
							"valueType": "IntegerType",
							"integerValue": 0
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
							"endTime": "1696253136855",
							"startTime": "1696253136855"
						},
						"value": {
							"valueType": "IntegerType",
							"integerValue": 0
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
						"metricName": "Interface Speed",
						"sampleType": "Value",
						"interval": {
							"endTime": "1696253136855",
							"startTime": "1696253136855"
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
	mResources := make([]transit.MonitoredResource, 0)
	if err := json.Unmarshal(mResourcesJSON, &mResources); err != nil {
		return nil, nil, nil, err
	}

	return state, metricDefinitions, mResources, nil
}
