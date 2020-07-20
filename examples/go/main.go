package main

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"sync"
	"time"

	"go.opentelemetry.io/otel/api/global"
	"go.opentelemetry.io/otel/api/kv"
	"go.opentelemetry.io/otel/api/metric"
	"go.opentelemetry.io/otel/exporters/metric/prometheus"
)

var (
	defaultResource = "golang-server"
	defaultGroup    = "Prometheus-Go"
	defaultWarning  = fmt.Sprintf("%f", rand.Float64())
	defaultCritical = fmt.Sprintf("%f", rand.Float64())
	defaultUnitType = "MB"
	defaultPort     = ":2222"
)

func initMeter() {
	exporter, err := prometheus.InstallNewPipeline(prometheus.Config{})
	if err != nil {
		log.Panicf("Failed to initialize prometheus exporter %v", err)
	}

	http.HandleFunc("/", exporter.ServeHTTP)

	go func() {
		_ = http.ListenAndServe(defaultPort, nil)
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
	*observerValueToReport = rand.Float64()
	*observerLabelsToReport = commonLabels
	(*observerLock).Unlock()
	meter.RecordBatch(
		ctx,
		commonLabels,
		valueRecorder.Measurement(rand.Float64()),
		counter.Measurement(rand.Float64()),
	)

	time.Sleep(5 * time.Second)

	(*observerLock).Lock()
	*observerValueToReport = rand.Float64()
	*observerLabelsToReport = notSoCommonLabels
	(*observerLock).Unlock()
	meter.RecordBatch(
		ctx,
		notSoCommonLabels,
	)

	time.Sleep(5 * time.Second)

	(*observerLock).Lock()
	*observerValueToReport = rand.Float64()
	*observerLabelsToReport = commonLabels
	(*observerLock).Unlock()
	meter.RecordBatch(
		ctx,
		commonLabels,
		valueRecorder.Measurement(rand.Float64()),
		counter.Measurement(rand.Float64()),
	)

	time.Sleep(100 * time.Second)
}
