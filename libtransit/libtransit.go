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
	"os"
	"unsafe"
)

func main() {
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// GoSetenv is for use by a calling application to alter environment variables in
// a manner that will be understood by the Go runtime.  We need it because the standard
// C-language putenv() and setenv() routines do not alter the Go environment as intended,
// due to issues with when os.Getenv() or related routines first get called.  To affect
// the initial setup for the services managed by libtransit, calls to GoSetenv() must be
// made *before* any call to one of the routines that might probe for or attempt to start,
// stop, or otherwise interact with one of the services.
//export GoSetenv
func GoSetenv(key, value, errorBuf *C.char) bool {
	err := os.Setenv(C.GoString(key), C.GoString(value))
	if err != nil {
		putError(errorBuf, err)
		return false
	}
	return true
}

func putError(errorBuf *C.char, err error) {
	buf := (*[int(C.ERROR_LEN)]byte)(unsafe.Pointer(errorBuf))
	buf[min(copy(buf[:], err.Error()), C.ERROR_LEN-1)] = 0
}

// SendResourcesWithMetrics is a C API for services.GetTransitService().SendResourceWithMetrics
//export SendResourcesWithMetrics
func SendResourcesWithMetrics(resourcesWithMetricsRequestJSON, errorBuf *C.char) bool {
	if err := services.GetTransitService().
		SendResourceWithMetrics([]byte(C.GoString(resourcesWithMetricsRequestJSON))); err != nil {
		putError(errorBuf, err)
		return false
	}
	return true
}

// SynchronizeInventory is a C API for services.GetTransitService().SynchronizeInventory
//export SynchronizeInventory
func SynchronizeInventory(sendInventoryRequestJSON, errorBuf *C.char) bool {
	if err := services.GetTransitService().
		SynchronizeInventory([]byte(C.GoString(sendInventoryRequestJSON))); err != nil {
		putError(errorBuf, err)
		return false
	}
	return true
}

// StartController is a C API for services.GetTransitService().StartController
//export StartController
func StartController(errorBuf *C.char) bool {
	if err := services.GetTransitService().StartController(); err != nil {
		putError(errorBuf, err)
		return false
	}
	return true
}

// StopController is a C API for services.GetTransitService().StopController
//export StopController
func StopController(errorBuf *C.char) bool {
	if err := services.GetTransitService().StopController(); err != nil {
		putError(errorBuf, err)
		return false
	}
	return true
}

// StartNats is a C API for services.GetTransitService().StartNats
//export StartNats
func StartNats(errorBuf *C.char) bool {
	if err := services.GetTransitService().StartNats(); err != nil {
		putError(errorBuf, err)
		return false
	}
	return true
}

// StopNats is a C API for services.GetTransitService().StopNats
//export StopNats
func StopNats(errorBuf *C.char) bool {
	if err := services.GetTransitService().StopNats(); err != nil {
		putError(errorBuf, err)
		return false
	}
	return true
}

// StartTransport is a C API for services.GetTransitService().StartTransport
//export StartTransport
func StartTransport(errorBuf *C.char) bool {
	if err := services.GetTransitService().StartTransport(); err != nil {
		putError(errorBuf, err)
		return false
	}
	return true
}

// StopTransport is a C API for services.GetTransitService().StopTransport
//export StopTransport
func StopTransport(errorBuf *C.char) bool {
	if err := services.GetTransitService().StopTransport(); err != nil {
		putError(errorBuf, err)
		return false
	}
	return true
}

// IsControllerRunning is a C API for services.GetTransitService().Status().Controller
//export IsControllerRunning
func IsControllerRunning() bool {
	return services.GetTransitService().Status().Controller == services.Running
}

// IsNatsRunning is a C API for services.GetTransitService().Status().Nats
//export IsNatsRunning
func IsNatsRunning() bool {
	return services.GetTransitService().Status().Nats == services.Running
}

// IsTransportRunning is a C API for services.GetTransitService().Status().Transport
//export IsTransportRunning
func IsTransportRunning() bool {
	return services.GetTransitService().Status().Transport == services.Running
}

// RegisterListMetricsHandler is a C API for services.GetController().RegisterListMetricsHandler
//export RegisterListMetricsHandler
func RegisterListMetricsHandler(fn C.getTextHandlerType) {
	/* See notes on getTextHandlerType and invokeGetTextHandler */
	services.GetController().RegisterListMetricsHandler(func() ([]byte, error) {
		textPtr := C.invokeGetTextHandler(fn)
		bytes := []byte(C.GoString(textPtr))
		C.free(unsafe.Pointer(textPtr))
		return bytes, nil
	})
}

// RemoveListMetricsHandler is a C API for services.GetController().RemoveListMetricsHandler
//export RemoveListMetricsHandler
func RemoveListMetricsHandler() {
	services.GetController().RemoveListMetricsHandler()
}
