package main

// #define ERROR_LEN 250 /* for strncpy error message */
//include <string.h> /* for strncpy error message */
import "C"
import (
	"encoding/json"
	"fmt"
	"github.com/gwos/tng/transit"
	"log"
)

var transitService transit.Service

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
func SendResourcesWithMetrics(resourcesWithMetricsJson, errorMsg *C.char) bool {
	err := transitService.SendResourceWithMetrics([]byte(C.GoString(resourcesWithMetricsJson)))
	if err != nil {
		C.strncpy((*C.char)(errorMsg), C.CString(err.Error()), C.ERROR_LEN)
		return false
	}

	return true
}

//export ListMetrics
func ListMetrics(errorMsg *C.char) *C.char {
	monitorDescriptor, err := transitService.ListMetrics()
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
func SynchronizeInventory(inventoryJson, errorMsg *C.char) bool {
	fmt.Println(C.GoString(inventoryJson))
	err := transitService.SynchronizeInventory([]byte(C.GoString(inventoryJson)))
	if err != nil {
		C.strncpy((*C.char)(errorMsg), C.CString(err.Error()), C.ERROR_LEN)
		return false
	}

	return true
}

//TODO:
func ListInventory() {
}
