package main

import (
	"context"
	"log"
	"net/http"
	"sync"
	"time"

	"go.opentelemetry.io/otel/api/global"
	"go.opentelemetry.io/otel/api/kv"
	"go.opentelemetry.io/otel/api/metric"
	"go.opentelemetry.io/otel/exporters/metric/prometheus"
)

var (
	defaultResource = "prometheus-example"
	defaultGroup    = "Prometheus"
	defaultWarning  = "400"
	defaultCritical = "800"
	defaultUnitType = "MB"
)

func initMeter() {
	exporter, err := prometheus.InstallNewPipeline(prometheus.Config{})
	if err != nil {
		log.Panicf("Failed to initialize prometheus exporter %v", err)
	}

	http.HandleFunc("/", exporter.ServeHTTP)

	go func() {
		_ = http.ListenAndServe(":2222", nil)
	}()
}

func main() {
	initMeter()

	meter := global.Meter("groundwork")
	observerLock := new(sync.RWMutex)
	observerValueToReport := new(float64)
	observerLabelsToReport := new([]kv.KeyValue)

	valueRecorder := metric.Must(meter).NewFloat64ValueRecorder("gw.service.one")
	counter := metric.Must(meter).NewFloat64Counter("gw.service.two")

	commonLabels := []kv.KeyValue{
		kv.String("resource", defaultResource),
		kv.String("group", defaultGroup),
		kv.String("warning", defaultWarning),
		kv.String("critical", defaultCritical),
		kv.String("unitType", defaultUnitType),
	}
	var notSoCommonLabels []kv.KeyValue

	ctx := context.Background()

	(*observerLock).Lock()
	*observerValueToReport = 1.0
	*observerLabelsToReport = commonLabels
	(*observerLock).Unlock()
	meter.RecordBatch(
		ctx,
		commonLabels,
		valueRecorder.Measurement(2.0),
		counter.Measurement(12.0),
	)

	time.Sleep(5 * time.Second)

	(*observerLock).Lock()
	*observerValueToReport = 1.0
	*observerLabelsToReport = notSoCommonLabels
	(*observerLock).Unlock()
	meter.RecordBatch(
		ctx,
		notSoCommonLabels,
	)

	time.Sleep(5 * time.Second)

	(*observerLock).Lock()
	*observerValueToReport = 13.0
	*observerLabelsToReport = commonLabels
	(*observerLock).Unlock()
	meter.RecordBatch(
		ctx,
		commonLabels,
		valueRecorder.Measurement(12.0),
		counter.Measurement(13.0),
	)

	time.Sleep(100 * time.Second)
}

