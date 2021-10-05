package main

import (
	"testing"
)

/*
The actual test functions are in non-_test.go files so that they can use cgo (import "C").
These wrappers are here for gotest to find.

https://stackoverflow.com/questions/27930737/import-c-is-unsupported-in-test-looking-for-alternatives
*/

func Test_SetCategory(t *testing.T) { test_SetCategory(t) }

func Test_SetContextTimestamp(t *testing.T) { test_SetContextTimestamp(t) }

func Test_SetContextToken(t *testing.T) { test_SetContextToken(t) }

func Test_SetDescription(t *testing.T) { test_SetDescription(t) }

func Test_SetDevice(t *testing.T) { test_SetDevice(t) }

func Test_SetIntervalEnd(t *testing.T) { test_SetIntervalEnd(t) }

func Test_SetIntervalStart(t *testing.T) { test_SetIntervalStart(t) }

func Test_SetLastPluginOutput(t *testing.T) { test_SetLastPluginOutput(t) }

func Test_SetLastCheckTime(t *testing.T) { test_SetLastCheckTime(t) }

func Test_SetNextCheckTime(t *testing.T) { test_SetNextCheckTime(t) }

func Test_SetName(t *testing.T) { test_SetName(t) }

func Test_SetOwner(t *testing.T) { test_SetOwner(t) }

func Test_SetPropertyBool(t *testing.T) { test_SetPropertyBool(t) }

func Test_SetPropertyDouble(t *testing.T) { test_SetPropertyDouble(t) }

func Test_SetPropertyInt(t *testing.T) { test_SetPropertyInt(t) }

func Test_SetPropertyStr(t *testing.T) { test_SetPropertyStr(t) }

func Test_SetPropertyTime(t *testing.T) { test_SetPropertyTime(t) }

func Test_SetSampleType(t *testing.T) { test_SetSampleType(t) }

func Test_SetStatus(t *testing.T) { test_SetStatus(t) }

func Test_SetTag(t *testing.T) { test_SetTag(t) }

func Test_SetType(t *testing.T) { test_SetType(t) }

func Test_SetUnit(t *testing.T) { test_SetUnit(t) }

func Test_SetValueBool(t *testing.T) { test_SetValueBool(t) }

func Test_SetValueDouble(t *testing.T) { test_SetValueDouble(t) }

func Test_SetValueInt(t *testing.T) { test_SetValueInt(t) }

func Test_SetValueStr(t *testing.T) { test_SetValueStr(t) }

func Test_SetValueTime(t *testing.T) { test_SetValueTime(t) }
