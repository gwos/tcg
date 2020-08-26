package main

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/gwos/tcg/connectors"
	"github.com/gwos/tcg/transit"
	"github.com/stretchr/testify/assert"
)

func TestUnmarshalConfig(t *testing.T) {
	expected := ExtConfig{
		Schedule: []ScheduleTask{
			{
				Command: []string{"/_workspace/gwos/loadtest/kahuna/kahuna", "-stdout"},
				Cron:    "* * * * * *",
			},
			{
				Command:     []string{"sh", "-c", "echo \"S;$(date +%s);${XHOST};${XSERVICE};0;OK - very good| ${XMETRIC}=20;85;95\""},
				Cron:        "*/2 * * * * *",
				Environment: []string{"XHOST=test-1", "XSERVICE=test-service-1", "XMETRIC=test-metric"},
			},
			{
				CombinedOutput: true,
				Command:        []string{"sh", "-c", "sleep 3; date --rfc-3339=ns; echo $ENV1 1>&2"},
				Cron:           "*/3 * * * * *",
				Environment:    []string{"ENV1=env 1"},
			},
		},
	}

	data := []byte(`
	{
		"agentId": "99998888-7777-6666-a3b0-b14622f7dd39",
		"monitorConnection": {
			"extensions": {
				"schedule": [
					{
						"command": ["/_workspace/gwos/loadtest/kahuna/kahuna", "-stdout"],
						"cron": "* * * * * *"
					},
					{
						"command": ["sh", "-c", "echo \"S;$(date +%s);${XHOST};${XSERVICE};0;OK - very good| ${XMETRIC}=20;85;95\""],
						"cron": "*/2 * * * * *",
						"environment": ["XHOST=test-1", "XSERVICE=test-service-1", "XMETRIC=test-metric"]
					},
					{
						"combinedOutput": true,
						"command": ["sh", "-c", "sleep 3; date --rfc-3339=ns; echo $ENV1 1>&2"],
						"cron": "*/3 * * * * *",
						"environment": ["ENV1=env 1"]
					}
				]
			}
		}
	}`)
	extConfig := &ExtConfig{}
	metricsProfile := &transit.MetricsProfile{}
	monitorConnection := &connectors.MonitorConnection{
		Extensions: extConfig,
	}

	assert.NoError(t, connectors.UnmarshalConfig(data, metricsProfile, monitorConnection))
	if !reflect.DeepEqual(*extConfig, expected) {
		t.Errorf("ExtConfig actual:\n%v\nexpected:\n%v", *extConfig, expected)
	}
}

func TestExtConfigValidateFails(t *testing.T) {
	data := []byte(`
	{
		"monitorConnection": {
			"extensions": {
				"schedule": [
					{
						"command": ["/_workspace/gwos/loadtest/kahuna/kahuna", "-stdout"],
						"cron": "* * * * * *"
					},
					{
						"cron": "* * * * * *"
					}
				]
			}
		}
	}`)
	extConfig := &ExtConfig{}
	metricsProfile := &transit.MetricsProfile{}
	monitorConnection := &connectors.MonitorConnection{
		Extensions: extConfig,
	}

	assert.NoError(t, connectors.UnmarshalConfig(data, metricsProfile, monitorConnection))
	actualErr := extConfig.Validate()
	expectedErr := fmt.Errorf("ExtConfig Schedule item error: Command is empty")
	if assert.Error(t, actualErr) {
		assert.Equal(t, expectedErr, actualErr)
	}
}

func TestNscaParser(t *testing.T) {
	data := []byte(
		`Server1;Disks1;0;CRITICAL - load average: 2.45, 2.32, 2.22|load1=2.450;0.000;0.000;0;
		 Server2;Disks2;1;CRITICAL - load average: 2.45, 2.32, 2.22|load1=2.450;0.000;0.000;0; load5=2.320;0.000;0.000;0; load15=2.220;0.000;0.000;0;
		 Server3;Disks3;2;CRITICAL - load average: 2.45, 2.32, 2.22|load1=2.450;0.000;0.000;0; load5=2.320;0.000;0.000;0;`,
	)

	monitoredResources, err := parseBody(data, NSCA)
	assert.NoError(t, err)

	if len(*monitoredResources) != 3 {
		assert.Fail(t, "invalid count of monitored resources")
	}

	if len((*monitoredResources)[0].Services) != 1 {
		assert.Fail(t, "invalid count of services for monitored resource")
	}

	if len((*monitoredResources)[1].Services[0].Metrics) != 3 {
		assert.Fail(t, "invalid count of metrics for service")
	}
}

func TestBronxParser(t *testing.T) {
	data := []byte(
		`S;1596719076;Server1;Disk1;0;OK| test-metric=44;85;95
		 S;1596719076;Server2;Disk2;1;WARNING| test-metric=44;85;95
		 S;1596719076;Server3;Disk3;2;CRITICAL| test-metric=44;85;95`,
	)

	monitoredResources, err := parseBody(data, Bronx)
	assert.NoError(t, err)

	if len(*monitoredResources) != 3 {
		assert.Fail(t, "invalid count of monitored resources")
	}

	if len((*monitoredResources)[0].Services) != 1 {
		assert.Fail(t, "invalid count of services for monitored resource")
	}

	if len((*monitoredResources)[1].Services[0].Metrics) != 1 {
		assert.Fail(t, "invalid count of metrics for service")
	}
}
