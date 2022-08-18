package main

/*
#include <stdbool.h>
#include <stddef.h>
#include <stdint.h>
#include <stdlib.h>

// textGetterType defines a function type that returns an allocated string.
// It should be safe to call `C.free` on it.
typedef char *(*textGetterType) ();

// textSetterType defines a function type that accepts a string.
typedef void (*textSetterType) (char *);

// invokeTextGetter provides a function call by reference.
// https://golang.org/cmd/cgo/#hdr-Go_references_to_C
static char *invokeTextGetter(textGetterType fn) {
	return fn();
}

// invokeTextSetter provides a function call by reference.
static void invokeTextSetter(textSetterType fn, char *p) {
	return fn(p);
}
*/
import "C"
import (
	"fmt"
	"os"
	"unsafe"

	"github.com/gwos/tcg/config"
	"github.com/gwos/tcg/services"
)

func init() {
	config.AllowFlags = false
	services.AllowSignalHandlers = false
}

func main() {
}

// min returns minimum value
func min(x int, rest ...int) int {
	m := x
	for _, y := range rest[:] {
		if m > y {
			m = y
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

// msgfBufTooSmall formats a message
func msgfBufTooSmall(n int) string {
	return fmt.Sprintf("Buffer too small, need at least %d bytes", n)
}

// GoSetenv is for use by a calling application to alter environment variables in
// a manner that will be understood by the Go runtime.  We need it because the standard
// C-language putenv() and setenv() routines do not alter the Go environment as intended,
// due to issues with when os.Getenv() or related routines first get called.  To affect
// the initial config for the services managed by libtransit, calls to GoSetenv() must be
// made *before* any call to one of the routines that might probe for or attempt to start,
// stop, or otherwise interact with one of the services.
//
//export GoSetenv
func GoSetenv(key, value, errBuf *C.char, errBufLen C.size_t) C.bool {
	err := os.Setenv(C.GoString(key), C.GoString(value))
	if err != nil {
		bufStr(errBuf, errBufLen, err.Error())
		return false
	}
	return true
}

// StartController is a C API for services.GetTransitService().StartController
//
//export StartController
func StartController(errBuf *C.char, errBufLen C.size_t) C.bool {
	if err := services.GetTransitService().StartController(); err != nil {
		bufStr(errBuf, errBufLen, err.Error())
		return false
	}
	return true
}

// StopController is a C API for services.GetTransitService().StopController
//
//export StopController
func StopController(errBuf *C.char, errBufLen C.size_t) C.bool {
	if err := services.GetTransitService().StopController(); err != nil {
		bufStr(errBuf, errBufLen, err.Error())
		return false
	}
	return true
}

// StartNats is a C API for services.GetTransitService().StartNats
//
//export StartNats
func StartNats(errBuf *C.char, errBufLen C.size_t) C.bool {
	if err := services.GetTransitService().StartNats(); err != nil {
		bufStr(errBuf, errBufLen, err.Error())
		return false
	}
	return true
}

// StopNats is a C API for services.GetTransitService().StopNats
//
//export StopNats
func StopNats(errBuf *C.char, errBufLen C.size_t) C.bool {
	if err := services.GetTransitService().StopNats(); err != nil {
		bufStr(errBuf, errBufLen, err.Error())
		return false
	}
	return true
}

// StartTransport is a C API for services.GetTransitService().StartTransport
//
//export StartTransport
func StartTransport(errBuf *C.char, errBufLen C.size_t) C.bool {
	if err := services.GetTransitService().StartTransport(); err != nil {
		bufStr(errBuf, errBufLen, err.Error())
		return false
	}
	return true
}

// StopTransport is a C API for services.GetTransitService().StopTransport
//
//export StopTransport
func StopTransport(errBuf *C.char, errBufLen C.size_t) C.bool {
	if err := services.GetTransitService().StopTransport(); err != nil {
		bufStr(errBuf, errBufLen, err.Error())
		return false
	}
	return true
}

// DemandConfig is a C API for services.GetTransitService().DemandConfig
//
//export DemandConfig
func DemandConfig(errBuf *C.char, errBufLen C.size_t) C.bool {
	if err := services.GetTransitService().DemandConfig(); err != nil {
		bufStr(errBuf, errBufLen, err.Error())
		return false
	}
	return true
}

// IsControllerRunning is a C API for services.GetTransitService().Status().Controller
//
//export IsControllerRunning
func IsControllerRunning() C.bool {
	return services.GetTransitService().Status().Controller == services.StatusRunning
}

// IsNatsRunning is a C API for services.GetTransitService().Status().Nats
//
//export IsNatsRunning
func IsNatsRunning() C.bool {
	return services.GetTransitService().Status().Nats == services.StatusRunning
}

// IsTransportRunning is a C API for services.GetTransitService().Status().Transport
//
//export IsTransportRunning
func IsTransportRunning() C.bool {
	return services.GetTransitService().Status().Transport == services.StatusRunning
}

// RegisterListMetricsHandler is a C API for services.GetTransitService().RegisterListMetricsHandler
//
//export RegisterListMetricsHandler
func RegisterListMetricsHandler(fn C.textGetterType) {
	/* See notes on textGetterType and invokeTextGetter */
	services.GetTransitService().RegisterListMetricsHandler(func() ([]byte, error) {
		ptr := C.invokeTextGetter(fn)
		res := []byte(C.GoString(ptr))
		C.free(unsafe.Pointer(ptr))
		return res, nil
	})
}

// RemoveListMetricsHandler is a C API for services.GetTransitService().RemoveListMetricsHandler
//
//export RemoveListMetricsHandler
func RemoveListMetricsHandler() {
	services.GetTransitService().RemoveListMetricsHandler()
}

// RegisterConfigHandler is a C API for services.GetTransitService().RegisterConfigHandler
//
//export RegisterConfigHandler
func RegisterConfigHandler(fn C.textSetterType) {
	services.GetTransitService().RegisterConfigHandler(func(p []byte) {
		ptr := C.CString(string(p))
		C.invokeTextSetter(fn, ptr)
		C.free(unsafe.Pointer(ptr))
	})
}

// RemoveConfigHandler is a C API for services.GetTransitService().RemoveConfigHandler()
//
//export RemoveConfigHandler
func RemoveConfigHandler() {
	services.GetTransitService().RemoveConfigHandler()
}

// GetAgentId returns AgentID
// The C string is allocated in the C heap using malloc.
// It should be freed after use with stdlib free().
//
//export GetAgentId
func GetAgentId() *C.char {
	return C.CString(services.GetTransitService().AgentID)
}

// GetAppName returns AppName
// The C string is allocated in the C heap using malloc.
// It should be freed after use with stdlib free().
//
//export GetAppName
func GetAppName() *C.char {
	return C.CString(services.GetTransitService().AppName)
}

// GetAppType returns AppType
// The C string is allocated in the C heap using malloc.
// It should be freed after use with stdlib free().
//
//export GetAppType
func GetAppType() *C.char {
	return C.CString(services.GetTransitService().AppType)
}
