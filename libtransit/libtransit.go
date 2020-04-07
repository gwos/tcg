package main

//#include <stddef.h>
//#include <stdlib.h>
//#include <stdbool.h>
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
//
//typedef bool (*demandConfigHandler) ();
//
//static bool invokeDemandConfigHandler(demandConfigHandler fn) {
//	return fn();
//}
import "C"
import (
	"encoding/json"
	"fmt"
	"github.com/gwos/tng/services"
	"os"
	"unsafe"
)

func main() {
}

func min(args ...int) int {
	m := args[0]
	for _, arg := range args[1:] {
		if m > arg {
			m = arg
		}
	}
	return m
}

// bufStr puts Go string into C buffer
func bufStr(buf *C.char, bufLen C.size_t, str string) {
	NulTermLen := 1
	if bufLen > 0 {
		/* cast the buf as big enough then use with length respect */
		b := (*[4096]byte)(unsafe.Pointer(buf))
		m := min(4096-NulTermLen, int(bufLen)-NulTermLen, len(str))
		n := copy(b[:], str[:m])
		b[n] = 0 /* set nul termination */
	}
}

// GoSetenv is for use by a calling application to alter environment variables in
// a manner that will be understood by the Go runtime.  We need it because the standard
// C-language putenv() and setenv() routines do not alter the Go environment as intended,
// due to issues with when os.Getenv() or related routines first get called.  To affect
// the initial config for the services managed by libtransit, calls to GoSetenv() must be
// made *before* any call to one of the routines that might probe for or attempt to start,
// stop, or otherwise interact with one of the services.
//export GoSetenv
func GoSetenv(key, value, errBuf *C.char, errBufLen C.size_t) bool {
	err := os.Setenv(C.GoString(key), C.GoString(value))
	if err != nil {
		bufStr(errBuf, errBufLen, err.Error())
		return false
	}
	return true
}

// SendEvents is a C API for services.GetController().SendEvents
//export SendEvents
func SendEvents(payloadJSON, errBuf *C.char, errBufLen C.size_t) bool {
	if err := services.GetController().
		SendEvents([]byte(C.GoString(payloadJSON))); err != nil {
		bufStr(errBuf, errBufLen, err.Error())
		return false
	}
	return true
}

// SendEventsAck is a C API for services.GetController().SendEventsAck
//export SendEventsAck
func SendEventsAck(payloadJSON, errBuf *C.char, errBufLen C.size_t) bool {
	if err := services.GetController().
		SendEventsAck([]byte(C.GoString(payloadJSON))); err != nil {
		bufStr(errBuf, errBufLen, err.Error())
		return false
	}
	return true
}

// SendEventsUnack is a C API for services.GetController().SendEventsUnack
//export SendEventsUnack
func SendEventsUnack(payloadJSON, errBuf *C.char, errBufLen C.size_t) bool {
	if err := services.GetController().
		SendEventsUnack([]byte(C.GoString(payloadJSON))); err != nil {
		bufStr(errBuf, errBufLen, err.Error())
		return false
	}
	return true
}

// SendResourcesWithMetrics is a C API for services.GetTransitService().SendResourceWithMetrics
//export SendResourcesWithMetrics
func SendResourcesWithMetrics(payloadJSON, errBuf *C.char, errBufLen C.size_t) bool {
	if err := services.GetTransitService().
		SendResourceWithMetrics([]byte(C.GoString(payloadJSON))); err != nil {
		bufStr(errBuf, errBufLen, err.Error())
		return false
	}
	return true
}

// SynchronizeInventory is a C API for services.GetTransitService().SynchronizeInventory
//export SynchronizeInventory
func SynchronizeInventory(payloadJSON, errBuf *C.char, errBufLen C.size_t) bool {
	if err := services.GetTransitService().
		SynchronizeInventory([]byte(C.GoString(payloadJSON))); err != nil {
		bufStr(errBuf, errBufLen, err.Error())
		return false
	}
	return true
}

// StartController is a C API for services.GetTransitService().StartController
//export StartController
func StartController(errBuf *C.char, errBufLen C.size_t) bool {
	if err := services.GetTransitService().StartController(); err != nil {
		bufStr(errBuf, errBufLen, err.Error())
		return false
	}
	return true
}

// StopController is a C API for services.GetTransitService().StopController
//export StopController
func StopController(errBuf *C.char, errBufLen C.size_t) bool {
	if err := services.GetTransitService().StopController(); err != nil {
		bufStr(errBuf, errBufLen, err.Error())
		return false
	}
	return true
}

// StartNats is a C API for services.GetTransitService().StartNats
//export StartNats
func StartNats(errBuf *C.char, errBufLen C.size_t) bool {
	if err := services.GetTransitService().StartNats(); err != nil {
		bufStr(errBuf, errBufLen, err.Error())
		return false
	}
	return true
}

// StopNats is a C API for services.GetTransitService().StopNats
//export StopNats
func StopNats(errBuf *C.char, errBufLen C.size_t) bool {
	if err := services.GetTransitService().StopNats(); err != nil {
		bufStr(errBuf, errBufLen, err.Error())
		return false
	}
	return true
}

// StartTransport is a C API for services.GetTransitService().StartTransport
//export StartTransport
func StartTransport(errBuf *C.char, errBufLen C.size_t) bool {
	if err := services.GetTransitService().StartTransport(); err != nil {
		bufStr(errBuf, errBufLen, err.Error())
		return false
	}
	return true
}

// StopTransport is a C API for services.GetTransitService().StopTransport
//export StopTransport
func StopTransport(errBuf *C.char, errBufLen C.size_t) bool {
	if err := services.GetTransitService().StopTransport(); err != nil {
		bufStr(errBuf, errBufLen, err.Error())
		return false
	}
	return true
}

// DemandConfig is a C API for services.GetTransitService().DemandConfig
//export DemandConfig
func DemandConfig(errBuf *C.char, errBufLen C.size_t) bool {
	if err := services.GetTransitService().DemandConfig(); err != nil {
		bufStr(errBuf, errBufLen, err.Error())
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
		res := []byte(C.GoString(textPtr))
		C.free(unsafe.Pointer(textPtr))
		return res, nil
	})
}

// RemoveListMetricsHandler is a C API for services.GetController().RemoveListMetricsHandler
//export RemoveListMetricsHandler
func RemoveListMetricsHandler() {
	services.GetController().RemoveListMetricsHandler()
}

// RegisterDemandConfigHandler is a C API for services.GetTransitService().RegisterDemandConfigHandler
//export RegisterDemandConfigHandler
func RegisterDemandConfigHandler(fn C.demandConfigHandler) {
	services.GetTransitService().RegisterDemandConfigHandler(func() bool{
		return bool(C.invokeDemandConfigHandler(fn))
	})
}

// RemoveDemandConfigHandler is a C API for services.GetTransitService().RemoveDemandConfigHandler()
//export RemoveDemandConfigHandler
func RemoveDemandConfigHandler() {
	services.GetTransitService().RemoveDemandConfigHandler()
}

// GetConnectorConfig is a C API for getting services.GetTransitService().Connector
//export GetConnectorConfig
func GetConnectorConfig(buf *C.char, bufLen C.size_t, errBuf *C.char, errBufLen C.size_t) bool {
	res, err := json.Marshal(services.GetTransitService().Connector)
	if err != nil {
		bufStr(errBuf, errBufLen, err.Error())
		return false
	}
	cStrLen := len(res) + 1
	if cStrLen > int(bufLen) {
		errMsg := fmt.Sprintf("Buffer too small, need at least %d bytes", cStrLen)
		bufStr(errBuf, errBufLen, errMsg)
		return false
	}
	bufStr(buf, bufLen, string(res))
	return true
}
