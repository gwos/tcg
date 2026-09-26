package metrics

import (
	"encoding/json"
	"fmt"
	"math"
	"testing"
	"time"

	"github.com/gwos/tcg/batcher"
	"github.com/gwos/tcg/sdk/transit"
	"github.com/stretchr/testify/assert"
)

func ts(t time.Time) *transit.Timestamp {
	p := transit.NewTimestamp()
	p.Time = t
	return p
}

// nagiosServiceCheck makes a request shaped like DataGeyser service-check results
func nagiosServiceCheck(host string, i, nMetrics int) *transit.ResourcesWithServicesRequest {
	now := time.Now()
	svc := transit.MonitoredService{}
	svc.Name = fmt.Sprintf("local_load_%d", i)
	svc.Type = transit.ResourceTypeService
	svc.Status = transit.ServiceOk
	svc.LastPluginOutput = "OK - load average: 0.12, 0.34, 0.56"
	svc.LastCheckTime, svc.NextCheckTime = ts(now), ts(now.Add(5*time.Minute))
	svc.SetProperty("isChecksEnabled", true)
	svc.SetProperty("CurrentAttempt", int64(1))
	svc.SetProperty("CurrentNotificationNumber", int64(0))
	svc.SetProperty("LastNotificationTime", transit.Timestamp{Time: now})
	svc.SetProperty("ScheduledDowntimeDepth", int64(0))
	svc.SetProperty("isAcceptPassiveChecks", true)
	svc.SetProperty("isNotificationsEnabled", true)
	svc.SetProperty("isProblemAcknowledged", false)
	svc.SetProperty("MaxAttempts", int64(3))
	svc.SetProperty("PercentStateChange", 0.0)
	svc.SetProperty("Latency", 0.001234)
	svc.SetProperty("ExecutionTime", 0.012345)
	svc.SetProperty("CheckType", "ACTIVE")
	svc.SetProperty("StateType", "HARD")
	svc.SetProperty("PerformanceData", "load1=0.120;5.000;10.000;0; load5=0.340;4.000;6.000;0;")
	for m := range nMetrics {
		name := fmt.Sprintf("load%d", m)
		t := transit.TimeSeries{MetricName: name, SampleType: transit.Value, Unit: transit.UnitCounter}
		t.SetIntervalStart(ts(now))
		t.SetIntervalEnd(ts(now))
		t.SetValue(0.12)
		t.AddThreshold(transit.ThresholdValue{SampleType: transit.Warning, Label: name + "_wn", Value: transit.NewTypedValue(5.0)})
		t.AddThreshold(transit.ThresholdValue{SampleType: transit.Critical, Label: name + "_cr", Value: transit.NewTypedValue(10.0)})
		t.AddThreshold(transit.ThresholdValue{SampleType: transit.Min, Label: name + "_min", Value: transit.NewTypedValue(0.0)})
		svc.AddMetric(t)
	}
	res := transit.MonitoredResource{}
	res.Name, res.Type, res.Status = host, transit.ResourceTypeHost, transit.HostUnchanged
	res.AddService(svc)
	q := new(transit.ResourcesWithServicesRequest)
	q.Context = transit.TracerContext{
		AppType: "NAGIOS", AgentID: "2a728b11-3359-49b2-8611-98854927af2c",
		TraceToken: "2a728b11-3359-49b2-8611-98854927af2c", TimeStamp: ts(now), Version: transit.ModelVersion,
	}
	q.AddResource(res)
	return q
}

// nagiosHostCheck makes a request shaped like DataGeyser host-check results
func nagiosHostCheck(host string) *transit.ResourcesWithServicesRequest {
	q := nagiosServiceCheck(host, 0, 0)
	res := &q.Resources[0]
	res.Services = []transit.MonitoredService{}
	res.Status = transit.HostUp
	res.LastPluginOutput = "PING OK - Packet loss = 0%, RTA = 0.05 ms"
	res.Properties = q.Resources[0].Properties
	for k, v := range map[string]any{"isAcknowledged": false, "CurrentAttempt": int64(1), "Latency": 0.1, "CheckType": "ACTIVE"} {
		res.SetProperty(k, v)
	}
	return q
}

func TestEstimateSize(t *testing.T) {
	cases := map[string]*transit.ResourcesWithServicesRequest{
		"host check":          nagiosHostCheck("host-1"),
		"service no perfdata": nagiosServiceCheck("host-1", 1, 0),
		"service 1 metric":    nagiosServiceCheck("host-1", 1, 1),
		"service 10 metrics":  nagiosServiceCheck("host-1", 1, 10),
		"empty":               {},
		"groups":              {Groups: []transit.ResourceGroup{{GroupName: "HG1", Type: transit.HostGroup, Resources: []transit.ResourceRef{{Name: "host-1", Type: transit.ResourceTypeHost}}}}},
	}
	for name, q := range cases {
		t.Run(name, func(t *testing.T) {
			bb, err := json.Marshal(q)
			assert.NoError(t, err)
			est := EstimateSize(q)
			ratio := float64(est) / float64(len(bb))
			t.Logf("estimate/actual %d/%d = %.2f", est, len(bb), ratio)
			assert.InDelta(t, 1.0, ratio, 0.25)
		})
	}
}

func TestBuildTyped(t *testing.T) {
	mbb := new(MetricsBatchBuilder)

	t.Run("status splits merge group", func(t *testing.T) {
		qq := []*transit.ResourcesWithServicesRequest{
			nagiosServiceCheck("host-1", 1, 1),
			nagiosServiceCheck("host-1", 2, 1),
			nagiosHostCheck("host-1"),
			nagiosServiceCheck("host-2", 3, 1),
		}
		payloads := mbb.Build(sized(qq), 1024*1024)
		assert.Equal(t, 3, len(payloads))
		out := unmarshalAll(t, payloads)
		assert.Equal(t, 2, len(out[0].Resources))
		assert.Equal(t, transit.HostUp, out[1].Resources[0].Status)
		assert.Equal(t, "local_load_3", out[2].Resources[0].Services[0].Name)
	})

	t.Run("chunk by estimate", func(t *testing.T) {
		qq := make([]*transit.ResourcesWithServicesRequest, 0, 100)
		for i := range 100 {
			qq = append(qq, nagiosServiceCheck("host-1", i, 3))
		}
		maxBytes := 16 * 1024
		payloads := mbb.Build(sized(qq), maxBytes)
		assert.Greater(t, len(payloads), 1)
		n := 0
		for _, p := range payloads {
			assert.Less(t, len(p), 2*maxBytes)
			n += len(unmarshalAll(t, [][]byte{p})[0].Resources)
		}
		assert.Equal(t, 100, n)
	})

	t.Run("drop invalid", func(t *testing.T) {
		qq := []*transit.ResourcesWithServicesRequest{
			nagiosServiceCheck("host-1", 1, 1),
			nagiosServiceCheck("host-1", 2, 1),
			nagiosServiceCheck("host-2", 3, 1),
			nagiosServiceCheck("host-3", 4, 1),
		}
		/* NaN metric value makes the service invalid */
		qq[1].Resources[0].Services[0].Metrics[0].SetValue(math.NaN())
		/* +Inf property makes the resource invalid */
		qq[2].Resources[0].SetProperty("Latency", math.Inf(1))

		payloads := mbb.Build(sized(qq), 1024*1024)
		if assert.Equal(t, 1, len(payloads)) {
			out := unmarshalAll(t, payloads)[0]
			names := []string{}
			for _, res := range out.Resources {
				for _, svc := range res.Services {
					names = append(names, res.Name+":"+svc.Name)
				}
			}
			assert.Equal(t, []string{"host-1:local_load_1", "host-3:local_load_4"}, names)
		}
	})

	t.Run("split oversized", func(t *testing.T) {
		q := nagiosServiceCheck("host-1", 0, 1)
		for i := 1; i < 50; i++ {
			q.Resources[0].AddService(nagiosServiceCheck("host-1", i, 1).Resources[0].Services[0])
		}
		maxBytes := 8 * 1024
		payloads := mbb.Build(sized([]*transit.ResourcesWithServicesRequest{q}), maxBytes)
		assert.Greater(t, len(payloads), 1)
		n := 0
		for i, x := range unmarshalAll(t, payloads) {
			assert.Contains(t, x.Context.TraceToken, fmt.Sprintf("2a728b11-%04d-", i))
			n += len(x.Resources[0].Services)
		}
		assert.Equal(t, 50, n)
		/* input is not modified */
		assert.Equal(t, 50, len(q.Resources[0].Services))
		assert.Equal(t, "2a728b11-3359-49b2-8611-98854927af2c", q.Context.TraceToken)
	})
}

// BenchmarkBatch compares the legacy flow (marshal on the caller's thread,
// unmarshal and marshal again in the builder) with the typed one
func BenchmarkBatch(b *testing.B) {
	const n = 200
	qq := make([]*transit.ResourcesWithServicesRequest, 0, n)
	for i := range n {
		qq = append(qq, nagiosServiceCheck(fmt.Sprintf("host-%d", i%20), i, 2))
	}
	mbb := new(MetricsBatchBuilder)
	maxBytes := 100 * 1024

	b.Run("legacy", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			items := make([]batcher.Sized[*transit.ResourcesWithServicesRequest], 0, n)
			for _, q := range qq {
				p, _ := json.Marshal(q) // libtransit SendChecks
				x := new(transit.ResourcesWithServicesRequest)
				_ = json.Unmarshal(p, x) // legacy Build
				items = append(items, batcher.Sized[*transit.ResourcesWithServicesRequest]{Value: x, Size: len(p)})
			}
			_ = mbb.Build(items, maxBytes)
		}
	})

	b.Run("typed", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			_ = mbb.Build(sized(qq), maxBytes)
		}
	})

	b.Run("add legacy", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			for _, q := range qq {
				_, _ = json.Marshal(q)
			}
		}
	})

	b.Run("add typed", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			for _, q := range qq {
				_ = EstimateSize(q)
			}
		}
	})
}

func sized(qq []*transit.ResourcesWithServicesRequest) []batcher.Sized[*transit.ResourcesWithServicesRequest] {
	items := make([]batcher.Sized[*transit.ResourcesWithServicesRequest], 0, len(qq))
	for _, q := range qq {
		items = append(items, batcher.Sized[*transit.ResourcesWithServicesRequest]{Value: q, Size: EstimateSize(q)})
	}
	return items
}

func unmarshalAll(t testing.TB, payloads [][]byte) []transit.ResourcesWithServicesRequest {
	qq := make([]transit.ResourcesWithServicesRequest, 0, len(payloads))
	for _, p := range payloads {
		var q transit.ResourcesWithServicesRequest
		assert.NoError(t, json.Unmarshal(p, &q))
		qq = append(qq, q)
	}
	return qq
}
