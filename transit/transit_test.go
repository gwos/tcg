package transit

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/gwos/tcg/milliseconds"
	"github.com/stretchr/testify/assert"
)

func TestStructMarshalJSON(t *testing.T) {
	expected := `{"name":"monSvc","type":"Service","status":"SERVICE_OK","lastCheckTime":"1609372800000","metrics":[]}`
	monSvc := MonitoredService{
		Name:          "monSvc",
		Type:          Service,
		Status:        ServiceOk,
		LastCheckTime: &milliseconds.MillisecondTimestamp{Time: time.Date(2020, time.December, 31, 0, 0, 0, 0, time.UTC)},
		Metrics:       []TimeSeries{},
	}
	output, err := json.Marshal(monSvc)

	assert.NoError(t, err)
	assert.Equal(t, expected, string(output))
}

func TestStructUnmarshalJSON(t *testing.T) {
	input := []byte(`{"name":"monSvc","type":"Service","status":"SERVICE_OK","lastCheckTime":"1609372800000","metrics":[]}`)
	expected := MonitoredService{
		Name:          "monSvc",
		Type:          Service,
		Status:        ServiceOk,
		LastCheckTime: &milliseconds.MillisecondTimestamp{Time: time.Date(2020, time.December, 31, 0, 0, 0, 0, time.UTC)},
		Metrics:       []TimeSeries{},
	}
	var monSvc MonitoredService

	err := json.Unmarshal(input, &monSvc)
	assert.NoError(t, err)
	assert.Equal(t, expected, monSvc)
}
