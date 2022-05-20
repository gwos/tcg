package main

import (
	"encoding/json"
	"fmt"
	"regexp"
	"testing"

	"github.com/gwos/tcg/sdk/transit"
	"github.com/stretchr/testify/assert"
)

func TestParsePrometheusBody(t *testing.T) {
	input := []byte(`# HELP bytes_per_minute Finance Services bytes transferred over http per minute.
# TYPE bytes_per_minute gauge
bytes_per_minute{critical="48000",group="PrometheusDemo",resource="FinanceServicesGo",service="analytics",warning="45000"} 13561
bytes_per_minute{critical="48000",group="PrometheusDemo",resource="FinanceServicesGo",service="distribution",warning="45000"} 44850
bytes_per_minute{critical="48000",group="PrometheusDemo",resource="FinanceServicesGo",service="sales",warning="45000"} 22339
# HELP requests_per_minute Finance Services http requests per minute.
# TYPE requests_per_minute gauge
requests_per_minute{critical="95",group="PrometheusDemo",resource="FinanceServicesGo",service="analytics",warning="85"} 56
requests_per_minute{critical="95",group="PrometheusDemo",resource="FinanceServicesGo",service="distribution",warning="85"} 86
requests_per_minute{critical="95",group="PrometheusDemo",resource="FinanceServicesGo",service="sales",warning="85"} 62
# HELP response_time Finance Services http response time average over 1 minute.
# TYPE response_time gauge
response_time{critical="2.8",group="PrometheusDemo",resource="FinanceServicesGo",service="analytics",warning="2.5"} 0.3
response_time{critical="2.8",group="PrometheusDemo",resource="FinanceServicesGo",service="distribution",warning="2.5"} 1.4
response_time{critical="2.8",group="PrometheusDemo",resource="FinanceServicesGo",service="sales",warning="2.5"} 1.1
`)

	pmd := promMetricsData{
		data:          input,
		resourceIndex: -1,
	}

	monitoredResources, resourceGroups, err := pmd.parse()
	assert.NoError(t, err)

	rr, err := json.Marshal(monitoredResources)
	assert.NoError(t, err)
	assert.Contains(t, string(rr), `"owner":"FinanceServicesGo"`)
	assert.Contains(t, string(rr), `"metricName":"requests_per_minute"`)
	assert.Contains(t, string(rr), `"value":{"valueType":"DoubleType","doubleValue":56}`)

	gg, err := json.Marshal(resourceGroups)
	assert.NoError(t, err)
	assert.JSONEq(t,
		`[{
			"groupName":"PrometheusDemo",
			"type":"HostGroup",
			"resources":[{
				"name":"FinanceServicesGo",
				"type":"host"
				}]
		}]`,
		string(gg),
	)
}

func TestParsePrometheusBodyWithResourceMappings(t *testing.T) {
	mappings = &transit.Mappings{
		Resource: []transit.Mapping{
			{
				Enabled:     true,
				Source:      "http://locahost:2222/metrics",
				Destination: "MyTestHostName",
				Regexp:      regexp.MustCompile("http://locahost:2222/metrics"),
			},
		},
	}

	input := []byte(`# HELP bytes_per_minute Finance Services bytes transferred over http per minute.
# TYPE bytes_per_minute gauge
bytes_per_minute{critical="48000",group="PrometheusDemo",resource="FinanceServicesGo",service="analytics",warning="45000"} 13561
bytes_per_minute{critical="48000",group="PrometheusDemo",resource="FinanceServicesGo",service="distribution",warning="45000"} 44850
bytes_per_minute{critical="48000",group="PrometheusDemo",resource="FinanceServicesGo",service="sales",warning="45000"} 22339
# HELP requests_per_minute Finance Services http requests per minute.
# TYPE requests_per_minute gauge
requests_per_minute{critical="95",group="PrometheusDemo",resource="FinanceServicesGo",service="analytics",warning="85"} 56
requests_per_minute{critical="95",group="PrometheusDemo",resource="FinanceServicesGo",service="distribution",warning="85"} 86
requests_per_minute{critical="95",group="PrometheusDemo",resource="FinanceServicesGo",service="sales",warning="85"} 62
# HELP response_time Finance Services http response time average over 1 minute.
# TYPE response_time gauge
response_time{critical="2.8",group="PrometheusDemo",resource="FinanceServicesGo",service="analytics",warning="2.5"} 0.3
response_time{critical="2.8",group="PrometheusDemo",resource="FinanceServicesGo",service="distribution",warning="2.5"} 1.4
response_time{critical="2.8",group="PrometheusDemo",resource="FinanceServicesGo",service="sales",warning="2.5"} 1.1
`)

	pmd := promMetricsData{
		resource:      "http://locahost:2222/metrics",
		data:          input,
		resourceIndex: -1,
	}

	monitoredResources, resourceGroups, err := pmd.parse()
	assert.NoError(t, err)

	rr, err := json.Marshal(monitoredResources)
	assert.NoError(t, err)
	assert.Contains(t, string(rr), `"owner":"MyTestHostName"`)
	assert.Contains(t, string(rr), `"metricName":"requests_per_minute"`)
	assert.Contains(t, string(rr), `"value":{"valueType":"DoubleType","doubleValue":56}`)

	gg, err := json.Marshal(resourceGroups)
	assert.NoError(t, err)
	assert.JSONEq(t,
		`[{
			"groupName":"PrometheusDemo",
			"type":"HostGroup",
			"resources":[{
				"name":"MyTestHostName",
				"type":"host"
				}]
		}]`,
		string(gg),
	)
}

func TestParsePrometheusBodyWithHostLabelMappings(t *testing.T) {
	mappings = &transit.Mappings{
		Resource: []transit.Mapping{
			{
				Enabled:     true,
				Source:      "http://locahost:2222/metrics",
				Destination: "MyTestHostName",
				Regexp:      regexp.MustCompile("http://locahost:2222/metrics"),
			},
		},
		HostLabel: []transit.Mapping{
			{
				Enabled:     true,
				Source:      "instance",
				Destination: "MyTestHostNameOverrode",
				Regexp:      regexp.MustCompile("instance"),
			},
		},
		ServiceLabel: []transit.Mapping{
			{
				Enabled:     true,
				Source:      "^eciv",
				Destination: "$1_test",
				Regexp:      regexp.MustCompile("^eciv"),
			},
		},
	}

	input := []byte(`# HELP bytes_per_minute Finance Services bytes transferred over http per minute.
# TYPE bytes_per_minute gauge
bytes_per_minute{critical="48000",group="PrometheusDemo",instance="FinanceServicesGo",ecivres="analytics",warning="45000"} 13561
bytes_per_minute{critical="48000",group="PrometheusDemo",instance="FinanceServicesGo",ecivres="distribution",warning="45000"} 44850
bytes_per_minute{critical="48000",group="PrometheusDemo",instance="FinanceServicesGo",ecivres="sales",warning="45000"} 22339
# HELP requests_per_minute Finance Services http requests per minute.
# TYPE requests_per_minute gauge
requests_per_minute{critical="95",group="PrometheusDemo",instance="FinanceServicesGo",ecivres="analytics",warning="85"} 56
requests_per_minute{critical="95",group="PrometheusDemo",instance="FinanceServicesGo",ecivres="distribution",warning="85"} 86
requests_per_minute{critical="95",group="PrometheusDemo",instance="FinanceServicesGo",ecivres="sales",warning="85"} 62
# HELP response_time Finance Services http response time average over 1 minute.
# TYPE response_time gauge
response_time{critical="2.8",group="PrometheusDemo",instance="FinanceServicesGo",ecivres="analytics",warning="2.5"} 0.3
response_time{critical="2.8",group="PrometheusDemo",instance="FinanceServicesGo",ecivres="distribution",warning="2.5"} 1.4
response_time{critical="2.8",group="PrometheusDemo",instance="FinanceServicesGo",ecivres="sales",warning="2.5"} 1.1
`)

	pmd := promMetricsData{
		resource:      "http://locahost:2222/metrics",
		data:          input,
		resourceIndex: -1,
	}

	monitoredResources, resourceGroups, err := pmd.parse()
	assert.NoError(t, err)

	rr, err := json.Marshal(monitoredResources)
	assert.NoError(t, err)
	fmt.Println(string(rr))
	assert.Contains(t, string(rr), `"owner":"MyTestHostNameOverrode"`)
	assert.Contains(t, string(rr), `"metricName":"requests_per_minute"`)
	assert.Contains(t, string(rr), `"value":{"valueType":"DoubleType","doubleValue":56}`)
	assert.Contains(t, string(rr), `"name":"analytics_test"`)

	gg, err := json.Marshal(resourceGroups)
	assert.NoError(t, err)
	assert.JSONEq(t,
		`[{
			"groupName":"PrometheusDemo",
			"type":"HostGroup",
			"resources":[{
				"name":"MyTestHostNameOverrode",
				"type":"host"
				}]
		}]`,
		string(gg),
	)
}
