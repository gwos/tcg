package main

import (
	"testing"
)

/*
The actual test functions are in non-_test.go files so that they can use cgo (import "C").
These wrappers are here for gotest to find.

https://stackoverflow.com/questions/27930737/import-c-is-unsupported-in-test-looking-for-alternatives
*/

func TestSetCategory(t *testing.T) { testSetCategory(t) }

func TestSetContextTimestamp(t *testing.T) { testSetContextTimestamp(t) }

func TestSetContextToken(t *testing.T) { testSetContextToken(t) }

func TestSetDescription(t *testing.T) { testSetDescription(t) }

func TestSetDevice(t *testing.T) { testSetDevice(t) }

func TestSetIntervalEnd(t *testing.T) { testSetIntervalEnd(t) }

func TestSetIntervalStart(t *testing.T) { testSetIntervalStart(t) }

func TestSetLastPluginOutput(t *testing.T) { testSetLastPluginOutput(t) }

func TestSetLastCheckTime(t *testing.T) { testSetLastCheckTime(t) }

func TestSetNextCheckTime(t *testing.T) { testSetNextCheckTime(t) }

func TestSetName(t *testing.T) { testSetName(t) }

func TestSetOwner(t *testing.T) { testSetOwner(t) }

func TestSetPropertyBool(t *testing.T) { testSetPropertyBool(t) }

func TestSetPropertyDouble(t *testing.T) { testSetPropertyDouble(t) }

func TestSetPropertyInt(t *testing.T) { testSetPropertyInt(t) }

func TestSetPropertyStr(t *testing.T) { testSetPropertyStr(t) }

func TestSetPropertyTime(t *testing.T) { testSetPropertyTime(t) }

func TestSetSampleType(t *testing.T) { testSetSampleType(t) }

func TestSetStatus(t *testing.T) { testSetStatus(t) }

func TestSetTag(t *testing.T) { testSetTag(t) }

func TestSetType(t *testing.T) { testSetType(t) }

func TestSetUnit(t *testing.T) { testSetUnit(t) }

func TestSetValueBool(t *testing.T) { testSetValueBool(t) }

func TestSetValueDouble(t *testing.T) { testSetValueDouble(t) }

func TestSetValueInt(t *testing.T) { testSetValueInt(t) }

func TestSetValueStr(t *testing.T) { testSetValueStr(t) }

func TestSetValueTime(t *testing.T) { testSetValueTime(t) }
