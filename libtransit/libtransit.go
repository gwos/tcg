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
	"github.com/gwos/tng/log"
	"github.com/gwos/tng/services"
	"unsafe"
)

var controller = services.GetController()
var transitService = services.GetTransitService()

func init() {
	if transitService.AgentConfig.StartController {
		if err := transitService.StartController(); err != nil {
			log.Error(err.Error())
		}
	}
	if transitService.AgentConfig.StartNats {
		if err := transitService.StartNats(); err != nil {
			log.Error(err.Error())
		}
	}
	// NOTE: the transitService.AgentConfig.StartTransport
	// processed by transitService.StartNats itself
	log.Info("libtransit:", transitService.Status())
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

// SendResourcesWithMetrics is a C API for transitService.SendResourceWithMetrics
//export SendResourcesWithMetrics
func SendResourcesWithMetrics(resourcesWithMetricsRequestJSON, errorBuf *C.char) bool {
	if err := transitService.
		SendResourceWithMetrics([]byte(C.GoString(resourcesWithMetricsRequestJSON))); err != nil {
		putError(errorBuf, err)
		return false
	}
	return true
}

// SynchronizeInventory is a C API for transitService.SynchronizeInventory
//export SynchronizeInventory
func SynchronizeInventory(sendInventoryRequestJSON, errorBuf *C.char) bool {
	if err := transitService.
		SynchronizeInventory([]byte(C.GoString(sendInventoryRequestJSON))); err != nil {
		putError(errorBuf, err)
		return false
	}
	return true
}

// StartController is a C API for transitService.StartController
//export StartController
func StartController(errorBuf *C.char) bool {
	if err := transitService.StartController(); err != nil {
		putError(errorBuf, err)
		return false
	}
	return true
}

// StopController is a C API for transitService.StopController
//export StopController
func StopController(errorBuf *C.char) bool {
	if err := transitService.StopController(); err != nil {
		putError(errorBuf, err)
		return false
	}
	return true
}

// StartNats is a C API for transitService.StartNats
//export StartNats
func StartNats(errorBuf *C.char) bool {
	if err := transitService.StartNats(); err != nil {
		putError(errorBuf, err)
		return false
	}
	return true
}

// StopNats is a C API for transitService.StopNats
//export StopNats
func StopNats(errorBuf *C.char) bool {
	if err := transitService.StopNats(); err != nil {
		putError(errorBuf, err)
		return false
	}
	return true
}

// StartTransport is a C API for transitService.StartTransport
//export StartTransport
func StartTransport(errorBuf *C.char) bool {
	if err := transitService.StartTransport(); err != nil {
		putError(errorBuf, err)
		return false
	}
	return true
}

// StopTransport is a C API for transitService.StopTransport
//export StopTransport
func StopTransport(errorBuf *C.char) bool {
	if err := transitService.StopTransport(); err != nil {
		putError(errorBuf, err)
		return false
	}
	return true
}

// IsControllerRunning is a C API for transitService.Status().Controller
//export IsControllerRunning
func IsControllerRunning() bool {
	return transitService.Status().Controller == services.Running
}

// IsNatsRunning is a C API for transitService.Status().Nats
//export IsNatsRunning
func IsNatsRunning() bool {
	return transitService.Status().Nats == services.Running
}

// IsTransportRunning is a C API for transitService.Status().Transport
//export IsTransportRunning
func IsTransportRunning() bool {
	return transitService.Status().Transport == services.Running
}

// RegisterListMetricsHandler is a C API for controller.RegisterListMetricsHandler
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

// RemoveListMetricsHandler is a C API for controller.RemoveListMetricsHandler
//export RemoveListMetricsHandler
func RemoveListMetricsHandler() {
	controller.RemoveListMetricsHandler()
}
