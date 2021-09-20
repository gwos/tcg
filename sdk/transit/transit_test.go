package transit

import (
	"encoding/json"
	"reflect"
	"testing"
	"time"
)

func TestMonitoredServiceMarshalJSON(t *testing.T) {
	expected := `{"name":"monSvc","type":"service","status":"SERVICE_OK","lastCheckTime":"1609372800000","metrics":[]}`
	monSvc := MonitoredService{
		BaseTransitData: BaseTransitData{
			Name: "monSvc",
			Type: ResourceTypeService,
		},
		Status:        ServiceOk,
		LastCheckTime: &Timestamp{Time: time.Date(2020, time.December, 31, 0, 0, 0, 0, time.UTC)},
		Metrics:       []TimeSeries{},
	}
	output, err := json.Marshal(monSvc)
	if err != nil {
		t.Errorf("json.Marshal returned an error: %v", err)
	}
	if expected != string(output) {
		t.Errorf("json.Marshal returned %v want %v", string(output), expected)
	}
}

func TestMonitoredServiceUnmarshalJSON(t *testing.T) {
	input := []byte(`{"name":"monSvc","type":"service","status":"SERVICE_OK","lastCheckTime":"1609372800000","metrics":[]}`)
	expected := MonitoredService{
		BaseTransitData: BaseTransitData{
			Name: "monSvc",
			Type: ResourceTypeService,
		},
		Status:        ServiceOk,
		LastCheckTime: &Timestamp{Time: time.Date(2020, time.December, 31, 0, 0, 0, 0, time.UTC)},
		Metrics:       []TimeSeries{},
	}
	var value MonitoredService

	err := json.Unmarshal(input, &value)
	if err != nil {
		t.Errorf("json.Unmarshal returned an error: %v", err)
	}
	if !reflect.DeepEqual(value, expected) {
		t.Errorf("json.Unmarshal returned %v, want %v", value, expected)
	}
}
