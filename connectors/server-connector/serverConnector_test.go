package main

import (
	"reflect"
	"testing"

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
			"extensions": {
				"timer": 2
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
	assert.Equal(t, "metricsProfile-name", metricsProfile.Name)
	assert.Equal(t, 11, monitorConnection.ID)
	assert.Equal(t, 111, monitorConnection.ConnectorID)
}
