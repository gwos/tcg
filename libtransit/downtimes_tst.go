package main

import "C"
import (
	"runtime/cgo"
	"testing"

	"github.com/gwos/tcg/sdk/transit"
	"github.com/stretchr/testify/assert"
)

func test_AddDowntime(t *testing.T) {
	ptr := func(s string) *C.char { return C.CString(s) }
	tests := []struct {
		name     string
		target   interface{}
		values   [][4]string
		expected *transit.Downtimes
	}{{
		name:   "Host",
		target: &transit.Downtimes{[]transit.Downtime{}},
		values: [][4]string{
			{"HOST", "HOST", "host-1", ""},
		},
		expected: &transit.Downtimes{BizHostServiceInDowntimes: []transit.Downtime{{
			EntityType:             "HOST",
			EntityName:             "HOST",
			HostName:               "host-1",
			ServiceDescription:     "",
			ScheduledDowntimeDepth: 1,
		}}},
	}, {
		name:   "Service",
		target: &transit.Downtimes{[]transit.Downtime{}},
		values: [][4]string{
			{"SERVICE", "SERVICE", "host-1", "service-1"},
			{"SERVICE", "SERVICE", "host-2", "service-2"},
		},
		expected: &transit.Downtimes{BizHostServiceInDowntimes: []transit.Downtime{{
			EntityType:             "SERVICE",
			EntityName:             "SERVICE",
			HostName:               "host-1",
			ServiceDescription:     "service-1",
			ScheduledDowntimeDepth: 1,
		}, {
			EntityType:             "SERVICE",
			EntityName:             "SERVICE",
			HostName:               "host-2",
			ServiceDescription:     "service-2",
			ScheduledDowntimeDepth: 1,
		}}},
	}}
	for _, it := range tests {
		t.Run(it.name, func(t *testing.T) {
			h := cgo.NewHandle(it.target)
			for i := range it.values {
				AddDowntime(C.ulong(h),
					ptr(it.values[i][0]),
					ptr(it.values[i][1]),
					ptr(it.values[i][2]),
					ptr(it.values[i][3]),
					1)
			}
			h.Delete()
			assert.Equal(t, it.expected, it.target)
		})
	}
}

func test_ExtendDowntimesRequest(t *testing.T) {
	ptr := func(s string) *C.char {
		if s == "" {
			return (*C.char)(nil)
		}
		return C.CString(s)
	}
	tests := []struct {
		name     string
		target   interface{}
		values   [][4]string
		expected *transit.DowntimesRequest
	}{{
		name:   "Empty",
		target: &transit.DowntimesRequest{[]string{}, []string{}, []string{}, []string{}, false, false},
		values: [][4]string{
			{"host-1", "host-group-1", "", ""},
			{"host-2", "host-group-2", "", ""},
		},
		expected: &transit.DowntimesRequest{
			HostNames:                 []string{"host-1", "host-2"},
			HostGroupNames:            []string{"host-group-1", "host-group-2"},
			ServiceDescriptions:       []string{},
			ServiceGroupCategoryNames: []string{},
			SetHosts:                  true,
			SetServices:               false,
		},
	}, {
		name:   "Notempty",
		target: &transit.DowntimesRequest{[]string{}, []string{}, []string{}, []string{}, false, false},
		values: [][4]string{
			{"host-1", "host-group-1", "service-1", "service-group-1"},
			{"host-2", "host-group-2", "service-2", "service-group-2"},
		},
		expected: &transit.DowntimesRequest{
			HostNames:                 []string{"host-1", "host-2"},
			HostGroupNames:            []string{"host-group-1", "host-group-2"},
			ServiceDescriptions:       []string{"service-1", "service-2"},
			ServiceGroupCategoryNames: []string{"service-group-1", "service-group-2"},
			SetHosts:                  true,
			SetServices:               true,
		},
	}}
	for _, it := range tests {
		t.Run(it.name, func(t *testing.T) {
			h := cgo.NewHandle(it.target)
			for i := range it.values {
				ExtendDowntimesRequest(C.ulong(h),
					ptr(it.values[i][0]),
					ptr(it.values[i][1]),
					ptr(it.values[i][2]),
					ptr(it.values[i][3]),
				)
			}
			h.Delete()
			assert.Equal(t, it.expected, it.target)
		})
	}
}
