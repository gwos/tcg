package main

/*
#include <stdlib.h>

typedef const char cchar_t;
*/
import "C"
import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
	"unsafe"

	"github.com/gwos/tcg/config"
	"github.com/gwos/tcg/sdk/transit"
	"github.com/gwos/tcg/services"
	"github.com/stretchr/testify/assert"
)

func cstr(t *testing.T, s string) *C.cchar_t {
	p := C.CString(s)
	t.Cleanup(func() { C.free(unsafe.Pointer(p)) })
	return (*C.cchar_t)(p)
}

// setupSend configures TCG to export requests and not to reach NATS
func setupSend(t *testing.T, batch time.Duration) string {
	dir := t.TempDir()
	svc := services.GetTransitService()
	exportDir, batchMetrics, batchEvents := svc.Connector.ExportTransitDir, svc.Connector.BatchMetrics, svc.Connector.BatchEvents
	suppressMetrics, suppressEvents := config.Suppress.Metrics, config.Suppress.Events
	t.Cleanup(func() {
		svc.Connector.ExportTransitDir, svc.Connector.BatchMetrics, svc.Connector.BatchEvents = exportDir, batchMetrics, batchEvents
		config.Suppress.Metrics, config.Suppress.Events = suppressMetrics, suppressEvents
	})
	svc.Connector.ExportTransitDir = dir
	svc.Connector.BatchMetrics, svc.Connector.BatchEvents = batch, batch
	/* without batching the request goes to NATS right away */
	config.Suppress.Metrics, config.Suppress.Events = batch == 0, batch == 0
	return dir
}

func readExported(t *testing.T, dir string, v any) {
	ff, err := filepath.Glob(filepath.Join(dir, "*.json"))
	assert.NoError(t, err)
	if assert.Equal(t, 1, len(ff)) {
		bb, err := os.ReadFile(ff[0])
		assert.NoError(t, err)
		assert.NoError(t, json.Unmarshal(bb, v))
	}
}

func testSendChecks(t *testing.T) {
	for name, batch := range map[string]time.Duration{"no batch": 0, "batch": time.Hour} {
		t.Run(name, func(t *testing.T) {
			dir := setupSend(t, batch)
			errBuf := (*C.char)(C.malloc(256))
			defer C.free(unsafe.Pointer(errBuf))

			/* the same calls as DataGeyser does */
			res := CreateMonitoredResource(cstr(t, "host-1"))
			SetStatus(res, cstr(t, string(transit.HostUnchanged)))
			svc := CreateMonitoredService(cstr(t, "svc-1"))
			SetStatus(svc, cstr(t, string(transit.ServiceOk)))
			ts := CreateTimeSeries(cstr(t, "load1"))
			SetValueDouble(ts, 0.5)
			AddThresholdDouble(ts, cstr(t, "load1_wn"), cstr(t, string(transit.Warning)), 5)
			AddMetric(svc, ts)
			DeleteHandle(ts)
			AddService(res, svc)
			DeleteHandle(svc)
			req := CreateResourcesWithServicesRequest()
			AddResource(req, res)
			DeleteHandle(res)

			ok := SendChecks(req, errBuf, 256)
			DeleteHandle(req)
			assert.True(t, bool(ok), C.GoString(errBuf))

			var q transit.ResourcesWithServicesRequest
			readExported(t, dir, &q)
			if assert.Equal(t, 1, len(q.Resources)) {
				assert.Equal(t, "host-1", q.Resources[0].Name)
				assert.Equal(t, "svc-1", q.Resources[0].Services[0].Name)
				assert.Equal(t, "load1_wn", q.Resources[0].Services[0].Metrics[0].Thresholds[0].Label)
			}
		})
	}
}

func testSendEvents(t *testing.T) {
	for name, batch := range map[string]time.Duration{"no batch": 0, "batch": time.Hour} {
		t.Run(name, func(t *testing.T) {
			dir := setupSend(t, batch)
			errBuf := (*C.char)(C.malloc(256))
			defer C.free(unsafe.Pointer(errBuf))

			ev := CreateEvent(cstr(t, "NAGIOS"), cstr(t, "host-1"), cstr(t, "WARNING"), 1700000000, 0)
			req := CreateEventsRequest()
			AddEvent(req, ev)
			DeleteHandle(ev)

			ok := Send(req, errBuf, 256)
			DeleteHandle(req)
			assert.True(t, bool(ok), C.GoString(errBuf))

			var q transit.GroundworkEventsRequest
			readExported(t, dir, &q)
			if assert.Equal(t, 1, len(q.Events)) {
				assert.Equal(t, "host-1", q.Events[0].Host)
			}
		})
	}
}
