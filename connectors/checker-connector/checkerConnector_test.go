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
				Command: []string{"kahuna", "-stdout"},
				Cron:    "1 * * * * *",
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
		"metricsProfile": {"name": "metricsProfile-name"},
		"monitorConnection": {
			"id": 11,
			"connectorId": 111,
			"extensions": {
				"schedule": [
					{
						"command": ["kahuna", "-stdout"],
						"cron": "1 * * * * *"
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
	monitorConnection := &transit.MonitorConnection{
		Extensions: extConfig,
	}

	assert.NoError(t, connectors.UnmarshalConfig(data, metricsProfile, monitorConnection))
	if !reflect.DeepEqual(*extConfig, expected) {
		t.Errorf("ExtConfig actual:\n%v\nexpected:\n%v", *extConfig, expected)
	}
	assert.Equal(t, "metricsProfile-name", metricsProfile.Name)
	assert.Equal(t, 11, monitorConnection.ID)
	assert.Equal(t, 111, monitorConnection.ConnectorID)
}

func TestExtConfigValidateFails(t *testing.T) {
	data := []byte(`
	{
		"monitorConnection": {
			"extensions": {
				"schedule": [
					{
						"command": ["kahuna", "-stdout"],
						"cron": "1 * * * * *"
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
	monitorConnection := &transit.MonitorConnection{
		Extensions: extConfig,
	}

	assert.NoError(t, connectors.UnmarshalConfig(data, metricsProfile, monitorConnection))
	actualErr := extConfig.Validate()
	expectedErr := fmt.Errorf("ExtConfig Schedule item error: Command is empty")
	if assert.Error(t, actualErr) {
		assert.Equal(t, expectedErr, actualErr)
	}
}
