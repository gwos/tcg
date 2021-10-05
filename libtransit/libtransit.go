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
	"time"
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
func GetAgentIdentity(buf *C.char, bufLen C.size_t, errBuf *C.char, errBufLen C.size_t) C.bool {
	res, err := json.Marshal(services.GetTransitService().Connector.AgentIdentity)
	if err != nil {
		bufStr(errBuf, errBufLen, err.Error())
		return false
	}
	cStrLen := len(res) + 1
	if cStrLen > int(bufLen) {
		bufStr(errBuf, errBufLen, msgfBufTooSmall(cStrLen))
		return false
	}
	bufStr(buf, bufLen, string(res))
	return true
}

/* Extended interface with cgo handles */

// AddMetric appends metric value to target
//export AddMetric
func AddMetric(target C.uintptr_t, value C.uintptr_t) {
	h, h2 := cgo.Handle(target), cgo.Handle(value)
	if hv, ok := h.Value().(interface{ AddMetric(transit.TimeSeries) }); ok {
		if hv2, ok := h2.Value().(*transit.TimeSeries); ok {
			hv.AddMetric(*hv2)
		}
	}
}

// AddResource appends resource value to target
//export AddResource
func AddResource(target C.uintptr_t, value C.uintptr_t) {
	h, h2 := cgo.Handle(target), cgo.Handle(value)
	if hv, ok := h.Value().(interface {
		AddResource(transit.InventoryResource)
	}); ok {
		if hv2, ok := h2.Value().(*transit.InventoryResource); ok {
			hv.AddResource(*hv2)
			return
		}
	}
	if hv, ok := h.Value().(interface {
		AddResource(transit.MonitoredResource)
	}); ok {
		if hv2, ok := h2.Value().(*transit.MonitoredResource); ok {
			hv.AddResource(*hv2)
			return
		}
	}
	if hv, ok := h.Value().(interface{ AddResource(transit.ResourceRef) }); ok {
		if hv2, ok := h2.Value().(*transit.ResourceRef); ok {
			hv.AddResource(*hv2)
			return
		}
		if hv2, ok := h2.Value().(interface{ ToResourceRef() transit.ResourceRef }); ok {
			hv.AddResource(hv2.ToResourceRef())
			return
		}
	}
}

// AddResourceGroup appends resource group value to target
//export AddResourceGroup
func AddResourceGroup(target C.uintptr_t, value C.uintptr_t) {
	h, h2 := cgo.Handle(target), cgo.Handle(value)
	if hv, ok := h.Value().(interface{ AddResourceGroup(transit.ResourceGroup) }); ok {
		if hv2, ok := h2.Value().(*transit.ResourceGroup); ok {
			hv.AddResourceGroup(*hv2)
			return
		}
	}
}

// AddService appends service value to target
//export AddService
func AddService(target C.uintptr_t, value C.uintptr_t) {
	h, h2 := cgo.Handle(target), cgo.Handle(value)
	if hv, ok := h.Value().(interface {
		AddService(transit.InventoryService)
	}); ok {
		if hv2, ok := h2.Value().(*transit.InventoryService); ok {
			hv.AddService(*hv2)
			return
		}
	}
	if hv, ok := h.Value().(interface {
		AddService(transit.MonitoredService)
	}); ok {
		if hv2, ok := h2.Value().(*transit.MonitoredService); ok {
			hv.AddService(*hv2)
			return
		}
	}
}

// AddThreshold appends threshold value to target
//export AddThreshold
func AddThreshold(target C.uintptr_t, value C.uintptr_t) {
	h, h2 := cgo.Handle(target), cgo.Handle(value)
	if hv, ok := h.Value().(interface{ AddThreshold(transit.ThresholdValue) }); ok {
		if hv2, ok := h2.Value().(*transit.ThresholdValue); ok {
			hv.AddThreshold(*hv2)
		}
	}
}

// AddThresholdDouble appends threshold to target
//export AddThresholdDouble
func AddThresholdDouble(target C.uintptr_t, lbl, sType *C.char, value C.double) {
	h := cgo.Handle(target)
	if hv, ok := h.Value().(interface{ AddThreshold(transit.ThresholdValue) }); ok {
		hv.AddThreshold(transit.ThresholdValue{
			SampleType: transit.MetricSampleType(C.GoString(sType)),
			Label:      C.GoString(lbl),
			Value:      transit.NewTypedValue(float64(value)),
		})
	}
}

// AddThresholdInt appends threshold to target
//export AddThresholdInt
func AddThresholdInt(target C.uintptr_t, lbl, sType *C.char, value C.longlong) {
	h := cgo.Handle(target)
	if hv, ok := h.Value().(interface{ AddThreshold(transit.ThresholdValue) }); ok {
		hv.AddThreshold(transit.ThresholdValue{
			SampleType: transit.MetricSampleType(C.GoString(sType)),
			Label:      C.GoString(lbl),
			Value:      transit.NewTypedValue(int64(value)),
		})
	}
}

// CalcStatus calculates status depending on handle:
// status of resource and services
// status of service
//export CalcStatus
func CalcStatus(target C.uintptr_t) {
	h := cgo.Handle(target)
	if hv, ok := h.Value().(*transit.MonitoredResource); ok {
		hv.SetStatus(transit.CalculateResourceStatus(hv.Services))
		for i := range hv.Services {
			if status, err := transit.CalculateServiceStatus(&hv.Services[i].Metrics); err == nil {
				hv.Services[i].SetStatus(status)
			}
		}
		return
	}
	if hv, ok := h.Value().(*transit.MonitoredService); ok {
		if status, err := transit.CalculateServiceStatus(&hv.Metrics); err == nil {
			hv.SetStatus(status)
		}
		return
	}
}

// CreateInventoryRequest returns a handle
//export CreateInventoryRequest
func CreateInventoryRequest() C.uintptr_t {
	p := new(transit.InventoryRequest)
	p.Context = services.GetTransitService().MakeTracerContext()
	p.Resources = []transit.InventoryResource{}
	return C.uintptr_t(cgo.NewHandle(p))
}

// CreateInventoryResource returns a handle
//export CreateInventoryResource
func CreateInventoryResource(name *C.char, resType *C.char) C.uintptr_t {
	p := new(transit.InventoryResource)
	p.Name = C.GoString(name)
	p.Type = transit.ResourceType(C.GoString(resType))
	p.Services = []transit.InventoryService{}
	return C.uintptr_t(cgo.NewHandle(p))
}

// CreateInventoryService returns a handle
//export CreateInventoryService
func CreateInventoryService(name *C.char, resType *C.char) C.uintptr_t {
	p := new(transit.InventoryService)
	p.Name = C.GoString(name)
	p.Type = transit.ResourceType(C.GoString(resType))
	return C.uintptr_t(cgo.NewHandle(p))
}

// CreateMonitoredResource returns a handle
//export CreateMonitoredResource
func CreateMonitoredResource(name *C.char, resType *C.char) C.uintptr_t {
	p := new(transit.MonitoredResource)
	p.Name = C.GoString(name)
	p.Type = transit.ResourceType(C.GoString(resType))
	p.Services = []transit.MonitoredService{}
	return C.uintptr_t(cgo.NewHandle(p))
}

// CreateMonitoredService returns a handle
//export CreateMonitoredService
func CreateMonitoredService(name *C.char, resType *C.char) C.uintptr_t {
	p := new(transit.MonitoredService)
	p.Name = C.GoString(name)
	p.Type = transit.ResourceType(C.GoString(resType))
	p.Metrics = []transit.TimeSeries{}
	return C.uintptr_t(cgo.NewHandle(p))
}

// CreateResourceGroup returns a handle
//export CreateResourceGroup
func CreateResourceGroup(name *C.char, grType *C.char) C.uintptr_t {
	p := new(transit.ResourceGroup)
	p.GroupName = C.GoString(name)
	p.Type = transit.GroupType(C.GoString(grType))
	p.Resources = []transit.ResourceRef{}
	return C.uintptr_t(cgo.NewHandle(p))
}

// CreateResourcesWithServicesRequest returns a handle
//export CreateResourcesWithServicesRequest
func CreateResourcesWithServicesRequest() C.uintptr_t {
	p := new(transit.ResourcesWithServicesRequest)
	p.Context = services.GetTransitService().MakeTracerContext()
	p.Resources = []transit.MonitoredResource{}
	return C.uintptr_t(cgo.NewHandle(p))
}

// CreateThresholdValue returns a handle
//export CreateThresholdValue
func CreateThresholdValue(
	lbl *C.char,
	sType *C.char,
) C.uintptr_t {
	p := new(transit.ThresholdValue)
	p.Label = C.GoString(lbl)
	p.SampleType = transit.MetricSampleType(C.GoString(sType))
	return C.uintptr_t(cgo.NewHandle(p))
}

// CreateTimeSeries returns a handle
//export CreateTimeSeries
func CreateTimeSeries(
	name *C.char,
) C.uintptr_t {
	p := new(transit.TimeSeries)
	p.MetricName = C.GoString(name)
	return C.uintptr_t(cgo.NewHandle(p))
}

// DeleteHandle invalidates a handle.
// This method should only be called once the C code no longer has a copy of the handle value.
//export DeleteHandle
func DeleteHandle(target C.uintptr_t) {
	cgo.Handle(target).Delete()
}

// SetCategory sets category value on target
//export SetCategory
func SetCategory(target C.uintptr_t, value *C.char) {
	h := cgo.Handle(target)
	if hv, ok := h.Value().(interface{ SetCategory(string) }); ok {
		hv.SetCategory(C.GoString(value))
	}
}

// SetContextTimestamp sets context timestamp value on target
//export SetContextTimestamp
func SetContextTimestamp(target C.uintptr_t, sec, nsec C.longlong) {
	h := cgo.Handle(target)
	if hv, ok := h.Value().(interface{ SetContext(transit.TracerContext) }); ok {
		t := transit.NewTimestamp()
		t.Time = time.Unix(int64(sec), int64(nsec)).UTC()
		hv.SetContext(transit.TracerContext{TimeStamp: t})
	}
}

// SetContextToken sets context token value on target
//export SetContextToken
func SetContextToken(target C.uintptr_t, value *C.char) {
	h := cgo.Handle(target)
	if hv, ok := h.Value().(interface{ SetContext(transit.TracerContext) }); ok {
		hv.SetContext(transit.TracerContext{TraceToken: C.GoString(value)})
	}
}

// SetDescription sets description value on target
//export SetDescription
func SetDescription(target C.uintptr_t, value *C.char) {
	h := cgo.Handle(target)
	if hv, ok := h.Value().(interface{ SetDescription(string) }); ok {
		hv.SetDescription(C.GoString(value))
	}
}

// SetDevice sets device value on target
//export SetDevice
func SetDevice(target C.uintptr_t, value *C.char) {
	h := cgo.Handle(target)
	if v, ok := h.Value().(interface{ SetDevice(string) }); ok {
		v.SetDevice(C.GoString(value))
	}
}

// SetIntervalEnd sets interval on target
//export SetIntervalEnd
func SetIntervalEnd(target C.uintptr_t, sec, nsec C.longlong) {
	h := cgo.Handle(target)
	if hv, ok := h.Value().(interface{ SetIntervalEnd(*transit.Timestamp) }); ok {
		t := transit.NewTimestamp()
		t.Time = time.Unix(int64(sec), int64(nsec)).UTC()
		hv.SetIntervalEnd(t)
	}
}

// SetIntervalStart sets interval on target
//export SetIntervalStart
func SetIntervalStart(target C.uintptr_t, sec, nsec C.longlong) {
	h := cgo.Handle(target)
	if hv, ok := h.Value().(interface{ SetIntervalStart(*transit.Timestamp) }); ok {
		t := transit.NewTimestamp()
		t.Time = time.Unix(int64(sec), int64(nsec)).UTC()
		hv.SetIntervalStart(t)
	}
}

// SetLastPluginOutput sets LastPluginOutput value on target
//export SetLastPluginOutput
func SetLastPluginOutput(target C.uintptr_t, value *C.char) {
	h := cgo.Handle(target)
	if hv, ok := h.Value().(interface{ SetLastPluginOutput(string) }); ok {
		hv.SetLastPluginOutput(C.GoString(value))
	}
}

// SetLastCheckTime sets LastCheckTime on target
//export SetLastCheckTime
func SetLastCheckTime(target C.uintptr_t, sec, nsec C.longlong) {
	h := cgo.Handle(target)
	if hv, ok := h.Value().(interface{ SetLastCheckTime(*transit.Timestamp) }); ok {
		t := transit.NewTimestamp()
		t.Time = time.Unix(int64(sec), int64(nsec)).UTC()
		hv.SetLastCheckTime(t)
	}
}

// SetNextCheckTime sets NextCheckTime on target
//export SetNextCheckTime
func SetNextCheckTime(target C.uintptr_t, sec, nsec C.longlong) {
	h := cgo.Handle(target)
	if hv, ok := h.Value().(interface{ SetNextCheckTime(*transit.Timestamp) }); ok {
		t := transit.NewTimestamp()
		t.Time = time.Unix(int64(sec), int64(nsec)).UTC()
		hv.SetNextCheckTime(t)
	}
}

// SetName sets name value on target
//export SetName
func SetName(target C.uintptr_t, value *C.char) {
	h := cgo.Handle(target)
	if hv, ok := h.Value().(interface{ SetName(string) }); ok {
		hv.SetName(C.GoString(value))
	}
}

// SetOwner sets owner value on target
//export SetOwner
func SetOwner(target C.uintptr_t, value *C.char) {
	h := cgo.Handle(target)
	if hv, ok := h.Value().(interface{ SetOwner(string) }); ok {
		hv.SetOwner(C.GoString(value))
	}
}

// SetPropertyBool sets key:value property on target
//export SetPropertyBool
func SetPropertyBool(target C.uintptr_t, key *C.char, value C.bool) {
	h := cgo.Handle(target)
	if hv, ok := h.Value().(interface{ SetProperty(string, interface{}) }); ok {
		hv.SetProperty(C.GoString(key), bool(value))
	}
}

// SetPropertyDouble sets key:value property on target
//export SetPropertyDouble
func SetPropertyDouble(target C.uintptr_t, key *C.char, value C.double) {
	h := cgo.Handle(target)
	if hv, ok := h.Value().(interface{ SetProperty(string, interface{}) }); ok {
		hv.SetProperty(C.GoString(key), float64(value))
	}
}

// SetPropertyInt sets key:value property on target
//export SetPropertyInt
func SetPropertyInt(target C.uintptr_t, key *C.char, value C.longlong) {
	h := cgo.Handle(target)
	if hv, ok := h.Value().(interface{ SetProperty(string, interface{}) }); ok {
		hv.SetProperty(C.GoString(key), int64(value))
	}
}

// SetPropertyStr sets key:value property on target
//export SetPropertyStr
func SetPropertyStr(target C.uintptr_t, key *C.char, value *C.char) {
	h := cgo.Handle(target)
	if hv, ok := h.Value().(interface{ SetProperty(string, interface{}) }); ok {
		hv.SetProperty(C.GoString(key), C.GoString(value))
	}
}

// SetPropertyTime sets key:timestamp property on target
//export SetPropertyTime
func SetPropertyTime(target C.uintptr_t, key *C.char, sec, nsec C.longlong) {
	h := cgo.Handle(target)
	if hv, ok := h.Value().(interface{ SetProperty(string, interface{}) }); ok {
		hv.SetProperty(C.GoString(key),
			transit.Timestamp{Time: time.Unix(int64(sec), int64(nsec)).UTC()})
	}
}

// SetSampleType sets status value on target
//export SetSampleType
func SetSampleType(target C.uintptr_t, value *C.char) {
	h := cgo.Handle(target)
	if hv, ok := h.Value().(interface {
		SetSampleType(transit.MetricSampleType)
	}); ok {
		hv.SetSampleType(transit.MetricSampleType(C.GoString(value)))
	}
}

// SetStatus sets status value on target
//export SetStatus
func SetStatus(target C.uintptr_t, value *C.char) {
	h := cgo.Handle(target)
	if hv, ok := h.Value().(interface{ SetStatus(transit.MonitorStatus) }); ok {
		hv.SetStatus(transit.MonitorStatus(C.GoString(value)))
	}
}

// SetTag sets key:value tag on target
//export SetTag
func SetTag(target C.uintptr_t, key, value *C.char) {
	h := cgo.Handle(target)
	if hv, ok := h.Value().(interface{ SetTag(string, string) }); ok {
		hv.SetTag(C.GoString(key), C.GoString(value))
	}
}

// SetType sets type value on target
//export SetType
func SetType(target C.uintptr_t, value *C.char) {
	h := cgo.Handle(target)
	if hv, ok := h.Value().(interface{ SetType(transit.GroupType) }); ok {
		hv.SetType(transit.GroupType(C.GoString(value)))
		return
	}
	if hv, ok := h.Value().(interface{ SetType(transit.ResourceType) }); ok {
		hv.SetType(transit.ResourceType(C.GoString(value)))
		return
	}
}

// SetUnit sets type value on target
//export SetUnit
func SetUnit(target C.uintptr_t, value *C.char) {
	h := cgo.Handle(target)
	if hv, ok := h.Value().(interface{ SetUnit(transit.UnitType) }); ok {
		hv.SetUnit(transit.UnitType(C.GoString(value)))
	}
}

// SetValueBool sets value to target
//export SetValueBool
func SetValueBool(target C.uintptr_t, value C.bool) {
	h := cgo.Handle(target)
	if hv, ok := h.Value().(interface{ SetValue(interface{}) }); ok {
		hv.SetValue(bool(value))
	}
}

// SetValueDouble sets value to target
//export SetValueDouble
func SetValueDouble(target C.uintptr_t, value C.double) {
	h := cgo.Handle(target)
	if hv, ok := h.Value().(interface{ SetValue(interface{}) }); ok {
		hv.SetValue(float64(value))
	}
}

// SetValueInt sets value to target
//export SetValueInt
func SetValueInt(target C.uintptr_t, value C.longlong) {
	h := cgo.Handle(target)
	if hv, ok := h.Value().(interface{ SetValue(interface{}) }); ok {
		hv.SetValue(int64(value))
	}
}

// SetValueStr sets value to target
//export SetValueStr
func SetValueStr(target C.uintptr_t, value *C.char) {
	h := cgo.Handle(target)
	if hv, ok := h.Value().(interface{ SetValue(interface{}) }); ok {
		hv.SetValue(C.GoString(value))
	}
}

// SetValueTime sets timestamp value to target
//export SetValueTime
func SetValueTime(target C.uintptr_t, sec, nsec C.longlong) {
	h := cgo.Handle(target)
	if hv, ok := h.Value().(interface{ SetValue(interface{}) }); ok {
		hv.SetValue(
			transit.Timestamp{Time: time.Unix(int64(sec), int64(nsec)).UTC()})
	}
}

// MarshalIndentJSON is like Marshal but applies Indent to format the output.
// Each JSON element in the output will begin on a new line beginning with prefix
// followed by one or more copies of indent according to the indentation nesting.
//export MarshallIndentJSON
func MarshallIndentJSON(
	target C.uintptr_t,
	prefix *C.char, indent *C.char,
	buf *C.char, bufLen C.size_t,
	errBuf *C.char, errBufLen C.size_t,
) C.bool {
	h := cgo.Handle(target)
	hv := h.Value()
	bb, err := json.MarshalIndent(hv, C.GoString(prefix), C.GoString(indent))
	if err != nil {
		bufStr(errBuf, errBufLen, err.Error())
		return false
	}
	cStrLen := len(bb) + 1
	if cStrLen > int(bufLen) {
		bufStr(errBuf, errBufLen, msgfBufTooSmall(cStrLen))
		return false
	}
	bufStr(buf, bufLen, string(bb))
	return true
}

// SendInventory sends inventory request
//export SendInventory
func SendInventory(req C.uintptr_t, errBuf *C.char, errBufLen C.size_t) C.bool {
	hv := cgo.Handle(req).Value().(*transit.InventoryRequest)
	bb, err := json.Marshal(hv)
	log.Debug().Err(err).RawJSON("payload", bb).Msg("SendInventory")
	if err != nil {
		bufStr(errBuf, errBufLen, err.Error())
		return false
	}
	if err := services.GetTransitService().
		SynchronizeInventory(context.Background(), bb); err != nil {
		bufStr(errBuf, errBufLen, err.Error())
		return false
	}
	return true
}

// SendMetrics sends metrics request
//export SendMetrics
func SendMetrics(req C.uintptr_t, errBuf *C.char, errBufLen C.size_t) C.bool {
	hv := cgo.Handle(req).Value().(*transit.ResourcesWithServicesRequest)
	bb, err := json.Marshal(hv)
	log.Debug().Err(err).RawJSON("payload", bb).Msg("SendMetrics")
	if err != nil {
		bufStr(errBuf, errBufLen, err.Error())
		return false
	}
	if err := services.GetTransitService().
		SendResourceWithMetrics(context.Background(), bb); err != nil {
		bufStr(errBuf, errBufLen, err.Error())
		return false
	}
	return true
}
