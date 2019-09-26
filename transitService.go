package main

import "C"
import (
	"github.com/gwos/tng/transit"
	"encoding/json"
	"log"
)

var transitPackage transit.Transit

func main() {

}

//export SendMetrics
func SendMetrics(timeSeriesJson *C.char) *C.char {

	return C.CString("")
}

//export SendResourcesWithMetrics
func SendResourcesWithMetrics(resourcesWithMetricsJson *C.char) *C.char {
	var resourceWithMetrics []transit.ResourceWithMetrics

	err := json.Unmarshal([]byte(C.GoString(resourcesWithMetricsJson)), &resourceWithMetrics)
	if err != nil {
		panic(err)
	}

	err = transitPackage.SendResourcesWithMetrics(resourceWithMetrics)
	if err != nil {
		panic(err)
	}

	return C.CString("Sent")
}

//export ListMetrics
func ListMetrics() *C.char {
	monitorDescriptor, err := transitPackage.ListMetrics()
	if err != nil {
		panic(err)
	}

	bytes, err := json.Marshal(monitorDescriptor)
	if err != nil {
		panic(err)
	}

	return C.CString(string(bytes))
}

//export SynchronizeInventory
func SynchronizeInventory(monitoredResourcesJson, groupsJson *C.char) *C.char {
	var monitoredResources []transit.MonitoredResource
	var groups []transit.Group

	err := json.Unmarshal([]byte(C.GoString(monitoredResourcesJson)), &monitoredResources)
	if err != nil {
		log.Fatal(err)
	}
	err = json.Unmarshal([]byte(C.GoString(groupsJson)), &groups)
	if err != nil {
		log.Fatal(err)
	}

	transitSynchronize, err := transitPackage.SynchronizeInventory(&monitoredResources, &groups)
	if err != nil {
		log.Fatal(err)
	}

	transitSynchronizeJson, err := json.Marshal(transitSynchronize)

	return C.CString(string(transitSynchronizeJson))
}

//TODO:
func ListInventory() {
}

//export Connect
func Connect(credentialsJson *C.char) *C.char{
	var credentials transit.Credentials

	err := json.Unmarshal([]byte(C.GoString(credentialsJson)), &credentials)
	if err != nil {
		log.Fatal(err)
	}

	transitConfig, err := transit.Connect(credentials)
	if err != nil {
		log.Fatal(err)
	}

	transitJson, err := json.Marshal(transitConfig)
	if err != nil {
		log.Fatal(err)
	}

	return C.CString(string(transitJson))
}

//export Disconnect
func Disconnect(transitJson *C.char) bool{
	var transitConfig transit.Transit

	err := json.Unmarshal([]byte(C.GoString(transitJson)), &transitConfig)
	if err != nil {
		log.Fatal(err)
		return false
	}

	return transit.Disconnect(&transitConfig)
}
