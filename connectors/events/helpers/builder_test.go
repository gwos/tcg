package helpers

import (
	"testing"
	"time"

	"github.com/gwos/tcg/sdk/mapping"
	"github.com/prometheus/alertmanager/template"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	tagAlertName = "alertname"
	tagInstance  = "instance"
	tagCluster   = "cluster"
	alertName    = "HighCPU"
	hostName     = "node1"
	clusterProd  = "prod"
	diskFull     = "disk full"
	onVar        = "on /var"
)

func testExtConfig() *ExtConfig {
	return &ExtConfig{
		MapHostgroup: mapping.Mappings{*mapping.NewMapping(tagCluster, "(.+)", "cluster-$1")},
		MapHostname:  mapping.Mappings{*mapping.NewMapping(tagInstance, "^([^:]+)", "$1")},
		MapService:   mapping.Mappings{*mapping.NewMapping(tagAlertName, "(.+)", "$1")},
		MapIgnore:    mapping.Mappings{*mapping.NewMapping("severity", "^info$", "ignore")},
	}
}

func TestParsePrometheusData(t *testing.T) {
	startsAt := time.Date(2026, 10, 2, 10, 0, 0, 0, time.FixedZone("CEST", 2*60*60))
	endsAt := startsAt.Add(time.Hour)
	labels := func(kv ...string) template.KV {
		m := template.KV{tagAlertName: alertName, tagInstance: "node1:9100", tagCluster: clusterProd}
		for i := 0; i < len(kv); i += 2 {
			if kv[i+1] == "" {
				delete(m, kv[i])
				continue
			}
			m[kv[i]] = kv[i+1]
		}
		return m
	}

	cases := []struct {
		name      string
		cfg       *ExtConfig
		alert     template.Alert
		wantHost  string
		wantGroup string
		skipped   bool
	}{
		{name: "mapped", cfg: testExtConfig(),
			alert:    template.Alert{Labels: labels(), StartsAt: startsAt, EndsAt: endsAt},
			wantHost: hostName, wantGroup: "cluster-prod"},
		{name: "ignored", cfg: testExtConfig(),
			alert:   template.Alert{Labels: labels("severity", "info")},
			skipped: true},
		{name: "not ignored", cfg: testExtConfig(),
			alert:    template.Alert{Labels: labels("severity", "critical")},
			wantHost: hostName, wantGroup: "cluster-prod"},
		{name: "hostgroup missed", cfg: testExtConfig(),
			alert:   template.Alert{Labels: labels(tagCluster, "")},
			skipped: true},
		{name: "hostgroup unconfigured", cfg: &ExtConfig{
			MapHostname: testExtConfig().MapHostname,
			MapService:  testExtConfig().MapService,
		},
			alert:    template.Alert{Labels: labels()},
			wantHost: hostName, wantGroup: ""},
		{name: "hostname missed", cfg: testExtConfig(),
			alert:   template.Alert{Labels: labels(tagInstance, "")},
			skipped: true},
		{name: "hostname unconfigured", cfg: &ExtConfig{MapService: testExtConfig().MapService},
			alert:   template.Alert{Labels: labels()},
			skipped: true},
		{name: "service missed", cfg: testExtConfig(),
			alert:   template.Alert{Labels: labels(tagAlertName, "")},
			skipped: true},
		{name: "service empty result", cfg: &ExtConfig{
			MapHostname: testExtConfig().MapHostname,
			MapService:  mapping.Mappings{*mapping.NewMapping(tagAlertName, "(.*)", "")},
		},
			alert:   template.Alert{Labels: labels()},
			skipped: true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			results, err := ParsePrometheusData(template.Data{Alerts: template.Alerts{tc.alert}}, tc.cfg)
			require.NoError(t, err)
			if tc.skipped {
				assert.Empty(t, results)
				return
			}
			require.Len(t, results, 1)
			assert.Equal(t, tc.wantHost, results[0].HostName)
			assert.Equal(t, tc.wantGroup, results[0].HostGroupName)
			assert.Equal(t, alertName, results[0].MetricBuilder.Name)
		})
	}
}

func TestParsePrometheusDataTimestamps(t *testing.T) {
	startsAt := time.Date(2026, 10, 2, 10, 0, 0, 0, time.FixedZone("CEST", 2*60*60))
	endsAt := startsAt.Add(time.Hour)
	alertLabels := template.KV{tagAlertName: alertName, tagInstance: hostName, tagCluster: clusterProd}
	data := template.Data{Alerts: template.Alerts{
		{Labels: alertLabels, StartsAt: startsAt, EndsAt: endsAt},
		{Labels: alertLabels, StartsAt: startsAt},
	}}

	results, err := ParsePrometheusData(data, testExtConfig())
	require.NoError(t, err)
	require.Len(t, results, 2)

	assert.Equal(t, startsAt.UTC(), results[0].MetricBuilder.StartTimestamp.Time)
	assert.Equal(t, endsAt.UTC(), results[0].MetricBuilder.EndTimestamp.Time)
	assert.Equal(t, time.UTC, results[0].MetricBuilder.StartTimestamp.Location())

	// Without EndsAt the alert ends when it starts.
	assert.Equal(t, startsAt.UTC(), results[1].MetricBuilder.StartTimestamp.Time)
	assert.Equal(t, startsAt.UTC(), results[1].MetricBuilder.EndTimestamp.Time)
}

func TestParsePrometheusDataTagPrecedence(t *testing.T) {
	data := template.Data{
		CommonAnnotations: template.KV{tagKeySummary: "common annotation", tagKeyDescription: "common description"},
		CommonLabels:      template.KV{tagCluster: "common", tagInstance: "common-node", tagKeySummary: "common label"},
		Alerts: template.Alerts{{
			Annotations: template.KV{tagKeySummary: "alert annotation"},
			Labels:      template.KV{tagAlertName: alertName, tagInstance: hostName},
		}},
	}

	results, err := ParsePrometheusData(data, testExtConfig())
	require.NoError(t, err)
	require.Len(t, results, 1)

	tags := results[0].MetricBuilder.Tags
	// Alert labels override common labels, which override annotations.
	assert.Equal(t, hostName, tags[tagInstance])
	assert.Equal(t, "common", tags[tagCluster])
	assert.Equal(t, "common label", tags[tagKeySummary])
	assert.Equal(t, "common description", tags[tagKeyDescription])
	assert.Equal(t, "cluster-common", results[0].HostGroupName)
	assert.Equal(t, hostName, results[0].HostName)
}

func TestGetLastPluginOutput(t *testing.T) {
	cases := []struct {
		name string
		tags map[string]string
		want string
	}{
		{"none", map[string]string{}, ""},
		{"summary only", map[string]string{tagKeySummary: diskFull}, diskFull},
		{"description only", map[string]string{tagKeyDescription: onVar}, onVar},
		{"both", map[string]string{tagKeySummary: diskFull, tagKeyDescription: onVar}, diskFull + " | " + onVar},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, GetLastPluginOutput(tc.tags))
		})
	}
}
