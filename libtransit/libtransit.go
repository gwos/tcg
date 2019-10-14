package main

//#define ERROR_LEN 250 /* buffer for error message */
import "C"
import (
	"github.com/gwos/tng/services"
	"unsafe"
)

var transitService services.Service

func main() {
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func putError(errorBuf *C.char, err error) {
	buf := (*[int(C.ERROR_LEN)]byte)(unsafe.Pointer(errorBuf))
	buf[min(copy(buf[:], err.Error()), C.ERROR_LEN-1)] = 0
}

//export SendResourcesWithMetrics
func SendResourcesWithMetrics(resourcesWithMetricsRequestJson, errorBuf *C.char) bool {
	err := transitService.SendResourceWithMetrics([]byte(C.GoString(resourcesWithMetricsRequestJson)))
	if err != nil {
		putError(errorBuf, err)
		return false
	}
	return true
}

////export ListMetrics
// func ListMetrics(errorBuf *C.char) *C.char {
// 	monitorDescriptor, err := transitService.ListMetrics()
// 	if err != nil {
// 		putError(errorBuf, err)
// 		return nil
// 	}
//
// 	bytes, err := json.Marshal(monitorDescriptor)
// 	if err != nil {
// 		putError(errorBuf, err)
// 		return nil
// 	}
//
// 	return C.CString(string(bytes))
// }

//export SynchronizeInventory
func SynchronizeInventory(sendInventoryRequestJson, errorBuf *C.char) bool {
	err := transitService.SynchronizeInventory([]byte(C.GoString(sendInventoryRequestJson)))
	if err != nil {
		putError(errorBuf, err)
		return false
	}
	return true
}

//TODO:
func ListInventory() {
}
