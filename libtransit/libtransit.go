package main

//#include <stdbool.h>
//#include <stddef.h>
//#include <stdint.h>
//#include <stdlib.h>
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
	"context"
	"encoding/json"
	"fmt"
	"os"
	"runtime/cgo"
	"unsafe"

	"github.com/gwos/tcg/sdk/transit"
	"github.com/gwos/tcg/services"
	"github.com/rs/zerolog/log"
)

func init() {
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

// ClearInDowntime is a C API for services.GetTransitService().ClearInDowntime
//export ClearInDowntime
func ClearInDowntime(payloadJSON, errBuf *C.char, errBufLen C.size_t) bool {
	if err := services.GetTransitService().
		ClearInDowntime(context.Background(), []byte(C.GoString(payloadJSON))); err != nil {
		bufStr(errBuf, errBufLen, err.Error())
		return false
	}
	return true
}

// SetInDowntime is a C API for services.GetTransitService().SetInDowntime
//export SetInDowntime
func SetInDowntime(payloadJSON, errBuf *C.char, errBufLen C.size_t) bool {
	if err := services.GetTransitService().
		SetInDowntime(context.Background(), []byte(C.GoString(payloadJSON))); err != nil {
		bufStr(errBuf, errBufLen, err.Error())
		return false
	}
	return true
}

// SendEvents is a C API for services.GetTransitService().SendEvents
//export SendEvents
func SendEvents(payloadJSON, errBuf *C.char, errBufLen C.size_t) bool {
	if err := services.GetTransitService().
		SendEvents(context.Background(), []byte(C.GoString(payloadJSON))); err != nil {
		bufStr(errBuf, errBufLen, err.Error())
		return false
	}
	return true
}

// SendEventsAck is a C API for services.GetTransitService().SendEventsAck
//export SendEventsAck
func SendEventsAck(payloadJSON, errBuf *C.char, errBufLen C.size_t) bool {
	if err := services.GetTransitService().
		SendEventsAck(context.Background(), []byte(C.GoString(payloadJSON))); err != nil {
		bufStr(errBuf, errBufLen, err.Error())
		return false
	}
	return true
}

// SendEventsUnack is a C API for services.GetTransitService().SendEventsUnack
//export SendEventsUnack
func SendEventsUnack(payloadJSON, errBuf *C.char, errBufLen C.size_t) bool {
	if err := services.GetTransitService().
		SendEventsUnack(context.Background(), []byte(C.GoString(payloadJSON))); err != nil {
		bufStr(errBuf, errBufLen, err.Error())
		return false
	}
	return true
}

// SendResourcesWithMetrics is a C API for services.GetTransitService().SendResourceWithMetrics
//export SendResourcesWithMetrics
func SendResourcesWithMetrics(payloadJSON, errBuf *C.char, errBufLen C.size_t) bool {
	if err := services.GetTransitService().
		SendResourceWithMetrics(context.Background(), []byte(C.GoString(payloadJSON))); err != nil {
		bufStr(errBuf, errBufLen, err.Error())
		return false
	}
	return true
}

// SynchronizeInventory is a C API for services.GetTransitService().SynchronizeInventory
//export SynchronizeInventory
func SynchronizeInventory(payloadJSON, errBuf *C.char, errBufLen C.size_t) bool {
	if err := services.GetTransitService().
		SynchronizeInventory(context.Background(), []byte(C.GoString(payloadJSON))); err != nil {
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
	return services.GetTransitService().Status().Controller == services.StatusRunning
}

// IsNatsRunning is a C API for services.GetTransitService().Status().Nats
//export IsNatsRunning
func IsNatsRunning() bool {
	return services.GetTransitService().Status().Nats == services.StatusRunning
}

// IsTransportRunning is a C API for services.GetTransitService().Status().Transport
//export IsTransportRunning
func IsTransportRunning() bool {
	return services.GetTransitService().Status().Transport == services.StatusRunning
}

// RegisterListMetricsHandler is a C API for services.GetTransitService().RegisterListMetricsHandler
//export RegisterListMetricsHandler
func RegisterListMetricsHandler(fn C.getTextHandlerType) {
	/* See notes on getTextHandlerType and invokeGetTextHandler */
	services.GetTransitService().RegisterListMetricsHandler(func() ([]byte, error) {
		textPtr := C.invokeGetTextHandler(fn)
		res := []byte(C.GoString(textPtr))
		C.free(unsafe.Pointer(textPtr))
		return res, nil
	})
}

// RemoveListMetricsHandler is a C API for services.GetTransitService().RemoveListMetricsHandler
//export RemoveListMetricsHandler
func RemoveListMetricsHandler() {
	services.GetTransitService().RemoveListMetricsHandler()
}

// RegisterDemandConfigHandler is a C API for services.GetTransitService().RegisterDemandConfigHandler
//export RegisterDemandConfigHandler
func RegisterDemandConfigHandler(fn C.demandConfigHandler) {
	services.GetTransitService().RegisterDemandConfigHandler(func() bool {
		return bool(C.invokeDemandConfigHandler(fn))
	})
}

// RemoveDemandConfigHandler is a C API for services.GetTransitService().RemoveDemandConfigHandler()
//export RemoveDemandConfigHandler
func RemoveDemandConfigHandler() {
	services.GetTransitService().RemoveDemandConfigHandler()
}

// GetAgentIdentity is a C API for getting AgentIdentity
//export GetAgentIdentity
func GetAgentIdentity(buf *C.char, bufLen C.size_t, errBuf *C.char, errBufLen C.size_t) bool {
	res, err := json.Marshal(services.GetTransitService().Connector.AgentIdentity)
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

/* Extended interface with cgo handles */

// CreateInventoryRequest returns a handle
//export CreateInventoryRequest
func CreateInventoryRequest() C.uintptr_t {
	p := new(transit.InventoryRequest)
	return C.uintptr_t(cgo.NewHandle(p))
}

// CreateInventoryResource returns a handle
//export CreateInventoryResource
func CreateInventoryResource(
	name *C.char,
	resType *C.char,
) C.uintptr_t {
	p := new(transit.InventoryResource)
	p.Name = C.GoString(name)
	p.Type = transit.ResourceType(C.GoString(resType))
	return C.uintptr_t(cgo.NewHandle(p))
}

// CreateInventoryService returns a handle
//export CreateInventoryService
func CreateInventoryService(
	name *C.char,
	resType *C.char,
) C.uintptr_t {
	p := new(transit.InventoryService)
	p.Name = C.GoString(name)
	p.Type = transit.ResourceType(C.GoString(resType))
	return C.uintptr_t(cgo.NewHandle(p))
}

// CreateMonitoredResource returns a handle
//export CreateMonitoredResource
func CreateMonitoredResource(
	name *C.char,
	resType *C.char,
) C.uintptr_t {
	p := new(transit.MonitoredResource)
	p.Name = C.GoString(name)
	p.Type = transit.ResourceType(C.GoString(resType))
	return C.uintptr_t(cgo.NewHandle(p))
}

// CreateMonitoredService returns a handle
//export CreateMonitoredService
func CreateMonitoredService(
	name *C.char,
	resType *C.char,
) C.uintptr_t {
	p := new(transit.MonitoredService)
	p.Name = C.GoString(name)
	p.Type = transit.ResourceType(C.GoString(resType))
	return C.uintptr_t(cgo.NewHandle(p))
}

// DeleteHandle invalidates a handle.
// This method should only be called once the C code no longer has a copy of the handle value.
//export DeleteHandle
func DeleteHandle(handle *C.uintptr_t) {
	cgo.Handle(*handle).Delete()
}

// AddResource appends resource into inventory
//export AddResource
func AddResource(pReq *C.uintptr_t, pRes C.uintptr_t) {
	var v interface{}
	hReq, hRes := cgo.Handle(*pReq), cgo.Handle(pRes)
	req := hReq.Value().(*transit.InventoryRequest)

	switch hRes.Value().(type) {
	case *transit.InventoryResource:
		res := hRes.Value().(*transit.InventoryResource)
		req.Resources = append(req.Resources, *res)
		v = req
	case *transit.MonitoredResource:
		res := hRes.Value().(*transit.MonitoredResource).ToInventoryResource()
		req.Resources = append(req.Resources, res)
		v = req
	}
	if v != nil {
		hReq.Delete()
		*pReq = C.uintptr_t(cgo.NewHandle(v))
	}
}

// AddService appends service in resource
//export AddService
func AddService(pRes *C.uintptr_t, pSvc C.uintptr_t) {
	var v interface{}
	hRes, hSvc := cgo.Handle(*pRes), cgo.Handle(pSvc)
	switch hRes.Value().(type) {
	case *transit.InventoryResource:
		res := hRes.Value().(*transit.InventoryResource)
		switch hSvc.Value().(type) {
		case *transit.InventoryService:
			svc := hSvc.Value().(*transit.InventoryService)
			res.Services = append(res.Services, *svc)
			v = res
		case *transit.MonitoredService:
			svc := hSvc.Value().(*transit.MonitoredService).ToInventoryService()
			res.Services = append(res.Services, svc)
			v = res
		}
	case *transit.MonitoredResource:
		res := hRes.Value().(*transit.MonitoredResource)
		switch hSvc.Value().(type) {
		case *transit.InventoryService:
			svc := hSvc.Value().(*transit.InventoryService)
			s := new(transit.MonitoredService)
			s.BaseTransitData = svc.BaseTransitData
			s.Status = transit.ServicePending
			res.Services = append(res.Services, *s)
			v = res
		case *transit.MonitoredService:
			svc := hSvc.Value().(*transit.MonitoredService)
			res.Services = append(res.Services, *svc)
			v = res
		}
	}
	if v != nil {
		hRes.Delete()
		*pRes = C.uintptr_t(cgo.NewHandle(v))
	}
}

//export SetName
func SetName(p *C.uintptr_t, s *C.char) {
	var v interface{}
	h := cgo.Handle(*p)
	switch h.Value().(type) {
	case *transit.InventoryResource:
		val := h.Value().(*transit.InventoryResource)
		val.Name = C.GoString(s)
		v = val
	case *transit.InventoryService:
		val := h.Value().(*transit.InventoryService)
		val.Name = C.GoString(s)
		v = val
	case *transit.MonitoredResource:
		val := h.Value().(*transit.MonitoredResource)
		val.Name = C.GoString(s)
		v = val
	case *transit.MonitoredService:
		val := h.Value().(*transit.MonitoredService)
		val.Name = C.GoString(s)
		v = val
	}
	if v != nil {
		h.Delete()
		*p = C.uintptr_t(cgo.NewHandle(v))
	}
}

//export SendInventory
func SendInventory(pInvReq C.uintptr_t, errBuf *C.char, errBufLen C.size_t) bool {
	invReq := cgo.Handle(pInvReq).Value().(*transit.InventoryRequest)
	invReq.Context = services.GetTransitService().MakeTracerContext()
	p, err := json.Marshal(invReq)
	log.Debug().Err(err).RawJSON("payload", p).Msg("inventory")
	if err != nil {
		bufStr(errBuf, errBufLen, err.Error())
		return false
	}
	if err := services.GetTransitService().
		SynchronizeInventory(context.Background(), p); err != nil {
		bufStr(errBuf, errBufLen, err.Error())
		return false
	}
	return true
}
