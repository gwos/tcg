package transit

import (
	"bytes"
	"encoding/json"
	"reflect"
	"testing"
	"time"
)

func TestMonitoredService_MarshalJSON(t *testing.T) {
	expected := `{"name":"monSvc","type":"service","status":"SERVICE_OK","lastCheckTime":"1609372800000","metrics":[]}`
	monSvc := MonitoredService{
		BaseInfo: BaseInfo{
			Name: "monSvc",
			Type: ResourceTypeService,
		},
		MonitoredInfo: MonitoredInfo{
			Status:        ServiceOk,
			LastCheckTime: &Timestamp{Time: time.Date(2020, time.December, 31, 0, 0, 0, 0, time.UTC)},
		},
		Metrics: []TimeSeries{},
	}
	output, err := json.Marshal(monSvc)
	if err != nil {
		t.Errorf("json.Marshal returned an error: %v", err)
	}
	if expected != string(output) {
		t.Errorf("json.Marshal returned %v want %v", string(output), expected)
	}
}

func TestMonitoredService_UnmarshalJSON(t *testing.T) {
	input := []byte(`{"name":"monSvc","type":"service","status":"SERVICE_OK","lastCheckTime":"1609372800000","metrics":[]}`)
	expected := MonitoredService{
		BaseInfo: BaseInfo{
			Name: "monSvc",
			Type: ResourceTypeService,
		},
		MonitoredInfo: MonitoredInfo{
			Status:        ServiceOk,
			LastCheckTime: &Timestamp{Time: time.Date(2020, time.December, 31, 0, 0, 0, 0, time.UTC)},
		},
		Metrics: []TimeSeries{},
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

func TestMonitoredService_CreateProperties(t *testing.T) {
	svc := new(MonitoredService)
	svc.CreateProperties(map[string]interface{}{
		"prop-0": true,
		"prop-1": false,
		"prop-2": 0.0,
		"prop-3": 1.1,
		"prop-4": 0,
		"prop-5": -1,
		"prop-6": "foo-bar",
		"prop-7": Timestamp{Time: time.Date(2020, time.December, 31, 0, 0, 0, 0, time.UTC)},
		"prop-8": &Timestamp{Time: time.Date(2022, time.December, 31, 0, 0, 0, 0, time.UTC)},
		"prop-9": TypedValue{ValueType: TimeType, TimeValue: &Timestamp{Time: time.Date(2010, time.December, 31, 0, 0, 0, 0, time.UTC)}},
	})

	expectedJSON := []byte(`{
		"name": "",
		"type": "",
		"properties": {
		  "prop-0": {
			"valueType": "BooleanType",
			"boolValue": true
		  },
		  "prop-1": {
			"valueType": "BooleanType",
			"boolValue": false
		  },
		  "prop-2": {
			"valueType": "DoubleType",
			"doubleValue": 0
		  },
		  "prop-3": {
			"valueType": "DoubleType",
			"doubleValue": 1.1
		  },
		  "prop-4": {
			"valueType": "IntegerType",
			"integerValue": 0
		  },
		  "prop-5": {
			"valueType": "IntegerType",
			"integerValue": -1
		  },
		  "prop-6": {
			"valueType": "StringType",
			"stringValue": "foo-bar"
		  },
		  "prop-7": {
			"valueType": "TimeType",
			"timeValue": "1609372800000"
		  },
		  "prop-8": {
			"valueType": "TimeType",
			"timeValue": "1672444800000"
		  },
		  "prop-9": {
			"valueType": "TimeType",
			"timeValue": "1293753600000"
		  }
		},
		"status": "",
		"metrics": null
	}`)

	/* check json equality */
	var expected, value interface{}
	valueJSON, err := json.Marshal(svc)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	err = json.Unmarshal(expectedJSON, &expected)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	err = json.Unmarshal(valueJSON, &value)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if !reflect.DeepEqual(value, expected) {
		var b bytes.Buffer
		if err := json.Compact(&b, expectedJSON); err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
		t.Errorf("json is not equal\n returned:\n%v\n want:\n%v", string(valueJSON), b.String())
	}
}
