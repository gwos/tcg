package main

import (
	"reflect"
	"testing"
	"time"

	"github.com/gwos/tcg/connectors"
	"github.com/gwos/tcg/transit"
	"github.com/stretchr/testify/assert"
)

func TestUnmarshalConfig(t *testing.T) {
	expected := ExtConfig{
		Timer: connectors.DefaultTimer,
	}

	data := []byte(`
	{
		"agentId": "99998888-7777-6666-a3b0-b14622f7dd39",
		"metricsProfile": {"name": "metricsProfile-name"},
		"monitorConnection": {
			"id": 11,
			"connectorId": 111,
			"extensions": {}
		}
	}`)
	data2 := []byte(`
	{
		"agentId": "99998888-7777-6666-a3b0-b14622f7dd39",
		"metricsProfile": {"name": "metricsProfile-name"},
		"monitorConnection": {
			"id": 11,
			"connectorId": 111,
			"extensions": {
				"checkIntervalMinutes": 3
			}
		}
	}`)
	extConfig := &ExtConfig{Timer: connectors.DefaultTimer}
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

	expected.Timer = 3 * time.Minute
	assert.NoError(t, connectors.UnmarshalConfig(data2, metricsProfile, monitorConnection))
	if !reflect.DeepEqual(*extConfig, expected) {
		t.Errorf("ExtConfig actual:\n%v\nexpected:\n%v", *extConfig, expected)
	}
}
