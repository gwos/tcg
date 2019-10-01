package main

//#include <string.h>
import "C"
import (
	"./transit"
	"encoding/json"
)

const (
	MAX_ERROR_MESSAGE_LENGTH int = 250
)

var transitPackage transit.Transit

func main() {
}

//export SendResourcesWithMetrics
func SendResourcesWithMetrics(resourcesWithMetricsJson, errorMsg *C.char) *C.char {
	var resourceWithMetrics []transit.ResourceWithMetrics

	err := json.Unmarshal([]byte(C.GoString(resourcesWithMetricsJson)), &resourceWithMetrics)
	if err != nil {
		C.strncpy((*C.char)(errorMsg), C.CString(err.Error()), C.size_t(MAX_ERROR_MESSAGE_LENGTH))
		return nil
	}

	err = transitPackage.SendResourcesWithMetrics(resourceWithMetrics)
	if err != nil {
		C.strncpy((*C.char)(errorMsg), C.CString(err.Error()), C.size_t(MAX_ERROR_MESSAGE_LENGTH))
		return nil
	}

	return C.CString("Success")
}

//export ListMetrics
func ListMetrics(errorMsg  *C.char) *C.char {
	monitorDescriptor, err := transitPackage.ListMetrics()
	if err != nil {
		C.strncpy((*C.char)(errorMsg), C.CString(err.Error()), C.size_t(MAX_ERROR_MESSAGE_LENGTH))
		return nil
	}

	bytes, err := json.Marshal(monitorDescriptor)
	if err != nil {
		C.strncpy((*C.char)(errorMsg), C.CString(err.Error()), C.size_t(MAX_ERROR_MESSAGE_LENGTH))
		return nil
	}

	return C.CString(string(bytes))
}

//export SynchronizeInventory
func SynchronizeInventory(monitoredResourcesJson, groupsJson, errorMsg *C.char) *C.char {
	var monitoredResources []transit.MonitoredResource
	var groups []transit.Group

	err := json.Unmarshal([]byte(C.GoString(monitoredResourcesJson)), &monitoredResources)
	if err != nil {
		C.strncpy((*C.char)(errorMsg), C.CString(err.Error()), C.size_t(MAX_ERROR_MESSAGE_LENGTH))
		return nil
	}
	err = json.Unmarshal([]byte(C.GoString(groupsJson)), &groups)
	if err != nil {
		C.strncpy((*C.char)(errorMsg), C.CString(err.Error()), C.size_t(MAX_ERROR_MESSAGE_LENGTH))
		return nil
	}

	transitSynchronize, err := transitPackage.SynchronizeInventory(&monitoredResources, &groups)
	if err != nil {
		C.strncpy((*C.char)(errorMsg), C.CString(err.Error()), C.size_t(MAX_ERROR_MESSAGE_LENGTH))
		return nil
	}

	transitSynchronizeJson, err := json.Marshal(transitSynchronize)
	if err != nil {
		C.strncpy((*C.char)(errorMsg), C.CString(err.Error()), C.size_t(MAX_ERROR_MESSAGE_LENGTH))
		return nil
	}

	return C.CString(string(transitSynchronizeJson))
}

//TODO:
func ListInventory() {
}

//export Connect
func Connect(credentialsJson, errorMsg *C.char) *C.char {
	var credentials transit.Credentials

	err := json.Unmarshal([]byte(C.GoString(credentialsJson)), &credentials)
	if err != nil {
		C.strncpy((*C.char)(errorMsg), C.CString(err.Error()), C.size_t(MAX_ERROR_MESSAGE_LENGTH))
		return nil
	}

	transitConfig, err := transit.Connect(credentials)
	if err != nil {
		C.strncpy((*C.char)(errorMsg), C.CString(err.Error()), C.size_t(MAX_ERROR_MESSAGE_LENGTH))
		return nil
	}

	//start nats

	transitJson, err := json.Marshal(transitConfig)
	if err != nil {
		C.strncpy((*C.char)(errorMsg), C.CString(err.Error()), C.size_t(MAX_ERROR_MESSAGE_LENGTH))
		return nil
	}

	return C.CString(string(transitJson))
}

//export Disconnect
func Disconnect(transitJson, errorMsg *C.char) bool {
	var transitConfig transit.Transit

	err := json.Unmarshal([]byte(C.GoString(transitJson)), &transitConfig)
	if err != nil {
		C.strncpy((*C.char)(errorMsg), C.CString(err.Error()), C.size_t(MAX_ERROR_MESSAGE_LENGTH))
		return false
	}

	return transit.Disconnect(&transitConfig)
}
