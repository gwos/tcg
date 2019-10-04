package main

// #define ERROR_LEN 250 /* for strncpy error message */
// #include <string.h> /* for strncpy error message */
import "C"
import (
	"encoding/json"
	"github.com/gwos/tng/transit"
	"log"
)

var transitPackage transit.Transit

func main() {
}

//export TestMonitoredResource
func TestMonitoredResource(str *C.char, errorMsg *C.char) *C.char {
	resource := transit.MonitoredResource{}
	if err := json.Unmarshal([]byte(C.GoString(str)), &resource); err != nil {
		C.strncpy((*C.char)(errorMsg), C.CString(err.Error()), C.ERROR_LEN)
		return nil
	}

	// resource.Labels = map[string]string{"key1": "value1", "key02": "value02"}
	resource.Status = transit.SERVICE_PENDING
	buf, _ := json.Marshal(&resource)

	log.Printf("#TestMonitoredResource: %v, %s", resource, buf)

	/* https://github.com/golang/go/wiki/cgo#go-strings-and-c-strings */
	return C.CString(string(buf))
}


//export SendResourcesWithMetrics
func SendResourcesWithMetrics(resourcesWithMetricsJson, errorMsg *C.char) *C.char {
	var resourceWithMetrics transit.TransitSendMetricsRequest

	err := json.Unmarshal([]byte(C.GoString(resourcesWithMetricsJson)), &resourceWithMetrics)
	if err != nil {
		C.strncpy((*C.char)(errorMsg), C.CString(err.Error()), C.ERROR_LEN)
		return nil
	}

	operationResults, err := transitPackage.SendResourcesWithMetrics(&resourceWithMetrics)
	if err != nil {
		C.strncpy((*C.char)(errorMsg), C.CString(err.Error()), C.ERROR_LEN)
		return nil
	}

	operationResultsJson, err := json.Marshal(operationResults)
	if err != nil {
		C.strncpy((*C.char)(errorMsg), C.CString(err.Error()), C.ERROR_LEN)
		return nil
	}

	return C.CString(string(operationResultsJson))
}

//export ListMetrics
func ListMetrics(errorMsg  *C.char) *C.char {
	monitorDescriptor, err := transitPackage.ListMetrics()
	if err != nil {
		C.strncpy((*C.char)(errorMsg), C.CString(err.Error()), C.ERROR_LEN)
		return nil
	}

	bytes, err := json.Marshal(monitorDescriptor)
	if err != nil {
		C.strncpy((*C.char)(errorMsg), C.CString(err.Error()), C.ERROR_LEN)
		return nil
	}

	return C.CString(string(bytes))
}

//export SynchronizeInventory
func SynchronizeInventory(inventoryJson, errorMsg *C.char) *C.char {
	var inventory transit.TransitSendInventoryRequest

	err := json.Unmarshal([]byte(C.GoString(inventoryJson)), &inventory)
	if err != nil {
		C.strncpy((*C.char)(errorMsg), C.CString(err.Error()), C.ERROR_LEN)
		return nil
	}

	operationResults, err := transitPackage.SynchronizeInventory(&inventory)
	if err != nil {
		C.strncpy((*C.char)(errorMsg), C.CString(err.Error()), C.ERROR_LEN)
		return nil
	}

	operationResultsJson, err := json.Marshal(operationResults)
	if err != nil {
		C.strncpy((*C.char)(errorMsg), C.CString(err.Error()), C.ERROR_LEN)
		return nil
	}

	return C.CString(string(operationResultsJson))
}

//TODO:
func ListInventory() {
}

//export Connect
func Connect(credentialsJson, errorMsg *C.char) *C.char {
	var credentials transit.Credentials

	err := json.Unmarshal([]byte(C.GoString(credentialsJson)), &credentials)
	if err != nil {
		C.strncpy((*C.char)(errorMsg), C.CString(err.Error()), C.ERROR_LEN)
		return nil
	}

	transitConfig, err := transit.Connect(credentials)
	if err != nil {
		C.strncpy((*C.char)(errorMsg), C.CString(err.Error()), C.ERROR_LEN)
		return nil
	}

	//start nats

	transitJson, err := json.Marshal(transitConfig)
	if err != nil {
		C.strncpy((*C.char)(errorMsg), C.CString(err.Error()), C.ERROR_LEN)
		return nil
	}

	return C.CString(string(transitJson))
}

//export Disconnect
func Disconnect(transitJson, errorMsg *C.char) bool {
	var transitConfig transit.Transit

	err := json.Unmarshal([]byte(C.GoString(transitJson)), &transitConfig)
	if err != nil {
		C.strncpy((*C.char)(errorMsg), C.CString(err.Error()), C.ERROR_LEN)
		return false
	}

	return transit.Disconnect(&transitConfig)
}
