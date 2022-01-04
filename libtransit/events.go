package main

/*
#include <stdbool.h>
#include <stdint.h>
*/
import "C"
import (
	"runtime/cgo"
	"time"

	"github.com/gwos/tcg/sdk/transit"
)

// CreateEvent creates an object used in EventsRequest.
// It returns a handle that should be deleted after use with DeleteHandle.
//export CreateEvent
func CreateEvent(
	appType, host, monStatus *C.char,
	sec, nsec C.longlong,
) C.uintptr_t {
	p := new(transit.GroundworkEvent)
	p.AppType = C.GoString(appType)
	p.Host = C.GoString(host)
	p.MonitorStatus = C.GoString(monStatus)
	p.ReportDate = transit.NewTimestamp()
	p.ReportDate.Time = time.Unix(int64(sec), int64(nsec)).UTC()
	return C.uintptr_t(cgo.NewHandle(p))
}

// SetEventAttrs sets attributes on target.
// It skips NULL params.
//export SetEventAttrs
func SetEventAttrs(target C.uintptr_t,
	appType,
	host,
	monStatus,
	// Optional attributes
	device,
	service,
	opStatus,
	severity,
	appSeverity,
	component,
	subComponent,
	priority,
	typeRule,
	textMessage,
	// Update level attributes
	appName,
	consolidationName,
	errorType,
	loggerName,
	logType,
	monServer *C.char,
) {
	attr := func(k *string, v *C.char) {
		if v != nil {
			*k = C.GoString(v)
		}
	}
	h := cgo.Handle(target)
	if hv, ok := h.Value().(*transit.GroundworkEvent); ok {
		attr(&hv.AppType, appType)
		attr(&hv.Host, host)
		attr(&hv.MonitorStatus, monStatus)
		// Optional attributes
		attr(&hv.Device, device)
		attr(&hv.Service, service)
		attr(&hv.OperationStatus, opStatus)
		attr(&hv.Severity, severity)
		attr(&hv.ApplicationSeverity, appSeverity)
		attr(&hv.Component, component)
		attr(&hv.SubComponent, subComponent)
		attr(&hv.Priority, priority)
		attr(&hv.TypeRule, typeRule)
		attr(&hv.TextMessage, textMessage)
		// Update level attributes
		attr(&hv.ApplicationName, appName)
		attr(&hv.ConsolidationName, consolidationName)
		attr(&hv.ErrorType, errorType)
		attr(&hv.LoggerName, loggerName)
		attr(&hv.LogType, logType)
		attr(&hv.MonitorServer, monServer)
	}
}

// SetEventDates sets date attributes on target.
// It skips NULL params.
//export SetEventDates
func SetEventDates(target C.uintptr_t,
	reportDateSec, reportDateNsec,
	lastInsertDateSec, lastInsertDateNsec *C.longlong,
) {
	attr := func(ts **transit.Timestamp, sec, nsec *C.longlong) {
		if sec == nil && nsec == nil {
			return
		}
		s, ns := int64(0), int64(0)
		if sec != nil {
			s = int64(*sec)
		}
		if nsec != nil {
			ns = int64(*nsec)
		}
		t := transit.NewTimestamp()
		t.Time = time.Unix(s, ns).UTC()
		*ts = t
	}
	h := cgo.Handle(target)
	if hv, ok := h.Value().(*transit.GroundworkEvent); ok {
		attr(&hv.ReportDate, reportDateSec, reportDateNsec)
		attr(&hv.LastInsertDate, lastInsertDateSec, lastInsertDateNsec)
	}
}

// CreateEventsRequest creates payload for SendEvents API.
// It returns a handle that should be deleted after use with DeleteHandle.
//export CreateEventsRequest
func CreateEventsRequest() C.uintptr_t {
	p := new(transit.GroundworkEventsRequest)
	p.Events = []transit.GroundworkEvent{}
	return C.uintptr_t(cgo.NewHandle(p))
}

// AddEvent appends Event value to target.
//export AddEvent
func AddEvent(target C.uintptr_t, value C.uintptr_t) {
	h, h2 := cgo.Handle(target), cgo.Handle(value)
	if hv, ok := h.Value().(*transit.GroundworkEventsRequest); ok {
		if hv2, ok := h2.Value().(*transit.GroundworkEvent); ok {
			hv.Events = append(hv.Events, *hv2)
		}
	}
}

// CreateEventsAckRequest creates payload for SendEventsAck API.
// It returns a handle that should be deleted after use with DeleteHandle.
//export CreateEventsAckRequest
func CreateEventsAckRequest() C.uintptr_t {
	p := new(transit.GroundworkEventsAckRequest)
	p.Acks = []transit.GroundworkEventAck{}
	return C.uintptr_t(cgo.NewHandle(p))
}

// AddEventAck appends EventAck value to target.
//export AddEventAck
func AddEventAck(
	target C.uintptr_t,
	appType, host, service, ackBy, ackComment *C.char,
) {
	h := cgo.Handle(target)
	if hv, ok := h.Value().(*transit.GroundworkEventsAckRequest); ok {
		p := new(transit.GroundworkEventAck)
		p.AppType = C.GoString(appType)
		p.Host = C.GoString(host)
		p.Service = C.GoString(service)
		p.AcknowledgedBy = C.GoString(ackBy)
		p.AcknowledgeComment = C.GoString(ackComment)
		hv.Acks = append(hv.Acks, *p)
	}
}

// CreateEventsUnackRequest creates payload for SendEventsUnack API.
// It returns a handle that should be deleted after use with DeleteHandle.
//export CreateEventsUnackRequest
func CreateEventsUnackRequest() C.uintptr_t {
	p := new(transit.GroundworkEventsUnackRequest)
	p.Unacks = []transit.GroundworkEventUnack{}
	return C.uintptr_t(cgo.NewHandle(p))
}

// AddEventUnack appends EventUnack value to target.
//export AddEventUnack
func AddEventUnack(
	target C.uintptr_t,
	appType, host, service *C.char,
) {
	h := cgo.Handle(target)
	if hv, ok := h.Value().(*transit.GroundworkEventsUnackRequest); ok {
		p := new(transit.GroundworkEventUnack)
		p.AppType = C.GoString(appType)
		p.Host = C.GoString(host)
		p.Service = C.GoString(service)
		hv.Unacks = append(hv.Unacks, *p)
	}
}
