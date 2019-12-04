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
	"os"
	"sync"
	"unsafe"
)

// instantiateServicesOnce guards initialization by instantiateServices
var instantiateServicesOnce sync.Once

var controller *services.Controller
var transitService *services.TransitService

// Encapsulating this initialization into a function, and calling that function when necessary,
// as opposed to making these assignments as global-variable initializaation assignments, gives
// the calling application a chance to alter the environment variables before they are frozen
// by the underlying os.Getenv() routine.
func instantiateServices() {
	controller = services.GetController()
	transitService = services.GetTransitService()
}

func Startup() bool {
        instantiateServicesOnce.Do(instantiateServices)
	var err error
	if transitService.AgentConfig.StartController {
		if err = transitService.StartController(); err != nil {
			log.Error(err.Error())
		}
	}
	if err == nil {
		if transitService.AgentConfig.StartNats {
			if err = transitService.StartNats(); err != nil {
				log.Error(err.Error())
			}
		}
	}
	// NOTE:  transitService.AgentConfig.StartTransport is already
	// called by transitService.StartNats, so we don't call it here
	log.Info("libtransit:", transitService.Status())

	if err != nil {
	    return false
	}
	return true
}

func main() {
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// GoSetenv() is for use by a calling application to alter environment variables in
// a manner that will be understood by the Go runtime.  We need it because the standard
// C-language putenv() and setenv() routines do not alter the Go environment as intended,
// due to issues with when os.Getenv() or related routines first get called.  To affect
// the setup for the services managed by libtransit, calls to GoSetenv() must be made
// *before* a call to Startup() or any of the other routines that might probe for or
// attempt to start, stop, or otherwise interact with one of the services.

//export GoSetenv
func GoSetenv(key, value *C.char) int {
    err := os.Setenv(C.GoString(key), C.GoString(value))
    if err != nil {
	// Logically, we would like to set errno here. corresponding
	// to what one would see with the POSIX setenv() routine.
	//     errno = ENOMEM
	//     errno = EINVAL
	// But until and unless we figure out how to do that, we just
	// skip that part of the setenv() emulation.
        return -1
    }
    return 0
}

func putError(errorBuf *C.char, err error) {
	buf := (*[int(C.ERROR_LEN)]byte)(unsafe.Pointer(errorBuf))
	buf[min(copy(buf[:], err.Error()), C.ERROR_LEN-1)] = 0
}

// SendResourcesWithMetrics is a C API for transitService.SendResourceWithMetrics
//export SendResourcesWithMetrics
func SendResourcesWithMetrics(resourcesWithMetricsRequestJSON, errorBuf *C.char) bool {
        instantiateServicesOnce.Do(instantiateServices)
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
        instantiateServicesOnce.Do(instantiateServices)
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
        instantiateServicesOnce.Do(instantiateServices)
	if err := transitService.StartController(); err != nil {
		putError(errorBuf, err)
		return false
	}
	return true
}

// StopController is a C API for transitService.StopController
//export StopController
func StopController(errorBuf *C.char) bool {
        instantiateServicesOnce.Do(instantiateServices)
	if err := transitService.StopController(); err != nil {
		putError(errorBuf, err)
		return false
	}
	return true
}

// StartNats is a C API for transitService.StartNats
//export StartNats
func StartNats(errorBuf *C.char) bool {
        instantiateServicesOnce.Do(instantiateServices)
	if err := transitService.StartNats(); err != nil {
		putError(errorBuf, err)
		return false
	}
	return true
}

// StopNats is a C API for transitService.StopNats
//export StopNats
func StopNats(errorBuf *C.char) bool {
        instantiateServicesOnce.Do(instantiateServices)
	if err := transitService.StopNats(); err != nil {
		putError(errorBuf, err)
		return false
	}
	return true
}

// StartTransport is a C API for transitService.StartTransport
//export StartTransport
func StartTransport(errorBuf *C.char) bool {
        instantiateServicesOnce.Do(instantiateServices)
	if err := transitService.StartTransport(); err != nil {
		putError(errorBuf, err)
		return false
	}
	return true
}

// StopTransport is a C API for transitService.StopTransport
//export StopTransport
func StopTransport(errorBuf *C.char) bool {
        instantiateServicesOnce.Do(instantiateServices)
	if err := transitService.StopTransport(); err != nil {
		putError(errorBuf, err)
		return false
	}
	return true
}

// IsControllerRunning is a C API for transitService.Status().Controller
//export IsControllerRunning
func IsControllerRunning() bool {
        instantiateServicesOnce.Do(instantiateServices)
	return transitService.Status().Controller == services.Running
}

// IsNatsRunning is a C API for transitService.Status().Nats
//export IsNatsRunning
func IsNatsRunning() bool {
        instantiateServicesOnce.Do(instantiateServices)
	return transitService.Status().Nats == services.Running
}

// IsTransportRunning is a C API for transitService.Status().Transport
//export IsTransportRunning
func IsTransportRunning() bool {
        instantiateServicesOnce.Do(instantiateServices)
	return transitService.Status().Transport == services.Running
}

// RegisterListMetricsHandler is a C API for controller.RegisterListMetricsHandler
//export RegisterListMetricsHandler
func RegisterListMetricsHandler(fn C.getTextHandlerType) {
        instantiateServicesOnce.Do(instantiateServices)
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
        instantiateServicesOnce.Do(instantiateServices)
	controller.RemoveListMetricsHandler()
}
