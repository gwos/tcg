package main

/*
#include <stdbool.h>
#include <stdint.h>
*/
import "C"
import (
	"runtime/cgo"

	"github.com/gwos/tcg/sdk/transit"
)

// CreateDowntimes creates payload for ClearInDowntime API.
// It returns a handle that should be deleted after use with DeleteHandle.
//
//export CreateDowntimes
func CreateDowntimes() C.uintptr_t {
	p := new(transit.Downtimes)
	p.BizHostServiceInDowntimes = []transit.Downtime{}
	return C.uintptr_t(cgo.NewHandle(p))
}

// AddDowntime appends HostServiceInDowntime value to target.
//
//export AddDowntime
func AddDowntime(target C.uintptr_t,
	entityType *C.char,
	entityName *C.char,
	hostName *C.char,
	serviceDesc *C.char,
	scheduledDepth C.int,
) {
	h := cgo.Handle(target)
	if hv, ok := h.Value().(*transit.Downtimes); ok {
		hv.BizHostServiceInDowntimes = append(hv.BizHostServiceInDowntimes,
			transit.Downtime{
				EntityType:             C.GoString(entityType),
				EntityName:             C.GoString(entityName),
				HostName:               C.GoString(hostName),
				ServiceDescription:     C.GoString(serviceDesc),
				ScheduledDowntimeDepth: int(scheduledDepth),
			})
	}
}

// CreateDowntimesRequest creates payload for SetInDowntime API.
// It returns a handle that should be deleted after use with DeleteHandle.
//
//export CreateDowntimesRequest
func CreateDowntimesRequest() C.uintptr_t {
	p := new(transit.DowntimesRequest)
	p.HostNames = []string{}
	p.HostGroupNames = []string{}
	p.ServiceDescriptions = []string{}
	p.ServiceGroupCategoryNames = []string{}
	return C.uintptr_t(cgo.NewHandle(p))
}

// ExtendDowntimesRequest extends target.
// It skips NULL params.
//
//export ExtendDowntimesRequest
func ExtendDowntimesRequest(target C.uintptr_t,
	hostName,
	hostGroup,
	serviceDesc,
	serviceGroup *C.char) {
	h := cgo.Handle(target)
	if hv, ok := h.Value().(*transit.DowntimesRequest); ok {
		if hostName != nil {
			hv.HostNames = append(hv.HostNames, C.GoString(hostName))
			hv.SetHosts = true
		}
		if hostGroup != nil {
			hv.HostGroupNames = append(hv.HostGroupNames, C.GoString(hostGroup))
			hv.SetHosts = true
		}
		if serviceDesc != nil {
			hv.ServiceDescriptions = append(hv.ServiceDescriptions, C.GoString(serviceDesc))
			hv.SetServices = true
		}
		if serviceGroup != nil {
			hv.ServiceGroupCategoryNames = append(hv.ServiceGroupCategoryNames, C.GoString(serviceGroup))
			hv.SetServices = true
		}
	}
}
