package events

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/gwos/tcg/config"
	"github.com/gwos/tcg/connectors/events/helpers"
	"github.com/gwos/tcg/sdk/transit"
	"github.com/gwos/tcg/services"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// installExport makes the transit service write sent metrics to a directory instead of sending them.
func installExport(t *testing.T) string {
	t.Helper()
	tmpFile, err := os.CreateTemp(t.TempDir(), "config")
	require.NoError(t, err)
	t.Setenv(config.ConfigEnv, tmpFile.Name())

	dir := t.TempDir()
	connector := services.GetTransitService().Connector
	origDir, origSuppress := connector.ExportTransitDir, config.Suppress.Metrics
	connector.ExportTransitDir, config.Suppress.Metrics = dir, true
	t.Cleanup(func() {
		connector.ExportTransitDir, config.Suppress.Metrics = origDir, origSuppress
	})
	return dir
}

func exportedMetrics(t *testing.T, dir string) []transit.ResourcesWithServicesRequest {
	t.Helper()
	files, err := filepath.Glob(filepath.Join(dir, "*-"+string(services.TOpSendMetrics)+".json"))
	require.NoError(t, err)
	requests := make([]transit.ResourcesWithServicesRequest, 0, len(files))
	for _, f := range files {
		b, err := os.ReadFile(f)
		require.NoError(t, err)
		var q transit.ResourcesWithServicesRequest
		require.NoError(t, json.Unmarshal(b, &q))
		requests = append(requests, q)
	}
	return requests
}

func postEvents(t *testing.T, body string) *httptest.ResponseRecorder {
	t.Helper()
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/receive/events", bytes.NewBufferString(body))
	receiver(c)
	return w
}

func TestReceiverBadRequest(t *testing.T) {
	dir := installExport(t)

	w := postEvents(t, `{"alerts": [`)
	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Empty(t, exportedMetrics(t, dir))
}

func TestReceiverSendsMappedAlerts(t *testing.T) {
	dir := installExport(t)
	helpers.ConfigHandler([]byte(`{
	  "monitorConnection": {
	    "id": 7,
	    "extensions": {
	      "mapHostgroup": [{"tag": "cluster", "matcher": "(.+)", "template": "cluster-$1"}],
	      "mapHostname": [{"tag": "instance", "matcher": "^([^:]+)", "template": "$1"}],
	      "mapService": [{"tag": "alertname", "matcher": "(.+)", "template": "$1"}],
	      "mapIgnore": [{"tag": "severity", "matcher": "^info$", "template": "ignore"}]
	    }
	  },
	  "metricsProfile": {"metrics": []}
	}`))

	w := postEvents(t, `{
	  "status": "firing",
	  "commonLabels": {"cluster": "prod"},
	  "alerts": [
	    {
	      "status": "firing",
	      "labels": {"alertname": "HighCPU", "instance": "node1:9100", "severity": "critical"},
	      "annotations": {"summary": "CPU is high", "description": "above 90%"},
	      "startsAt": "2026-10-02T10:00:00Z"
	    },
	    {
	      "status": "firing",
	      "labels": {"alertname": "Heartbeat", "instance": "node1:9100", "severity": "info"},
	      "startsAt": "2026-10-02T10:00:00Z"
	    },
	    {
	      "status": "firing",
	      "labels": {"alertname": "NoInstance", "severity": "critical"},
	      "startsAt": "2026-10-02T10:00:00Z"
	    }
	  ]
	}`)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	requests := exportedMetrics(t, dir)
	require.Len(t, requests, 1)
	q := requests[0]

	require.Len(t, q.Resources, 1)
	res := q.Resources[0]
	assert.Equal(t, "node1", res.Name)
	require.Len(t, res.Services, 1, "the ignored alert and the alert without instance must be skipped")
	svc := res.Services[0]
	assert.Equal(t, "HighCPU", svc.Name)
	assert.Equal(t, transit.ServiceWarning, svc.Status)
	assert.Equal(t, "CPU is high | above 90%", svc.LastPluginOutput)

	require.Len(t, q.Groups, 1)
	assert.Equal(t, "cluster-prod", q.Groups[0].GroupName)
	require.Len(t, q.Groups[0].Resources, 1)
	assert.Equal(t, "node1", q.Groups[0].Resources[0].Name)
}
