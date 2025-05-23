package main

/*
#include <stdbool.h>
#include <stdint.h>

typedef const char cchar_t;
*/
import "C"
import (
	"context"
	"encoding/json"
	"fmt"
	"runtime/cgo"
	"time"

	"github.com/gwos/tcg/sdk/transit"
	"github.com/gwos/tcg/services"
	"github.com/rs/zerolog/log"
)

// CreateInventoryRequest creates payload for SendInventory API.
// It returns a handle that should be deleted after use with DeleteHandle.
//
//export CreateInventoryRequest
func CreateInventoryRequest() C.uintptr_t {
	p := new(transit.InventoryRequest)
	p.Context = services.GetTransitService().MakeTracerContext()
	p.Resources = []transit.InventoryResource{}
	return C.uintptr_t(cgo.NewHandle(p))
}

// CreateInventoryResource creates an object used in InventoryRequest.
// It returns a handle that should be deleted after use with DeleteHandle.
//
//export CreateInventoryResource
func CreateInventoryResource(name, resType *C.cchar_t) C.uintptr_t {
	p := new(transit.InventoryResource)
	p.Name = C.GoString(name)
	p.Type = transit.ResourceType(C.GoString(resType))
	p.Services = []transit.InventoryService{}
	return C.uintptr_t(cgo.NewHandle(p))
}

// CreateInventoryService creates an object used in InventoryResource.
// It returns a handle that should be deleted after use with DeleteHandle.
//
//export CreateInventoryService
func CreateInventoryService(name, resType *C.cchar_t) C.uintptr_t {
	p := new(transit.InventoryService)
	p.Name = C.GoString(name)
	p.Type = transit.ResourceType(C.GoString(resType))
	return C.uintptr_t(cgo.NewHandle(p))
}

// CreateMonitoredResource creates an object used in ResourcesWithServicesRequest.
// It returns a handle that should be deleted after use with DeleteHandle.
//
//export CreateMonitoredResource
func CreateMonitoredResource(name, resType *C.cchar_t) C.uintptr_t {
	p := new(transit.MonitoredResource)
	p.Name = C.GoString(name)
	p.Type = transit.ResourceType(C.GoString(resType))
	p.Services = []transit.MonitoredService{}
	return C.uintptr_t(cgo.NewHandle(p))
}

// CreateMonitoredService creates an object used in MonitoredResource.
// It returns a handle that should be deleted after use with DeleteHandle.
//
//export CreateMonitoredService
func CreateMonitoredService(name, resType *C.cchar_t) C.uintptr_t {
	p := new(transit.MonitoredService)
	p.Name = C.GoString(name)
	p.Type = transit.ResourceType(C.GoString(resType))
	p.Metrics = []transit.TimeSeries{}
	return C.uintptr_t(cgo.NewHandle(p))
}

// CreateResourceGroup creates an object used in InventoryRequest and ResourcesWithServicesRequest.
// It returns a handle that should be deleted after use with DeleteHandle.
//
//export CreateResourceGroup
func CreateResourceGroup(name, grType *C.cchar_t) C.uintptr_t {
	p := new(transit.ResourceGroup)
	p.GroupName = C.GoString(name)
	p.Type = transit.GroupType(C.GoString(grType))
	p.Resources = []transit.ResourceRef{}
	return C.uintptr_t(cgo.NewHandle(p))
}

// CreateResourcesWithServicesRequest creates payload for SendMetrics API.
// It returns a handle that should be deleted after use with DeleteHandle.
//
//export CreateResourcesWithServicesRequest
func CreateResourcesWithServicesRequest() C.uintptr_t {
	p := new(transit.ResourcesWithServicesRequest)
	p.Context = services.GetTransitService().MakeTracerContext()
	p.Resources = []transit.MonitoredResource{}
	return C.uintptr_t(cgo.NewHandle(p))
}

// CreateThresholdValue creates an object used in TimeSeries.
// It returns a handle that should be deleted after use with DeleteHandle.
//
//export CreateThresholdValue
func CreateThresholdValue(lbl, sType *C.cchar_t) C.uintptr_t {
	p := new(transit.ThresholdValue)
	p.Label = C.GoString(lbl)
	p.SampleType = transit.MetricSampleType(C.GoString(sType))
	return C.uintptr_t(cgo.NewHandle(p))
}

// CreateTimeSeries creates an object used in MonitoredResource.
// It returns a handle that should be deleted after use with DeleteHandle.
//
//export CreateTimeSeries
func CreateTimeSeries(name *C.cchar_t) C.uintptr_t {
	p := new(transit.TimeSeries)
	p.MetricName = C.GoString(name)
	return C.uintptr_t(cgo.NewHandle(p))
}

// DeleteHandle invalidates a handle.
// This method should only be called once the C code no longer has a copy of the handle value.
//
//export DeleteHandle
func DeleteHandle(target C.uintptr_t) {
	cgo.Handle(target).Delete()
}

// AddMetric appends metric value to target
//
//export AddMetric
func AddMetric(target, value C.uintptr_t) {
	h, h2 := cgo.Handle(target), cgo.Handle(value)
	if hv, ok := h.Value().(interface{ AddMetric(transit.TimeSeries) }); ok {
		if hv2, ok := h2.Value().(*transit.TimeSeries); ok {
			hv.AddMetric(*hv2)
		}
	}
}

// AddResource appends resource value to target
//
//export AddResource
func AddResource(target, value C.uintptr_t) {
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
//
//export AddResourceGroup
func AddResourceGroup(target, value C.uintptr_t) {
	h, h2 := cgo.Handle(target), cgo.Handle(value)
	if hv, ok := h.Value().(interface{ AddResourceGroup(transit.ResourceGroup) }); ok {
		if hv2, ok := h2.Value().(*transit.ResourceGroup); ok {
			hv.AddResourceGroup(*hv2)
			return
		}
	}
}

// AddService appends service value to target
//
//export AddService
func AddService(target, value C.uintptr_t) {
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
//
//export AddThreshold
func AddThreshold(target, value C.uintptr_t) {
	h, h2 := cgo.Handle(target), cgo.Handle(value)
	if hv, ok := h.Value().(interface{ AddThreshold(transit.ThresholdValue) }); ok {
		if hv2, ok := h2.Value().(*transit.ThresholdValue); ok {
			hv.AddThreshold(*hv2)
		}
	}
}

// AddThresholdDouble appends threshold to target
//
//export AddThresholdDouble
func AddThresholdDouble(target C.uintptr_t, lbl, sType *C.cchar_t, value C.double) {
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
//
//export AddThresholdInt
func AddThresholdInt(target C.uintptr_t, lbl, sType *C.cchar_t, value C.longlong) {
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
//
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

// SetCategory sets category value on target
//
//export SetCategory
func SetCategory(target C.uintptr_t, value *C.cchar_t) {
	h := cgo.Handle(target)
	if hv, ok := h.Value().(interface{ SetCategory(string) }); ok {
		hv.SetCategory(C.GoString(value))
	}
}

// SetContextTimestamp sets context timestamp value on target
//
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
//
//export SetContextToken
func SetContextToken(target C.uintptr_t, value *C.cchar_t) {
	h := cgo.Handle(target)
	if hv, ok := h.Value().(interface{ SetContext(transit.TracerContext) }); ok {
		hv.SetContext(transit.TracerContext{TraceToken: C.GoString(value)})
	}
}

// SetDescription sets description value on target
//
//export SetDescription
func SetDescription(target C.uintptr_t, value *C.cchar_t) {
	h := cgo.Handle(target)
	if hv, ok := h.Value().(interface{ SetDescription(string) }); ok {
		hv.SetDescription(C.GoString(value))
	}
}

// SetDevice sets device value on target
//
//export SetDevice
func SetDevice(target C.uintptr_t, value *C.char) {
	h := cgo.Handle(target)
	if hv, ok := h.Value().(interface{ SetDevice(string) }); ok {
		hv.SetDevice(C.GoString(value))
	}
}

// SetIntervalEnd sets interval on target
//
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
//
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
//
//export SetLastPluginOutput
func SetLastPluginOutput(target C.uintptr_t, value *C.cchar_t) {
	h := cgo.Handle(target)
	if hv, ok := h.Value().(interface{ SetLastPluginOutput(string) }); ok {
		hv.SetLastPluginOutput(C.GoString(value))
	}
}

// SetLastCheckTime sets LastCheckTime on target
//
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
//
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
//
//export SetName
func SetName(target C.uintptr_t, value *C.cchar_t) {
	h := cgo.Handle(target)
	if hv, ok := h.Value().(interface{ SetName(string) }); ok {
		hv.SetName(C.GoString(value))
	}
}

// SetOwner sets owner value on target
//
//export SetOwner
func SetOwner(target C.uintptr_t, value *C.cchar_t) {
	h := cgo.Handle(target)
	if hv, ok := h.Value().(interface{ SetOwner(string) }); ok {
		hv.SetOwner(C.GoString(value))
	}
}

// SetPropertyBool sets key:value property on target
//
//export SetPropertyBool
func SetPropertyBool(target C.uintptr_t, key *C.cchar_t, value C.bool) {
	h := cgo.Handle(target)
	if hv, ok := h.Value().(interface{ SetProperty(string, interface{}) }); ok {
		hv.SetProperty(C.GoString(key), bool(value))
	}
}

// SetPropertyDouble sets key:value property on target
//
//export SetPropertyDouble
func SetPropertyDouble(target C.uintptr_t, key *C.cchar_t, value C.double) {
	h := cgo.Handle(target)
	if hv, ok := h.Value().(interface{ SetProperty(string, interface{}) }); ok {
		hv.SetProperty(C.GoString(key), float64(value))
	}
}

// SetPropertyInt sets key:value property on target
//
//export SetPropertyInt
func SetPropertyInt(target C.uintptr_t, key *C.cchar_t, value C.longlong) {
	h := cgo.Handle(target)
	if hv, ok := h.Value().(interface{ SetProperty(string, interface{}) }); ok {
		hv.SetProperty(C.GoString(key), int64(value))
	}
}

// SetPropertyStr sets key:value property on target
//
//export SetPropertyStr
func SetPropertyStr(target C.uintptr_t, key, value *C.cchar_t) {
	h := cgo.Handle(target)
	if hv, ok := h.Value().(interface{ SetProperty(string, interface{}) }); ok {
		hv.SetProperty(C.GoString(key), C.GoString(value))
	}
}

// SetPropertyTime sets key:timestamp property on target
//
//export SetPropertyTime
func SetPropertyTime(target C.uintptr_t, key *C.cchar_t, sec, nsec C.longlong) {
	h := cgo.Handle(target)
	if hv, ok := h.Value().(interface{ SetProperty(string, interface{}) }); ok {
		hv.SetProperty(C.GoString(key),
			transit.Timestamp{Time: time.Unix(int64(sec), int64(nsec)).UTC()})
	}
}

// SetSampleType sets status value on target
//
//export SetSampleType
func SetSampleType(target C.uintptr_t, value *C.cchar_t) {
	h := cgo.Handle(target)
	if hv, ok := h.Value().(interface {
		SetSampleType(transit.MetricSampleType)
	}); ok {
		hv.SetSampleType(transit.MetricSampleType(C.GoString(value)))
	}
}

// SetStatus sets status value on target
//
//export SetStatus
func SetStatus(target C.uintptr_t, value *C.cchar_t) {
	h := cgo.Handle(target)
	if hv, ok := h.Value().(interface{ SetStatus(transit.MonitorStatus) }); ok {
		hv.SetStatus(transit.MonitorStatus(C.GoString(value)))
	}
}

// SetTag sets key:value tag on target
//
//export SetTag
func SetTag(target C.uintptr_t, key, value *C.cchar_t) {
	h := cgo.Handle(target)
	if hv, ok := h.Value().(interface{ SetTag(string, string) }); ok {
		hv.SetTag(C.GoString(key), C.GoString(value))
	}
}

// SetType sets type value on target
//
//export SetType
func SetType(target C.uintptr_t, value *C.cchar_t) {
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
//
//export SetUnit
func SetUnit(target C.uintptr_t, value *C.cchar_t) {
	h := cgo.Handle(target)
	if hv, ok := h.Value().(interface{ SetUnit(transit.UnitType) }); ok {
		hv.SetUnit(transit.UnitType(C.GoString(value)))
	}
}

// SetValueBool sets value to target
//
//export SetValueBool
func SetValueBool(target C.uintptr_t, value C.bool) {
	h := cgo.Handle(target)
	if hv, ok := h.Value().(interface{ SetValue(interface{}) }); ok {
		hv.SetValue(bool(value))
	}
}

// SetValueDouble sets value to target
//
//export SetValueDouble
func SetValueDouble(target C.uintptr_t, value C.double) {
	h := cgo.Handle(target)
	if hv, ok := h.Value().(interface{ SetValue(interface{}) }); ok {
		hv.SetValue(float64(value))
	}
}

// SetValueInt sets value to target
//
//export SetValueInt
func SetValueInt(target C.uintptr_t, value C.longlong) {
	h := cgo.Handle(target)
	if hv, ok := h.Value().(interface{ SetValue(interface{}) }); ok {
		hv.SetValue(int64(value))
	}
}

// SetValueStr sets value to target
//
//export SetValueStr
func SetValueStr(target C.uintptr_t, value *C.cchar_t) {
	h := cgo.Handle(target)
	if hv, ok := h.Value().(interface{ SetValue(interface{}) }); ok {
		hv.SetValue(C.GoString(value))
	}
}

// SetValueTime sets timestamp value to target
//
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
//
//export MarshallIndentJSON
func MarshallIndentJSON(
	target C.uintptr_t,
	prefix, indent *C.cchar_t,
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

// Send sends request
//
//export Send
func Send(req C.uintptr_t, errBuf *C.char, errBufLen C.size_t) C.bool {
	var sender func(context.Context, []byte) error

	h := cgo.Handle(req)
	switch h.Value().(type) {
	case *transit.Downtimes:
		sender = services.GetTransitService().ClearInDowntime
	case *transit.DowntimesRequest:
		sender = services.GetTransitService().SetInDowntime
	case *transit.GroundworkEventsRequest:
		sender = services.GetTransitService().SendEvents
	case *transit.GroundworkEventsAckRequest:
		sender = services.GetTransitService().SendEventsAck
	case *transit.GroundworkEventsUnackRequest:
		sender = services.GetTransitService().SendEventsUnack
	case *transit.InventoryRequest:
		sender = services.GetTransitService().SynchronizeInventory
	case *transit.ResourcesWithServicesRequest:
		sender = services.GetTransitService().SendResourceWithMetrics
	default:
		msg := fmt.Sprintf("unknown type: %+v", h.Value())
		bufStr(errBuf, errBufLen, msg)
		log.Warn().Msg(msg)
		return false
	}

	bb, err := json.Marshal(h.Value())
	log.Trace().Err(err).RawJSON("payload", bb).Msg("Send")
	if err != nil {
		bufStr(errBuf, errBufLen, err.Error())
		return false
	}
	if err := sender(context.Background(), bb); err != nil {
		bufStr(errBuf, errBufLen, err.Error())
		return false
	}
	return true
}

// SyncExt processes extended inventory included additional properties
//
//export SyncExt
func SyncExt(p C.uintptr_t, errBuf *C.char, errBufLen C.size_t) C.bool {
	var inv *transit.InventoryRequest
	h := cgo.Handle(p)
	if v, ok := h.Value().(*transit.InventoryRequest); ok {
		inv = v
	} else {
		msg := fmt.Sprintf("unexpected type: %+v", h.Value())
		bufStr(errBuf, errBufLen, msg)
		log.Warn().Msg(msg)
		return false
	}

	if err := services.GetTransitService().SyncExt(context.Background(), inv); err != nil {
		bufStr(errBuf, errBufLen, err.Error())
		return false
	}
	return true
}
