package main

import "C"
import (
	"reflect"
	"runtime/cgo"
	"testing"
	"time"

	"github.com/gwos/tcg/sdk/transit"
	"github.com/stretchr/testify/assert"
)

func test_SetEventAttrs(t *testing.T) {
	ptr := func(s string) *C.char {
		if s == "" {
			return (*C.char)(nil)
		}
		return C.CString(s)
	}
	tests := []struct {
		name   string
		target interface{}
		fields []string
		values []string
	}{{
		name:   "Notempty",
		target: new(transit.GroundworkEvent),
		fields: []string{"AppType", "Host", "MonitorStatus",
			"Device", "Service", "OperationStatus",
			"Severity", "ApplicationSeverity", "Component", "SubComponent",
			"Priority", "TypeRule", "TextMessage",
			"ApplicationName", "ConsolidationName", "ErrorType",
			"LoggerName", "LogType", "MonitorServer"},
		values: []string{"app-type", "host", "monitor-status",
			"device", "service", "operation-status",
			"severity", "application-severity", "component", "sub-component",
			"priority", "type-rule", "text-message",
			"application-name", "consolidation-name", "error-type",
			"logger-name", "log-type", "monitor-server"},
	}, {
		name:   "Mixed",
		target: new(transit.GroundworkEvent),
		fields: []string{"AppType", "Host", "MonitorStatus",
			"Device", "Service", "OperationStatus",
			"Severity", "ApplicationSeverity", "Component", "SubComponent",
			"Priority", "TypeRule", "TextMessage",
			"ApplicationName", "ConsolidationName", "ErrorType",
			"LoggerName", "LogType", "MonitorServer"},
		values: []string{"", "", "",
			"device", "service", "operation-status",
			"", "", "", "",
			"priority", "type-rule", "text-message",
			"", "", "",
			"logger-name", "log-type", "monitor-server"},
	}}
	for _, it := range tests {
		t.Run(it.name, func(t *testing.T) {
			h := cgo.NewHandle(it.target)
			SetEventAttrs(C.ulong(h), ptr(it.values[0]), ptr(it.values[1]), ptr(it.values[2]),
				ptr(it.values[3]), ptr(it.values[4]), ptr(it.values[5]), ptr(it.values[6]),
				ptr(it.values[7]), ptr(it.values[8]), ptr(it.values[9]), ptr(it.values[10]),
				ptr(it.values[11]), ptr(it.values[12]), ptr(it.values[13]), ptr(it.values[14]),
				ptr(it.values[15]), ptr(it.values[16]), ptr(it.values[17]), ptr(it.values[18]),
			)
			h.Delete()
			r := reflect.ValueOf(it.target)
			for i := range it.fields {
				f := reflect.Indirect(r).FieldByName(it.fields[i])
				assert.Equal(t, it.values[i], f.String())
			}
		})
	}
}

func test_SetEventDates(t *testing.T) {
	ptr := func(i int64) *C.longlong { return (*C.longlong)(&i) }
	tests := []struct {
		name     string
		target   interface{}
		fields   []string
		values   [][2]*C.longlong
		expected []*transit.Timestamp
	}{{
		name:     "Empty",
		target:   new(transit.GroundworkEvent),
		fields:   []string{"ReportDate", "LastInsertDate"},
		values:   [][2]*C.longlong{{nil, nil}, {nil, nil}},
		expected: []*transit.Timestamp{nil, nil},
	}, {
		name:   "Notempty",
		target: new(transit.GroundworkEvent),
		fields: []string{"ReportDate", "LastInsertDate"},
		values: [][2]*C.longlong{
			{ptr(1609372800), ptr(0)},
			{ptr(1609372800), ptr(0)},
		},
		expected: []*transit.Timestamp{
			{Time: time.Unix(1609372800, 0).UTC()},
			{Time: time.Unix(1609372800, 0).UTC()},
		},
	}, {
		name:   "Mixed",
		target: new(transit.GroundworkEvent),
		fields: []string{"ReportDate", "LastInsertDate"},
		values: [][2]*C.longlong{
			{ptr(1609372800), nil},
			{nil, nil},
		},
		expected: []*transit.Timestamp{
			{Time: time.Unix(1609372800, 0).UTC()},
			nil,
		},
	}}
	for _, it := range tests {
		t.Run(it.name, func(t *testing.T) {
			h := cgo.NewHandle(it.target)
			SetEventDates(C.ulong(h), it.values[0][0], it.values[0][1], it.values[1][0], it.values[1][1])
			h.Delete()
			r := reflect.ValueOf(it.target)
			for i := range it.fields {
				f := reflect.Indirect(r).FieldByName(it.fields[i])
				assert.Equal(t, it.expected[i], f.Interface().(*transit.Timestamp))
			}
		})
	}
}
