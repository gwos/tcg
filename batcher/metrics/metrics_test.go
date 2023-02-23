package metrics

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/gwos/tcg/sdk/transit"
	"github.com/stretchr/testify/assert"
)

func TestResourcesWithServicesRequest(t *testing.T) {
	p := []byte(`{
	"context": {
		"agentId": "3939333393342",
		"appType": "VEMA",
		"timeStamp": "1575367534330",
		"traceToken": "token-99e93",
		"version": "1.0.0"
	},
	"resources": [
		{
			"name": "watcher",
			"type": "host",
			"status": "HOST_UP",
			"lastCheckTime": "1575367534293",
			"nextCheckTime": "1575367534302",
			"services": [
				{
					"name": "disk.free",
					"type": "",
					"owner": "watcher",
					"status": "SERVICE_OK",
					"metrics": [
						{
							"metricName": "diskFree",
							"sampleType": "Value",
							"interval": {
								"endTime": "1575367534302",
								"startTime": "1575367534302"
							},
							"value": {
								"valueType": "IntegerType",
								"integerValue": 117369
							},
							"unit": "MB"
						}
					]
				}
			]
		}
	]
	}`)
	q := new(transit.ResourcesWithServicesRequest)
	assert.NoError(t, json.Unmarshal(p, q))
	s := fmt.Sprintf("%#v", q)
	assert.Contains(t, s, `Status:"SERVICE_OK", LastCheckTime:<nil>, NextCheckTime:<nil>`)
	/* no wrong dates added on re-marshaling */
	bb, err := json.MarshalIndent(q, "", "  ")
	assert.NoError(t, err)
	assert.JSONEq(t, string(p), string(bb))
	// t.Logf("%#v\n\n%s", q, bb)
}

func TestBuild(t *testing.T) {
	mbb := new(MetricsBatchBuilder)

	t.Run("split oversized", func(t *testing.T) {
		input := [][]byte{
			[]byte(`{"context":{"agentId":"b491b98e-c0cf-40c8-9938-3e30cb6a444c","appType":"APM","timeStamp":"1676407719960","traceToken":"b491b98e-c0cf-40c8-9938-3e30cb6a444c","version":"1.0.0"},"groups":[{"groupName":"KahunaHostGroup","resources":[{"name":"test-1","type":"host"}],"type":"HostGroup"}],"resources":[{"lastCheckTime":"1676407719000","lastPluginOutput":"Host has 8 OK, 0 WARNING, 0 CRITICAL and 0 other services.","name":"test-1","nextCheckTime":"1676408019000","services":[{"lastCheckTime":"1676407719000","lastPluginOutput":"OK - very good","metrics":[{"interval":{"endTime":"1676407719000","startTime":"1676407719000"},"metricName":"test_metric","sampleType":"Value","thresholds":[{"label":"test_metric_wn","sampleType":"Warning","value":{"doubleValue":85,"valueType":"DoubleType"}},{"label":"test_metric_cr","sampleType":"Critical","value":{"doubleValue":95,"valueType":"DoubleType"}}],"unit":"1","value":{"doubleValue":60,"valueType":"DoubleType"}}],"name":"test-service-10","nextCheckTime":"1676408019000","owner":"test-1","status":"SERVICE_OK","type":"service"},{"lastCheckTime":"1676407719000","lastPluginOutput":"OK - very good","metrics":[{"interval":{"endTime":"1676407719000","startTime":"1676407719000"},"metricName":"test_metric","sampleType":"Value","thresholds":[{"label":"test_metric_wn","sampleType":"Warning","value":{"doubleValue":85,"valueType":"DoubleType"}},{"label":"test_metric_cr","sampleType":"Critical","value":{"doubleValue":95,"valueType":"DoubleType"}}],"unit":"1","value":{"doubleValue":51,"valueType":"DoubleType"}}],"name":"test-service-3","nextCheckTime":"1676408019000","owner":"test-1","status":"SERVICE_OK","type":"service"},{"lastCheckTime":"1676407719000","lastPluginOutput":"OK - very good","metrics":[{"interval":{"endTime":"1676407719000","startTime":"1676407719000"},"metricName":"test_metric","sampleType":"Value","thresholds":[{"label":"test_metric_wn","sampleType":"Warning","value":{"doubleValue":85,"valueType":"DoubleType"}},{"label":"test_metric_cr","sampleType":"Critical","value":{"doubleValue":95,"valueType":"DoubleType"}}],"unit":"1","value":{"doubleValue":54,"valueType":"DoubleType"}}],"name":"test-service-4","nextCheckTime":"1676408019000","owner":"test-1","status":"SERVICE_OK","type":"service"},{"lastCheckTime":"1676407719000","lastPluginOutput":"OK - very good","metrics":[{"interval":{"endTime":"1676407719000","startTime":"1676407719000"},"metricName":"test_metric","sampleType":"Value","thresholds":[{"label":"test_metric_wn","sampleType":"Warning","value":{"doubleValue":85,"valueType":"DoubleType"}},{"label":"test_metric_cr","sampleType":"Critical","value":{"doubleValue":95,"valueType":"DoubleType"}}],"unit":"1","value":{"doubleValue":54,"valueType":"DoubleType"}}],"name":"test-service-5","nextCheckTime":"1676408019000","owner":"test-1","status":"SERVICE_OK","type":"service"},{"lastCheckTime":"1676407719000","lastPluginOutput":"OK - very good","metrics":[{"interval":{"endTime":"1676407719000","startTime":"1676407719000"},"metricName":"test_metric","sampleType":"Value","thresholds":[{"label":"test_metric_wn","sampleType":"Warning","value":{"doubleValue":85,"valueType":"DoubleType"}},{"label":"test_metric_cr","sampleType":"Critical","value":{"doubleValue":95,"valueType":"DoubleType"}}],"unit":"1","value":{"doubleValue":82,"valueType":"DoubleType"}}],"name":"test-service-6","nextCheckTime":"1676408019000","owner":"test-1","status":"SERVICE_OK","type":"service"},{"lastCheckTime":"1676407719000","lastPluginOutput":"OK - very good","metrics":[{"interval":{"endTime":"1676407719000","startTime":"1676407719000"},"metricName":"test_metric","sampleType":"Value","thresholds":[{"label":"test_metric_wn","sampleType":"Warning","value":{"doubleValue":85,"valueType":"DoubleType"}},{"label":"test_metric_cr","sampleType":"Critical","value":{"doubleValue":95,"valueType":"DoubleType"}}],"unit":"1","value":{"doubleValue":61,"valueType":"DoubleType"}}],"name":"test-service-7","nextCheckTime":"1676408019000","owner":"test-1","status":"SERVICE_OK","type":"service"},{"lastCheckTime":"1676407719000","lastPluginOutput":"OK - very good","metrics":[{"interval":{"endTime":"1676407719000","startTime":"1676407719000"},"metricName":"test_metric","sampleType":"Value","thresholds":[{"label":"test_metric_wn","sampleType":"Warning","value":{"doubleValue":85,"valueType":"DoubleType"}},{"label":"test_metric_cr","sampleType":"Critical","value":{"doubleValue":95,"valueType":"DoubleType"}}],"unit":"1","value":{"doubleValue":54,"valueType":"DoubleType"}}],"name":"test-service-8","nextCheckTime":"1676408019000","owner":"test-1","status":"SERVICE_OK","type":"service"},{"lastCheckTime":"1676407719000","lastPluginOutput":"OK - very good","metrics":[{"interval":{"endTime":"1676407719000","startTime":"1676407719000"},"metricName":"test_metric","sampleType":"Value","thresholds":[{"label":"test_metric_wn","sampleType":"Warning","value":{"doubleValue":85,"valueType":"DoubleType"}},{"label":"test_metric_cr","sampleType":"Critical","value":{"doubleValue":95,"valueType":"DoubleType"}}],"unit":"1","value":{"doubleValue":48,"valueType":"DoubleType"}}],"name":"test-service-9","nextCheckTime":"1676408019000","owner":"test-1","status":"SERVICE_OK","type":"service"}],"status":"HOST_UP","type":"host"}]}`),
		}
		output := mbb.Build(input, 1024)
		qq := make([]transit.ResourcesWithServicesRequest, 0)
		for _, p := range output {
			q := new(transit.ResourcesWithServicesRequest)
			assert.NoError(t, json.Unmarshal(p, q))
			qq = append(qq, *q)
		}
		assert.Equal(t, 3, len(qq))
		assert.Contains(t, qq[0].Context.TraceToken, "b491b98e-0000")
		assert.Contains(t, qq[1].Context.TraceToken, "b491b98e-0001")
		assert.Equal(t, 3, len(qq[0].Resources[0].Services))
		assert.Equal(t, 2, len(qq[2].Resources[0].Services))

		// bb, err := json.MarshalIndent(qq, "", "  ")
		// assert.NoError(t, err)
		// t.Logf("%#v\n\n%s", qq, bb)
	})

	t.Run("combine small", func(t *testing.T) {
		input := [][]byte{
			[]byte(`{"context":{"agentId":"2a728b11-3359-49b2-8611-98854927af2c","appType":"NAGIOS","timeStamp":"1676911781735","traceToken":"2a728b11-3359-49b2-8611-98854927af2c","version":"1.0.0"},"groups":[{"groupName":"HHostGroup22","resources":[{"name":"a-test-2","type":"host"}],"type":"HostGroup"}],"resources":[{"name":"a-test-2","services":[{"lastCheckTime":"1676911781000","lastPluginOutput":"CRITICAL - load average: 83.57, 36.40, 18.21","metrics":[{"interval":{"endTime":"1676911781280","startTime":"1676911781181"},"metricName":"load1","sampleType":"Value","thresholds":[{"label":"load1_wn","sampleType":"Warning","value":{"doubleValue":5.0,"integerValue":0,"valueType":"DoubleType"}},{"label":"load1_cr","sampleType":"Critical","value":{"doubleValue":10.0,"integerValue":0,"valueType":"DoubleType"}},{"label":"load1_mn","sampleType":"Min","value":{"doubleValue":0.0,"integerValue":0,"valueType":"DoubleType"}}],"unit":"1","value":{"doubleValue":83.569999999999993,"integerValue":0,"valueType":"DoubleType"}},{"interval":{"endTime":"1676911781280","startTime":"1676911781181"},"metricName":"load5","sampleType":"Value","thresholds":[{"label":"load5_wn","sampleType":"Warning","value":{"doubleValue":4.0,"integerValue":0,"valueType":"DoubleType"}},{"label":"load5_cr","sampleType":"Critical","value":{"doubleValue":8.0,"integerValue":0,"valueType":"DoubleType"}},{"label":"load5_mn","sampleType":"Min","value":{"doubleValue":0.0,"integerValue":0,"valueType":"DoubleType"}}],"unit":"1","value":{"doubleValue":36.399999999999999,"integerValue":0,"valueType":"DoubleType"}},{"interval":{"endTime":"1676911781280","startTime":"1676911781181"},"metricName":"load15","sampleType":"Value","thresholds":[{"label":"load15_wn","sampleType":"Warning","value":{"doubleValue":3.0,"integerValue":0,"valueType":"DoubleType"}},{"label":"load15_cr","sampleType":"Critical","value":{"doubleValue":6.0,"integerValue":0,"valueType":"DoubleType"}},{"label":"load15_mn","sampleType":"Min","value":{"doubleValue":0.0,"integerValue":0,"valueType":"DoubleType"}}],"unit":"1","value":{"doubleValue":18.210000000000001,"integerValue":0,"valueType":"DoubleType"}}],"name":"local_load_6009","nextCheckTime":"1676912381000","properties":{"CheckType":{"doubleValue":0.0,"integerValue":0,"stringValue":"ACTIVE","valueType":"StringType"},"CurrentAttempt":{"doubleValue":0.0,"integerValue":3,"valueType":"IntegerType"},"CurrentNotificationNumber":{"doubleValue":0.0,"integerValue":0,"valueType":"IntegerType"},"ExecutionTime":{"doubleValue":0.098902000000000004,"integerValue":0,"valueType":"DoubleType"},"LastNotificationTime":{"doubleValue":0.0,"integerValue":0,"timeValue":"0","valueType":"TimeType"},"LastStateChange":{"doubleValue":0.0,"integerValue":0,"timeValue":"1676911144000","valueType":"TimeType"},"Latency":{"doubleValue":11.620201110839844,"integerValue":0,"valueType":"DoubleType"},"MaxAttempts":{"doubleValue":0.0,"integerValue":3,"valueType":"IntegerType"},"PercentStateChange":{"doubleValue":31.44736842105263,"integerValue":0,"valueType":"DoubleType"},"ScheduledDowntimeDepth":{"doubleValue":0.0,"integerValue":0,"valueType":"IntegerType"},"StateType":{"doubleValue":0.0,"integerValue":0,"stringValue":"HARD","valueType":"StringType"},"isAcceptPassiveChecks":{"boolValue":true,"doubleValue":0.0,"integerValue":0,"valueType":"BooleanType"},"isChecksEnabled":{"boolValue":true,"doubleValue":0.0,"integerValue":0,"valueType":"BooleanType"},"isEventHandlersEnabled":{"boolValue":true,"doubleValue":0.0,"integerValue":0,"valueType":"BooleanType"},"isFlapDetectionEnabled":{"boolValue":true,"doubleValue":0.0,"integerValue":0,"valueType":"BooleanType"},"isNotificationsEnabled":{"boolValue":true,"doubleValue":0.0,"integerValue":0,"valueType":"BooleanType"},"isProblemAcknowledged":{"doubleValue":0.0,"integerValue":0,"valueType":"BooleanType"}},"status":"SERVICE_UNSCHEDULED_CRITICAL","type":"hypervisor"}],"status":"HOST_UNCHANGED","type":"host"}]}`),
			[]byte(`{"context":{"agentId":"2a728b11-3359-49b2-8611-98854927af2c","appType":"NAGIOS","timeStamp":"1676911783375","traceToken":"2a728b11-3359-49b2-8611-98854927af2c","version":"1.0.0"},"groups":[{"groupName":"HHostGroup11","resources":[{"name":"a-test-1","type":"host"}],"type":"HostGroup"}],"resources":[{"name":"a-test-1","services":[{"lastCheckTime":"1676911783000","lastPluginOutput":"WARNING: Stale Status","name":"linux_load_996","nextCheckTime":"1676911771000","properties":{"CheckType":{"doubleValue":0.0,"integerValue":0,"stringValue":"ACTIVE","valueType":"StringType"},"CurrentAttempt":{"doubleValue":0.0,"integerValue":1,"valueType":"IntegerType"},"CurrentNotificationNumber":{"doubleValue":0.0,"integerValue":0,"valueType":"IntegerType"},"ExecutionTime":{"doubleValue":0.0060609999999999995,"integerValue":0,"valueType":"DoubleType"}},"status":"SERVICE_WARNING","type":"hypervisor"}],"status":"HOST_UNCHANGED","type":"host"}]}`),
			[]byte(`{"context":{"agentId":"2a728b11-3359-49b2-8611-98854927af2c","appType":"NAGIOS","timeStamp":"1676911783386","traceToken":"2a728b11-3359-49b2-8611-98854927af2c","version":"1.0.0"},"groups":[{"groupName":"HHostGroup11","resources":[{"name":"a-test-1","type":"host"}],"type":"HostGroup"}],"resources":[{"name":"a-test-1","services":[{"lastCheckTime":"1676911783000","lastPluginOutput":"WARNING: Stale Status","name":"linux_load_997","nextCheckTime":"1676911771000","properties":{"CheckType":{"doubleValue":0.0,"integerValue":0,"stringValue":"ACTIVE","valueType":"StringType"},"CurrentAttempt":{"doubleValue":0.0,"integerValue":1,"valueType":"IntegerType"},"CurrentNotificationNumber":{"doubleValue":0.0,"integerValue":0,"valueType":"IntegerType"},"ExecutionTime":{"doubleValue":0.0053230000000000005,"integerValue":0,"valueType":"DoubleType"}},"status":"SERVICE_WARNING","type":"hypervisor"}],"status":"HOST_UNCHANGED","type":"host"}]}`),
			[]byte(`{"context":{"agentId":"2a728b11-3359-49b2-8611-98854927af2c","appType":"NAGIOS","timeStamp":"1676911783397","traceToken":"2a728b11-3359-49b2-8611-98854927af2c","version":"1.0.0"},"groups":[{"groupName":"HHostGroup11","resources":[{"name":"a-test-1","type":"host"}],"type":"HostGroup"}],"resources":[{"name":"a-test-1","services":[{"lastCheckTime":"1676911783000","lastPluginOutput":"WARNING: Stale Status","name":"linux_load_998","nextCheckTime":"1676911771000","properties":{"CheckType":{"doubleValue":0.0,"integerValue":0,"stringValue":"ACTIVE","valueType":"StringType"},"CurrentAttempt":{"doubleValue":0.0,"integerValue":1,"valueType":"IntegerType"},"CurrentNotificationNumber":{"doubleValue":0.0,"integerValue":0,"valueType":"IntegerType"},"ExecutionTime":{"doubleValue":0.0064409999999999997,"integerValue":0,"valueType":"DoubleType"}},"status":"SERVICE_WARNING","type":"hypervisor"}],"status":"HOST_UNCHANGED","type":"host"}]}`),
		}
		output := mbb.Build(input, 1024)
		qq := make([]transit.ResourcesWithServicesRequest, 0)
		for _, p := range output {
			// t.Logf("\n\n%s\n", p)
			q := new(transit.ResourcesWithServicesRequest)
			assert.NoError(t, json.Unmarshal(p, q))
			qq = append(qq, *q)
		}
		assert.Equal(t, 3, len(qq))
		assert.Equal(t, 1, len(qq[0].Groups))
		assert.Equal(t, "linux_load_996", qq[0].Resources[0].Services[0].Name)
		assert.Equal(t, "linux_load_997", qq[0].Resources[1].Services[0].Name)
		assert.Equal(t, "linux_load_998", qq[1].Resources[0].Services[0].Name)
		assert.Equal(t, "local_load_6009", qq[2].Resources[0].Services[0].Name)
	})
}

func BenchmarkCatStrings(b *testing.B) {
	rr := transit.ResourceRef{Type: transit.ResourceTypeHost, Name: "test_10"}

	b.Run("fmt.Sprintf", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			rk := fmt.Sprintf("%s:%s", rr.Type, rr.Name)
			assert.Contains(b, rk, ":test_")
		}
	})

	b.Run("strings.Join", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			rk := strings.Join([]string{string(rr.Type), rr.Name}, ":")
			assert.Contains(b, rk, ":test_")
		}
	})
}

func BenchmarkUpdateToken(b *testing.B) {
	t := "2a728b11-3359-49b2-8611-98854927af2c"

	b.Run("fmt.Sprintf", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			t2 := fmt.Sprintf("%s-%04d-%s", t[:8], i, t[14:])
			assert.Contains(b, t2, "2a728b11-")
		}
	})

	/* looking for alternatives */
}
