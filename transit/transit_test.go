package transit

import (
	"encoding/json"
	"fmt"
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
	expected := `{"name":"","type":"","status":4,"lastCheckTime":"0001-01-01T00:00:00Z","nextCheckTime":"0001-01-01T00:00:00Z","properties":{"key01":{"valueType":3,"stringValue":"some string","timeValue":"0001-01-01T00:00:00Z"},"key02":{"valueType":4,"boolValue":true,"timeValue":"0001-01-01T00:00:00Z"}}}`

	if expected != (string)(buf) {
		t.Error("resource.Status")
	}
}

func TestSendResourcesWithMetrics(t *testing.T) {
	var transit Transit

	resourcesWithMetricsJson := "[{\"resource\":{\"name\":\"mc-test-host\",\"type\":\"HOST\",\"status\":2}},{\"resource\":{\"name\":\"mc-test-service-0\",\"type\":\"SERVICE\",\"status\":2,\"owner\":\"mc-test-host\"},\"metrics\":[{\"tags\":{},\"metricName\":\"mc-test-service-0\",\"sampleType\":2,\"interval\":{\"startTime\":5961284951238,\"endTime\":5961284951238},\"value\":{\"valueType\":3,\"stringValue\":\"5.5\"}},{\"tags\":{},\"metricName\":\"mc-test-service-0\",\"sampleType\":2,\"interval\":{\"startTime\":5961284951238,\"endTime\":5961284951238},\"value\":{\"valueType\":3,\"stringValue\":\"7\"}},{\"tags\":{},\"metricName\":\"mc-test-service-0\",\"sampleType\":3,\"interval\":{\"startTime\":5961284951238,\"endTime\":5961284951238},\"value\":{\"valueType\":3,\"stringValue\":\"10\"}}]},{\"resource\":{\"name\":\"mc-test-service-1\",\"type\":\"SERVICE\",\"status\":2,\"owner\":\"mc-test-host\"},\"metrics\":[{\"tags\":{},\"metricName\":\"mc-test-service-0-A\",\"sampleType\":1,\"interval\":{\"startTime\":5961284951238,\"endTime\":5961284951238},\"value\":{\"valueType\":2,\"doubleValue\":5.5}},{\"tags\":{},\"metricName\":\"mc-test-service-0-A\",\"sampleType\":3,\"interval\":{\"startTime\":5961284951238,\"endTime\":5961284951238},\"value\":{\"valueType\":1,\"integerValue\":7}},{\"tags\":{},\"metricName\":\"mc-test-service-0-A\",\"sampleType\":1,\"interval\":{\"startTime\":5961284951238,\"endTime\":5961284951238},\"value\":{\"valueType\":1,\"integerValue\":10}},{\"tags\":{},\"metricName\":\"mc-test-service-0-A\",\"sampleType\":2,\"interval\":{\"startTime\":5961284951238,\"endTime\":5961284951238},\"value\":{\"valueType\":1,\"integerValue\":0}},{\"tags\":{},\"metricName\":\"mc-test-service-0-A\",\"sampleType\":4,\"interval\":{\"startTime\":5961284951238,\"endTime\":5961284951238},\"value\":{\"valueType\":1,\"integerValue\":15}},{\"tags\":{\"cpu\":\"cpu0\"},\"metricName\":\"mc-test-service-0-B\",\"sampleType\":1,\"interval\":{\"startTime\":5961284951238,\"endTime\":5961284951238},\"value\":{\"valueType\":2,\"doubleValue\":1.0}},{\"tags\":{\"cpu\":\"cpu1\"},\"metricName\":\"mc-test-service-0-B\",\"sampleType\":1,\"interval\":{\"startTime\":5961284951238,\"endTime\":5961284951238},\"value\":{\"valueType\":2,\"doubleValue\":1.1}},{\"tags\":{\"cpu\":\"cpu2\"},\"metricName\":\"mc-test-service-0-B\",\"sampleType\":1,\"interval\":{\"startTime\":5961284951238,\"endTime\":5961284951238},\"value\":{\"valueType\":2,\"doubleValue\":0.9}},{\"tags\":{\"x\":\"x0\"},\"metricName\":\"mc-test-service-0-C\",\"sampleType\":2,\"interval\":{\"startTime\":5961284951238,\"endTime\":5961284951238},\"value\":{\"valueType\":3,\"stringValue\":\"10\"}},{\"tags\":{\"x\":\"x1\"},\"metricName\":\"mc-test-service-0-C\",\"sampleType\":3,\"interval\":{\"startTime\":5961284951238,\"endTime\":5961284951238},\"value\":{\"valueType\":3,\"stringValue\":\"12\"}}]}]"

	var resourcesWithMetrics []ResourceWithMetrics

	err := json.Unmarshal([]byte(resourcesWithMetricsJson), &resourcesWithMetrics)
	if err != nil {
		t.Error(err)
	}

	operationResults, err := transit.SendResourcesWithMetrics(&resourcesWithMetrics)
	if err != nil {
		t.Error(err)
	}

	fmt.Println(operationResults.ResourcesAdded)
	fmt.Println(operationResults.ResourcesDeleted)
}

func TestSynchronizeInventory(t *testing.T) {
	var transit Transit

	inventoryJson := "{\"context\":{\"appType\":\"test-app\",\"agentID\":\"test-agent\",\"traceToken\":\"test-token\",\"timeStamp\":1570030732928},\"resources\":[{\"properties\":{},\"name\":\"test-name1\",\"type\":\"HOST\",\"status\":1},{\"properties\":{},\"name\":\"test-name2\",\"type\":\"SERVICE\",\"status\":1,\"owner\":\"test-name1\"}],\"groups\":[{\"groupName\":\"test-groupName1\",\"resources\":[{\"properties\":{},\"name\":\"test-name1\",\"type\":\"HOST\",\"status\":1},{\"properties\":{},\"name\":\"test-name2\",\"type\":\"SERVICE\",\"status\":1,\"owner\":\"test-name1\"}]},{\"groupName\": \"test-groupName2\",\"resources\": [{\"properties\": {},\"name\": \"test-name1\",\"type\": \"HOST\",\"status\": 1},{\"properties\": {},\"name\": \"test-name2\",\"type\": \"SERVICE\",\"status\": 1,\"owner\": \"test-name1\"}]}]}"

	var inventory transitSendInventoryRequest

	err := json.Unmarshal([]byte(inventoryJson), &inventory)
	if err != nil {
		t.Error(err)
	}

	operationResults, err := transit.SynchronizeInventory(&inventory)
	if err != nil {
		t.Error(err)
	}

	fmt.Println(operationResults.ResourcesAdded)
	fmt.Println(operationResults.ResourcesDeleted)
}
