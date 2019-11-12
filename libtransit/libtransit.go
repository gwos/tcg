package main

//#include <stdlib.h>
//#define ERROR_LEN 250 /* buffer for error message */
//
///* getTextHandlerType defines a function type that returns an allocated string.
// * It should be safe to call `C.free` on it. */
//typedef char *(*getTextHandlerType) ();
//
///* invokeGetTextHandler provides a function call by reference.
// * https://golang.org/cmd/cgo/#hdr-Go_references_to_C */
//static char *invokeGetTextHandler(getTextHandlerType fn) {
//	return fn();
//}
import "C"
import (
	"github.com/gwos/tng/services"
	"log"
	"unsafe"
)

var controller = services.GetController()
var transitService = services.GetTransitService()

func init() {
	if transitService.AgentConfig.StartController {
		if err := transitService.StartController(); err != nil {
			log.Println(err)
		}
	}
	if transitService.AgentConfig.StartNATS {
		if err := transitService.StartNATS(); err != nil {
			log.Println(err)
		}
	}
	// NOTE: the transitService.AgentConfig.StartTransport
	// processed by transitService.StartNATS itself
	log.Println("libtransit:", transitService.Status())
}

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
func SendResourcesWithMetrics(resourcesWithMetricsRequestJSON, errorBuf *C.char) bool {
	if err := transitService.
		SendResourceWithMetrics([]byte(C.GoString(resourcesWithMetricsRequestJSON))); err != nil {
		putError(errorBuf, err)
		return false
	}
	return true
}

//export SynchronizeInventory
func SynchronizeInventory(sendInventoryRequestJSON, errorBuf *C.char) bool {
	if err := transitService.
		SynchronizeInventory([]byte(C.GoString(sendInventoryRequestJSON))); err != nil {
		putError(errorBuf, err)
		return false
	}
	return true
}

//export StartController
func StartController(errorBuf *C.char) bool {
	if err := transitService.StartController(); err != nil {
		putError(errorBuf, err)
		return false
	}
	return true
}

//export StopController
func StopController(errorBuf *C.char) bool {
	if err := transitService.StopController(); err != nil {
		putError(errorBuf, err)
		return false
	}
	return true
}

//export StartNATS
func StartNATS(errorBuf *C.char) bool {
	if err := transitService.StartNATS(); err != nil {
		putError(errorBuf, err)
		return false
	}
	return true
}

//export StopNATS
func StopNATS() {
	transitService.StopNATS()
}

//export StartTransport
func StartTransport(errorBuf *C.char) bool {
	if err := transitService.StartTransport(); err != nil {
		putError(errorBuf, err)
		return false
	}
	return true
}

//export StopTransport
func StopTransport(errorBuf *C.char) bool {
	if err := transitService.StopTransport(); err != nil {
		putError(errorBuf, err)
		return false
	}
	return true
}

//export IsControllerRunning
func IsControllerRunning() bool {
	return transitService.Status().Controller == services.Running
}

//export IsNATSRunning
func IsNATSRunning() bool {
	return transitService.Status().NATS == services.Running
}

//export IsTransportRunning
func IsTransportRunning() bool {
	return transitService.Status().Transport == services.Running
}

//export RegisterListMetricsHandler
func RegisterListMetricsHandler(fn C.getTextHandlerType) {
	/* See notes on getTextHandlerType and invokeGetTextHandler */
	controller.RegisterListMetricsHandler(func() ([]byte, error) {
		textPtr := C.invokeGetTextHandler(fn)
		bytes := []byte(C.GoString(textPtr))
		C.free(unsafe.Pointer(textPtr))
		return bytes, nil
	})
}

//export RemoveListMetricsHandler
func RemoveListMetricsHandler() {
	controller.RemoveListMetricsHandler()
}
