package main

//#define ERROR_LEN 250 /* for strncpy error message */
//#include <string.h> /* for strncpy error message */
import "C"
import (
	"github.com/gwos/tng/transit"
)

var transitService transit.Service

func main() {
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

////export ListMetrics
//func ListMetrics(errorMsg *C.char) *C.char {
//	monitorDescriptor, err := transitService.ListMetrics()
//	if err != nil {
//		C.strncpy((*C.char)(errorMsg), C.CString(err.Error()), C.ERROR_LEN)
//		return nil
//	}
//
//	bytes, err := json.Marshal(monitorDescriptor)
//	if err != nil {
//		C.strncpy((*C.char)(errorMsg), C.CString(err.Error()), C.ERROR_LEN)
//		return nil
//	}
//
//	return C.CString(string(bytes))
//}

//export SynchronizeInventory
func SynchronizeInventory(inventoryJson, errorMsg *C.char) bool {
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
