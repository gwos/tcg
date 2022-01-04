package main

import (
	"testing"
)

/*
The actual test functions are in non-_test.go files so that they can use cgo (import "C").
These wrappers are here for gotest to find.

https://stackoverflow.com/questions/27930737/import-c-is-unsupported-in-test-looking-for-alternatives
*/

func Test_SetEventAttrs(t *testing.T) { test_SetEventAttrs(t) }

func Test_SetEventDates(t *testing.T) { test_SetEventDates(t) }
