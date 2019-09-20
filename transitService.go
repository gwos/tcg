package main

import "C"
import (
	"./transit"
	"encoding/json"
)

var transitPackage transit.Transit

func main()  {
	
}

//export SendMetrics
func SendMetrics(timeSeriesJson *C.char) *C.char {

	return C.CString("")
}

//export SendResourcesWithMetrics
func SendResourcesWithMetrics(resourcesWithMetricsJson *C.char) *C.char {

	return C.CString("")
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

	return C.CString("")
}

//TODO:
func ListInventory() {

}
