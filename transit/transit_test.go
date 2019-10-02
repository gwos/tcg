package transit

import (
	"encoding/json"
	"log"
	"testing"
)

func TestMarshalMonitoredResource(t *testing.T) {
	resource := MonitoredResource{}
	buf, _ := json.Marshal(&resource)
	log.Println(resource, (string)(buf))

	typedValue := TypedValue{ValueType: StringType, StringValue: "some string"}
	props := map[string]TypedValue{"key01": typedValue}
	resource.Properties = props

	props["key02"] = TypedValue{ValueType: BooleanType, BoolValue: true}
	resource.Status = SERVICE_PENDING

	buf, _ = json.Marshal(&resource)
	log.Println(resource, (string)(buf))
	expected := `{"name":"","type":"","status":4,"lastCheckTime":"0001-01-01T00:00:00Z","nextCheckTime":"0001-01-01T00:00:00Z","properties":{"key01":{"valueType":3,"stringValue":"some string","dateValue":"0001-01-01T00:00:00Z"},"key02":{"valueType":4,"boolValue":true,"dateValue":"0001-01-01T00:00:00Z"}}}`

	if expected != (string)(buf) {
		t.Error("resource.Status")
	}
}
