package helpers

import (
	"os"
	"testing"

	"github.com/gwos/tcg/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func configPayload(extensions string) []byte {
	return []byte(`{
	  "monitorConnection": {
	    "id": 7,
	    "extensions": ` + extensions + `
	  },
	  "metricsProfile": {"metrics": []}
	}`)
}

// installConfigState isolates the package config globals and the config file for a test.
func installConfigState(t *testing.T) {
	t.Helper()
	tmpFile, err := os.CreateTemp(t.TempDir(), "config")
	require.NoError(t, err)
	t.Setenv(config.ConfigEnv, tmpFile.Name())

	origExt, origProf, origConn, origCancel := extConfig, metricsProfile, monitorConnection, cancel
	t.Cleanup(func() {
		extConfig, metricsProfile, monitorConnection, cancel = origExt, origProf, origConn, origCancel
	})
}

func TestConfigHandlerAppliesMappings(t *testing.T) {
	installConfigState(t)

	ConfigHandler(configPayload(`{
	  "mapHostgroup": [{"tag": "cluster", "matcher": "(.+)", "template": "cluster-$1"}],
	  "mapHostname": [{"tag": "instance", "matcher": "^([^:]+)", "template": "$1"}],
	  "mapService": [{"tag": "alertname", "matcher": "(.+)", "template": "$1"}],
	  "mapIgnore": [{"tag": "severity", "matcher": "^info$", "template": "ignore"}]
	}`))

	cfg := GetExtConfig()
	require.Len(t, cfg.MapHostgroup, 1)
	require.Len(t, cfg.MapIgnore, 1)
	assert.Equal(t, 7, GetMonitorConnection().ID)
	assert.Same(t, cfg, GetMonitorConnection().Extensions)
	assert.NotNil(t, GetMetricsProfile())

	// The applied mappings are compiled and ready to use.
	tags := map[string]string{tagCluster: clusterProd, tagInstance: "node1:9100", tagAlertName: alertName}
	v, err := cfg.MapHostname.ApplyOR(tags)
	require.NoError(t, err)
	assert.Equal(t, hostName, v)
	assert.True(t, cfg.MapIgnore.Matches(map[string]string{"severity": "info"}))
}

func TestConfigHandlerKeepsConfigOnInvalidMapping(t *testing.T) {
	installConfigState(t)

	ConfigHandler(configPayload(`{
	  "mapHostname": [{"tag": "instance", "matcher": "(.+)", "template": "$1"}],
	  "mapService": [{"tag": "alertname", "matcher": "(.+)", "template": "$1"}]
	}`))
	current := GetExtConfig()

	for _, ext := range []string{
		`{"mapHostgroup": [{"tag": "cluster", "matcher": "(", "template": "$1"}]}`,
		`{"mapHostname": [{"tag": "instance", "matcher": "(", "template": "$1"}]}`,
		`{"mapService": [{"tag": "alertname", "matcher": "(", "template": "$1"}]}`,
		`{"mapIgnore": [{"tag": "severity", "matcher": "(", "template": "ignore"}]}`,
	} {
		ConfigHandler(configPayload(ext))
		assert.Same(t, current, GetExtConfig(), "an invalid config must keep the current one: %s", ext)
	}

	// The kept config still works.
	v, err := GetExtConfig().MapService.ApplyOR(map[string]string{tagAlertName: alertName})
	require.NoError(t, err)
	assert.Equal(t, alertName, v)
}

func TestConfigHandlerIgnoresUnparsableConfig(t *testing.T) {
	installConfigState(t)
	current := GetExtConfig()

	ConfigHandler([]byte(`{not json`))
	assert.Same(t, current, GetExtConfig())
}
