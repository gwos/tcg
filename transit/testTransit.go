package main

/*
#include "stdio.h"

typedef struct{
	int Id;
	char *Name;
	char *Type;
	char *Description;
}Metric;
*/
import "C"
import (
	"fmt"
)

func main() {
}

type MetricTest struct {
	Id          int
	Name        string
	Type        string
	Description string
}

//export SendMetricsTest
func SendMetricsTest(cMetric *C.Metric) *C.char {

	metric := MetricTest{}
	metric.toGoStruct(cMetric)

	fmt.Println("Go console:")
	fmt.Printf("Id: %d\nName: %s\nType: %s\nDescription: %s\n", metric.Id, string(metric.Name[:]), string(metric.Type[:]), string(metric.Description[:]))
	fmt.Println()

	return C.CString("Success")
}

//export ListMetricsTest
func ListMetricsTest() C.Metric {
	metric := &MetricTest{
		Id:          1234,
		Name:        "Metric from Go",
		Type:        "local_load_go",
		Description: "Local Load for 1 minute",
	}

	cMetric := C.Metric{}

	metric.toCStruct(&cMetric)

	return cMetric
}

func (metric *MetricTest) toCStruct(cMetric *C.Metric) {
	cMetric.Id = C.int(metric.Id)
	cMetric.Name = C.CString(metric.Name)
	cMetric.Type = C.CString(metric.Type)
	cMetric.Description = C.CString(metric.Description)
}

func (metric *MetricTest) toGoStruct(cMetric *C.Metric) {
	metric.Id = int(cMetric.Id)
	metric.Name = C.GoString(cMetric.Name)
	metric.Type = C.GoString(cMetric.Type)
	metric.Description = C.GoString(cMetric.Description)
}