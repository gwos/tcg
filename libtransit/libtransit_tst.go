package main

import "C"
import (
	"fmt"
	"reflect"
	"runtime/cgo"
	"testing"

	"github.com/gwos/tcg/sdk/transit"
	"github.com/stretchr/testify/assert"
)

func test_SetCategory(t *testing.T) {
	value := "test-test"
	tests := []struct {
		name   string
		target interface{}
		field  string
	}{{
		name:   "InventoryResource",
		target: new(transit.InventoryResource),
		field:  "Category",
	}, {
		name:   "InventoryService",
		target: new(transit.InventoryService),
		field:  "Category",
	}, {
		name:   "MonitoredResource",
		target: new(transit.MonitoredResource),
		field:  "Category",
	}, {
		name:   "MonitoredService",
		target: new(transit.MonitoredService),
		field:  "Category",
	}}
	for _, it := range tests {
		t.Run(it.name, func(t *testing.T) {
			h := cgo.NewHandle(it.target)
			SetCategory(C.ulong(h), C.CString(value))
			h.Delete()
			r := reflect.ValueOf(it.target)
			f := reflect.Indirect(r).FieldByName(it.field)
			assert.Equal(t, value, f.String())
		})
	}
}

func test_SetDescription(t *testing.T) {
	value := "test-test"
	tests := []struct {
		name   string
		target interface{}
		field  string
	}{{
		name:   "InventoryResource",
		target: new(transit.InventoryResource),
		field:  "Description",
	}, {
		name:   "InventoryService",
		target: new(transit.InventoryService),
		field:  "Description",
	}, {
		name:   "MonitoredResource",
		target: new(transit.MonitoredResource),
		field:  "Description",
	}, {
		name:   "MonitoredService",
		target: new(transit.MonitoredService),
		field:  "Description",
	}, {
		name:   "ResourceGroup",
		target: new(transit.ResourceGroup),
		field:  "Description",
	}}
	for _, it := range tests {
		t.Run(it.name, func(t *testing.T) {
			h := cgo.NewHandle(it.target)
			SetDescription(C.ulong(h), C.CString(value))
			h.Delete()
			r := reflect.ValueOf(it.target)
			f := reflect.Indirect(r).FieldByName(it.field)
			assert.Equal(t, value, f.String())
		})
	}
}

func test_SetDevice(t *testing.T) {
	value := "test-test"
	tests := []struct {
		name   string
		target interface{}
		field  string
	}{{
		name:   "InventoryResource",
		target: new(transit.InventoryResource),
		field:  "Device",
	}, {
		name:   "MonitoredResource",
		target: new(transit.MonitoredResource),
		field:  "Device",
	}}
	for _, it := range tests {
		t.Run(it.name, func(t *testing.T) {
			h := cgo.NewHandle(it.target)
			SetDevice(C.ulong(h), C.CString(value))
			h.Delete()
			r := reflect.ValueOf(it.target)
			f := reflect.Indirect(r).FieldByName(it.field)
			assert.Equal(t, value, f.String())
		})
	}
}

func test_SetIntervalEnd(t *testing.T) {
	v, v1, v2 := "1609372800000", 1609372800, 0
	tests := []struct {
		name   string
		target interface{}
		field  string
	}{{
		name:   "TimeSeries",
		target: new(transit.TimeSeries),
		field:  "Interval",
	}}
	for _, it := range tests {
		t.Run(it.name, func(t *testing.T) {
			h := cgo.NewHandle(it.target)
			SetIntervalEnd(C.ulong(h), C.longlong(v1), C.longlong(v2))
			h.Delete()
			r := reflect.ValueOf(it.target)
			f := reflect.Indirect(r).FieldByName(it.field)
			assert.Equal(t, v, f.Interface().(*transit.TimeInterval).EndTime.String())
		})
	}
}

func test_SetIntervalStart(t *testing.T) {
	v, v1, v2 := "1609372800000", 1609372800, 0
	tests := []struct {
		name   string
		target interface{}
		field  string
	}{{
		name:   "TimeSeries",
		target: new(transit.TimeSeries),
		field:  "Interval",
	}}
	for _, it := range tests {
		t.Run(it.name, func(t *testing.T) {
			h := cgo.NewHandle(it.target)
			SetIntervalStart(C.ulong(h), C.longlong(v1), C.longlong(v2))
			h.Delete()
			r := reflect.ValueOf(it.target)
			f := reflect.Indirect(r).FieldByName(it.field)
			assert.Equal(t, v, f.Interface().(*transit.TimeInterval).StartTime.String())
		})
	}
}

func test_SetLastPluginOutput(t *testing.T) {
	value := "test-test"
	tests := []struct {
		name   string
		target interface{}
		field  string
	}{{
		name:   "MonitoredResource",
		target: new(transit.MonitoredResource),
		field:  "LastPluginOutput",
	}, {
		name:   "MonitoredService",
		target: new(transit.MonitoredService),
		field:  "LastPluginOutput",
	}}
	for _, it := range tests {
		t.Run(it.name, func(t *testing.T) {
			h := cgo.NewHandle(it.target)
			SetLastPluginOutput(C.ulong(h), C.CString(value))
			h.Delete()
			r := reflect.ValueOf(it.target)
			f := reflect.Indirect(r).FieldByName(it.field)
			assert.Equal(t, value, f.String())
		})
	}
}

func test_SetLastCheckTime(t *testing.T) {
	v, v1, v2 := "1609372800000", 1609372800, 0
	tests := []struct {
		name   string
		target interface{}
		field  string
	}{{
		name:   "MonitoredResource",
		target: new(transit.MonitoredResource),
		field:  "LastCheckTime",
	}, {
		name:   "MonitoredService",
		target: new(transit.MonitoredService),
		field:  "LastCheckTime",
	}}
	for _, it := range tests {
		t.Run(it.name, func(t *testing.T) {
			h := cgo.NewHandle(it.target)
			SetLastCheckTime(C.ulong(h), C.longlong(v1), C.longlong(v2))
			h.Delete()
			r := reflect.ValueOf(it.target)
			f := reflect.Indirect(r).FieldByName(it.field)
			assert.Equal(t, v, f.Interface().(*transit.Timestamp).String())
		})
	}
}

func test_SetNextCheckTime(t *testing.T) {
	v, v1, v2 := "1609372800000", 1609372800, 0
	tests := []struct {
		name   string
		target interface{}
		field  string
	}{{
		name:   "MonitoredResource",
		target: new(transit.MonitoredResource),
		field:  "NextCheckTime",
	}, {
		name:   "MonitoredService",
		target: new(transit.MonitoredService),
		field:  "NextCheckTime",
	}}
	for _, it := range tests {
		t.Run(it.name, func(t *testing.T) {
			h := cgo.NewHandle(it.target)
			SetNextCheckTime(C.ulong(h), C.longlong(v1), C.longlong(v2))
			h.Delete()
			r := reflect.ValueOf(it.target)
			f := reflect.Indirect(r).FieldByName(it.field)
			assert.Equal(t, v, f.Interface().(*transit.Timestamp).String())
		})
	}
}

func test_SetName(t *testing.T) {
	value := "test-test"
	tests := []struct {
		name   string
		target interface{}
		field  string
	}{{
		name:   "InventoryResource",
		target: new(transit.InventoryResource),
		field:  "Name",
	}, {
		name:   "InventoryService",
		target: new(transit.InventoryService),
		field:  "Name",
	}, {
		name:   "MonitoredResource",
		target: new(transit.MonitoredResource),
		field:  "Name",
	}, {
		name:   "MonitoredService",
		target: new(transit.MonitoredService),
		field:  "Name",
	}, {
		name:   "ResourceGroup",
		target: new(transit.ResourceGroup),
		field:  "GroupName",
	}, {
		name:   "TimeSeries",
		target: new(transit.TimeSeries),
		field:  "MetricName",
	}}
	for _, it := range tests {
		t.Run(it.name, func(t *testing.T) {
			h := cgo.NewHandle(it.target)
			SetName(C.ulong(h), C.CString(value))
			h.Delete()
			r := reflect.ValueOf(it.target)
			f := reflect.Indirect(r).FieldByName(it.field)
			assert.Equal(t, value, f.String())
		})
	}
}

func test_SetOwner(t *testing.T) {
	value := "test-test"
	tests := []struct {
		name   string
		target interface{}
		field  string
	}{{
		name:   "InventoryResource",
		target: new(transit.InventoryResource),
		field:  "Owner",
	}, {
		name:   "InventoryService",
		target: new(transit.InventoryService),
		field:  "Owner",
	}, {
		name:   "MonitoredResource",
		target: new(transit.MonitoredResource),
		field:  "Owner",
	}, {
		name:   "MonitoredService",
		target: new(transit.MonitoredService),
		field:  "Owner",
	}}
	for _, it := range tests {
		t.Run(it.name, func(t *testing.T) {
			h := cgo.NewHandle(it.target)
			SetOwner(C.ulong(h), C.CString(value))
			h.Delete()
			r := reflect.ValueOf(it.target)
			f := reflect.Indirect(r).FieldByName(it.field)
			assert.Equal(t, value, f.String())
		})
	}
}

func test_SetPropertyBool(t *testing.T) {
	key, value := "test-test", false
	tests := []struct {
		name   string
		target interface{}
		field  string
	}{{
		name:   "InventoryResource",
		target: new(transit.InventoryResource),
		field:  "Properties",
	}, {
		name:   "InventoryService",
		target: new(transit.InventoryService),
		field:  "Properties",
	}, {
		name:   "MonitoredResource",
		target: new(transit.MonitoredResource),
		field:  "Properties",
	}, {
		name:   "MonitoredService",
		target: new(transit.MonitoredService),
		field:  "Properties",
	}}
	for _, it := range tests {
		t.Run(it.name, func(t *testing.T) {
			h := cgo.NewHandle(it.target)
			SetPropertyBool(C.ulong(h), C.CString(key), C._Bool(value))
			r := reflect.ValueOf(it.target)
			f := reflect.Indirect(r).FieldByName(it.field)
			assert.Equal(t, value, *f.Interface().(map[string]transit.TypedValue)[key].BoolValue)
			SetPropertyBool(C.ulong(h), C.CString(key), C._Bool(!value))
			f = reflect.Indirect(r).FieldByName(it.field)
			assert.Equal(t, transit.BooleanType, f.Interface().(map[string]transit.TypedValue)[key].ValueType)
			assert.Equal(t, !value, *f.Interface().(map[string]transit.TypedValue)[key].BoolValue)
			h.Delete()
		})
	}
}

func test_SetPropertyDouble(t *testing.T) {
	key, value := "test-test", -1.1
	tests := []struct {
		name   string
		target interface{}
		field  string
	}{{
		name:   "InventoryResource",
		target: new(transit.InventoryResource),
		field:  "Properties",
	}, {
		name:   "InventoryService",
		target: new(transit.InventoryService),
		field:  "Properties",
	}, {
		name:   "MonitoredResource",
		target: new(transit.MonitoredResource),
		field:  "Properties",
	}, {
		name:   "MonitoredService",
		target: new(transit.MonitoredService),
		field:  "Properties",
	}}
	for _, it := range tests {
		t.Run(it.name, func(t *testing.T) {
			h := cgo.NewHandle(it.target)
			SetPropertyDouble(C.ulong(h), C.CString(key), C.double(value))
			h.Delete()
			r := reflect.ValueOf(it.target)
			f := reflect.Indirect(r).FieldByName(it.field)
			assert.Equal(t, transit.DoubleType, f.Interface().(map[string]transit.TypedValue)[key].ValueType)
			assert.Equal(t, value, *f.Interface().(map[string]transit.TypedValue)[key].DoubleValue)
		})
	}
}

func test_SetPropertyInt(t *testing.T) {
	key, value := "test-test", int64(42)
	tests := []struct {
		name   string
		target interface{}
		field  string
	}{{
		name:   "InventoryResource",
		target: new(transit.InventoryResource),
		field:  "Properties",
	}, {
		name:   "InventoryService",
		target: new(transit.InventoryService),
		field:  "Properties",
	}, {
		name:   "MonitoredResource",
		target: new(transit.MonitoredResource),
		field:  "Properties",
	}, {
		name:   "MonitoredService",
		target: new(transit.MonitoredService),
		field:  "Properties",
	}}
	for _, it := range tests {
		t.Run(it.name, func(t *testing.T) {
			h := cgo.NewHandle(it.target)
			SetPropertyInt(C.ulong(h), C.CString(key), C.longlong(value))
			h.Delete()
			r := reflect.ValueOf(it.target)
			f := reflect.Indirect(r).FieldByName(it.field)
			assert.Equal(t, transit.IntegerType, f.Interface().(map[string]transit.TypedValue)[key].ValueType)
			assert.Equal(t, value, *f.Interface().(map[string]transit.TypedValue)[key].IntegerValue)
		})
	}
}

func test_SetPropertyStr(t *testing.T) {
	key, value := "test-test", "foo-bar"
	tests := []struct {
		name   string
		target interface{}
		field  string
	}{{
		name:   "InventoryResource",
		target: new(transit.InventoryResource),
		field:  "Properties",
	}, {
		name:   "InventoryService",
		target: new(transit.InventoryService),
		field:  "Properties",
	}, {
		name:   "MonitoredResource",
		target: new(transit.MonitoredResource),
		field:  "Properties",
	}, {
		name:   "MonitoredService",
		target: new(transit.MonitoredService),
		field:  "Properties",
	}}
	for _, it := range tests {
		t.Run(it.name, func(t *testing.T) {
			h := cgo.NewHandle(it.target)
			SetPropertyStr(C.ulong(h), C.CString(key), C.CString(value))
			h.Delete()
			r := reflect.ValueOf(it.target)
			f := reflect.Indirect(r).FieldByName(it.field)
			assert.Equal(t, transit.StringType, f.Interface().(map[string]transit.TypedValue)[key].ValueType)
			assert.Equal(t, value, *f.Interface().(map[string]transit.TypedValue)[key].StringValue)
		})
	}
}

func test_SetPropertyTime(t *testing.T) {
	key, v, v1, v2 := "test-test", "1609372800000", 1609372800, 0
	tests := []struct {
		name   string
		target interface{}
		field  string
	}{{
		name:   "InventoryResource",
		target: new(transit.InventoryResource),
		field:  "Properties",
	}, {
		name:   "InventoryService",
		target: new(transit.InventoryService),
		field:  "Properties",
	}, {
		name:   "MonitoredResource",
		target: new(transit.MonitoredResource),
		field:  "Properties",
	}, {
		name:   "MonitoredService",
		target: new(transit.MonitoredService),
		field:  "Properties",
	}}
	for _, it := range tests {
		t.Run(it.name, func(t *testing.T) {
			h := cgo.NewHandle(it.target)
			SetPropertyTime(C.ulong(h), C.CString(key), C.longlong(v1), C.longlong(v2))
			h.Delete()
			r := reflect.ValueOf(it.target)
			f := reflect.Indirect(r).FieldByName(it.field)
			assert.Equal(t, transit.TimeType, f.Interface().(map[string]transit.TypedValue)[key].ValueType)
			assert.Equal(t, v, f.Interface().(map[string]transit.TypedValue)[key].TimeValue.String())
		})
	}
}

func test_SetSampleType(t *testing.T) {
	value := transit.Value
	tests := []struct {
		name   string
		target interface{}
		field  string
	}{{
		name:   "TimeSeries",
		target: new(transit.TimeSeries),
		field:  "SampleType",
	}}
	for _, it := range tests {
		t.Run(it.name, func(t *testing.T) {
			h := cgo.NewHandle(it.target)
			SetSampleType(C.ulong(h), C.CString(string(value)))
			h.Delete()
			r := reflect.ValueOf(it.target)
			f := reflect.Indirect(r).FieldByName(it.field)
			assert.Equal(t, string(value), f.String())
		})
	}
}

func test_SetStatus(t *testing.T) {
	tests := []struct {
		name   string
		target interface{}
		field  string
		value  interface{}
	}{{
		name:   "MonitoredResource",
		target: new(transit.MonitoredResource),
		field:  "Status",
		value:  transit.HostUp,
	}, {
		name:   "MonitoredService",
		target: new(transit.MonitoredService),
		field:  "Status",
		value:  transit.ServiceOk,
	}}
	for _, it := range tests {
		t.Run(it.name, func(t *testing.T) {
			h := cgo.NewHandle(it.target)
			SetStatus(C.ulong(h), C.CString(fmt.Sprint(it.value)))
			h.Delete()
			r := reflect.ValueOf(it.target)
			f := reflect.Indirect(r).FieldByName(it.field)
			assert.Equal(t, fmt.Sprint(it.value), f.String())
		})
	}
}

func test_SetTag(t *testing.T) {
	key, value := "test-test", "foo-bar"
	tests := []struct {
		name   string
		target interface{}
		field  string
	}{{
		name:   "TimeSeries",
		target: new(transit.TimeSeries),
		field:  "Tags",
	}}
	for _, it := range tests {
		t.Run(it.name, func(t *testing.T) {
			h := cgo.NewHandle(it.target)
			SetTag(C.ulong(h), C.CString(key), C.CString(value))
			h.Delete()
			r := reflect.ValueOf(it.target)
			f := reflect.Indirect(r).FieldByName(it.field)
			assert.Equal(t, value, f.Interface().(map[string]string)[key])
		})
	}
}

func test_SetType(t *testing.T) {
	tests := []struct {
		name   string
		target interface{}
		field  string
		value  interface{}
	}{{
		name:   "InventoryResource",
		target: new(transit.InventoryResource),
		field:  "Type",
		value:  transit.ResourceTypeHost,
	}, {
		name:   "InventoryService",
		target: new(transit.InventoryService),
		field:  "Type",
		value:  transit.ResourceTypeService,
	}, {
		name:   "MonitoredResource",
		target: new(transit.MonitoredResource),
		field:  "Type",
		value:  transit.ResourceTypeHypervisor,
	}, {
		name:   "MonitoredService",
		target: new(transit.MonitoredService),
		field:  "Type",
		value:  transit.ResourceTypeInstance,
	}, {
		name:   "ResourceGroup",
		target: new(transit.ResourceGroup),
		field:  "Type",
		value:  transit.HostGroup,
	}}
	for _, it := range tests {
		t.Run(it.name, func(t *testing.T) {
			h := cgo.NewHandle(it.target)
			SetType(C.ulong(h), C.CString(fmt.Sprint(it.value)))
			h.Delete()
			r := reflect.ValueOf(it.target)
			f := reflect.Indirect(r).FieldByName(it.field)
			assert.Equal(t, fmt.Sprint(it.value), f.String())
		})
	}
}

func test_SetUnit(t *testing.T) {
	value := transit.UnitCounter
	tests := []struct {
		name   string
		target interface{}
		field  string
	}{{
		name:   "TimeSeries",
		target: new(transit.TimeSeries),
		field:  "Unit",
	}}
	for _, it := range tests {
		t.Run(it.name, func(t *testing.T) {
			h := cgo.NewHandle(it.target)
			SetUnit(C.ulong(h), C.CString(string(value)))
			h.Delete()
			r := reflect.ValueOf(it.target)
			f := reflect.Indirect(r).FieldByName(it.field)
			assert.Equal(t, string(value), f.String())
		})
	}
}

func test_SetValueBool(t *testing.T) {
	value := false
	tests := []struct {
		name   string
		target interface{}
		field  string
	}{{
		name:   "ThresholdValue",
		target: new(transit.ThresholdValue),
		field:  "Value",
	}, {
		name:   "TimeSeries",
		target: new(transit.TimeSeries),
		field:  "Value",
	}}
	for _, it := range tests {
		t.Run(it.name, func(t *testing.T) {
			h := cgo.NewHandle(it.target)
			SetValueBool(C.ulong(h), C._Bool(value))
			r := reflect.ValueOf(it.target)
			f := reflect.Indirect(r).FieldByName(it.field)
			assert.Equal(t, value, *f.Interface().(*transit.TypedValue).BoolValue)
			SetValueBool(C.ulong(h), C._Bool(!value))
			f = reflect.Indirect(r).FieldByName(it.field)
			assert.Equal(t, transit.BooleanType, f.Interface().(*transit.TypedValue).ValueType)
			assert.Equal(t, !value, *f.Interface().(*transit.TypedValue).BoolValue)
			h.Delete()
		})
	}
}

func test_SetValueDouble(t *testing.T) {
	value := -1.1
	tests := []struct {
		name   string
		target interface{}
		field  string
	}{{
		name:   "ThresholdValue",
		target: new(transit.ThresholdValue),
		field:  "Value",
	}, {
		name:   "TimeSeries",
		target: new(transit.TimeSeries),
		field:  "Value",
	}}
	for _, it := range tests {
		t.Run(it.name, func(t *testing.T) {
			h := cgo.NewHandle(it.target)
			SetValueDouble(C.ulong(h), C.double(value))
			h.Delete()
			r := reflect.ValueOf(it.target)
			f := reflect.Indirect(r).FieldByName(it.field)
			assert.Equal(t, transit.DoubleType, f.Interface().(*transit.TypedValue).ValueType)
			assert.Equal(t, value, *f.Interface().(*transit.TypedValue).DoubleValue)
		})
	}
}

func test_SetValueInt(t *testing.T) {
	value := int64(42)
	tests := []struct {
		name   string
		target interface{}
		field  string
	}{{
		name:   "ThresholdValue",
		target: new(transit.ThresholdValue),
		field:  "Value",
	}, {
		name:   "TimeSeries",
		target: new(transit.TimeSeries),
		field:  "Value",
	}}
	for _, it := range tests {
		t.Run(it.name, func(t *testing.T) {
			h := cgo.NewHandle(it.target)
			SetValueInt(C.ulong(h), C.longlong(value))
			h.Delete()
			r := reflect.ValueOf(it.target)
			f := reflect.Indirect(r).FieldByName(it.field)
			assert.Equal(t, transit.IntegerType, f.Interface().(*transit.TypedValue).ValueType)
			assert.Equal(t, value, *f.Interface().(*transit.TypedValue).IntegerValue)
		})
	}
}

func test_SetValueStr(t *testing.T) {
	value := "foo-bar"
	tests := []struct {
		name   string
		target interface{}
		field  string
	}{{
		name:   "ThresholdValue",
		target: new(transit.ThresholdValue),
		field:  "Value",
	}, {
		name:   "TimeSeries",
		target: new(transit.TimeSeries),
		field:  "Value",
	}}
	for _, it := range tests {
		t.Run(it.name, func(t *testing.T) {
			h := cgo.NewHandle(it.target)
			SetValueStr(C.ulong(h), C.CString(value))
			h.Delete()
			r := reflect.ValueOf(it.target)
			f := reflect.Indirect(r).FieldByName(it.field)
			assert.Equal(t, transit.StringType, f.Interface().(*transit.TypedValue).ValueType)
			assert.Equal(t, value, *f.Interface().(*transit.TypedValue).StringValue)
		})
	}
}

func test_SetValueTime(t *testing.T) {
	v, v1, v2 := "1609372800000", 1609372800, 0
	tests := []struct {
		name   string
		target interface{}
		field  string
	}{{
		name:   "ThresholdValue",
		target: new(transit.ThresholdValue),
		field:  "Value",
	}, {
		name:   "TimeSeries",
		target: new(transit.TimeSeries),
		field:  "Value",
	}}
	for _, it := range tests {
		t.Run(it.name, func(t *testing.T) {
			h := cgo.NewHandle(it.target)
			SetValueTime(C.ulong(h), C.longlong(v1), C.longlong(v2))
			h.Delete()
			r := reflect.ValueOf(it.target)
			f := reflect.Indirect(r).FieldByName(it.field)
			assert.Equal(t, transit.TimeType, f.Interface().(*transit.TypedValue).ValueType)
			assert.Equal(t, v, f.Interface().(*transit.TypedValue).TimeValue.String())
		})
	}
}
