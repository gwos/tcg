package metrics

import (
	"encoding/json"
	"fmt"
	"runtime"
	"strings"
	"testing"
	"time"

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
	var q transit.ResourcesWithServicesRequest
	assert.NoError(t, json.Unmarshal(p, &q))
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
		buf := [][]byte{
			[]byte(`{"context":{"agentId":"b491b98e-c0cf-40c8-9938-3e30cb6a444c","appType":"APM","timeStamp":"1676407719960","traceToken":"b491b98e-c0cf-40c8-9938-3e30cb6a444c","version":"1.0.0"},"groups":[{"groupName":"KahunaHostGroup","resources":[{"name":"test-1","type":"host"}],"type":"HostGroup"}],"resources":[{"lastCheckTime":"1676407719000","lastPluginOutput":"Host has 8 OK, 0 WARNING, 0 CRITICAL and 0 other services.","name":"test-1","nextCheckTime":"1676408019000","services":[{"lastCheckTime":"1676407719000","lastPluginOutput":"OK - very good","metrics":[{"interval":{"endTime":"1676407719000","startTime":"1676407719000"},"metricName":"test_metric","sampleType":"Value","thresholds":[{"label":"test_metric_wn","sampleType":"Warning","value":{"doubleValue":85,"valueType":"DoubleType"}},{"label":"test_metric_cr","sampleType":"Critical","value":{"doubleValue":95,"valueType":"DoubleType"}}],"unit":"1","value":{"doubleValue":60,"valueType":"DoubleType"}}],"name":"test-service-10","nextCheckTime":"1676408019000","owner":"test-1","status":"SERVICE_OK","type":"service"},{"lastCheckTime":"1676407719000","lastPluginOutput":"OK - very good","metrics":[{"interval":{"endTime":"1676407719000","startTime":"1676407719000"},"metricName":"test_metric","sampleType":"Value","thresholds":[{"label":"test_metric_wn","sampleType":"Warning","value":{"doubleValue":85,"valueType":"DoubleType"}},{"label":"test_metric_cr","sampleType":"Critical","value":{"doubleValue":95,"valueType":"DoubleType"}}],"unit":"1","value":{"doubleValue":51,"valueType":"DoubleType"}}],"name":"test-service-3","nextCheckTime":"1676408019000","owner":"test-1","status":"SERVICE_OK","type":"service"},{"lastCheckTime":"1676407719000","lastPluginOutput":"OK - very good","metrics":[{"interval":{"endTime":"1676407719000","startTime":"1676407719000"},"metricName":"test_metric","sampleType":"Value","thresholds":[{"label":"test_metric_wn","sampleType":"Warning","value":{"doubleValue":85,"valueType":"DoubleType"}},{"label":"test_metric_cr","sampleType":"Critical","value":{"doubleValue":95,"valueType":"DoubleType"}}],"unit":"1","value":{"doubleValue":54,"valueType":"DoubleType"}}],"name":"test-service-4","nextCheckTime":"1676408019000","owner":"test-1","status":"SERVICE_OK","type":"service"},{"lastCheckTime":"1676407719000","lastPluginOutput":"OK - very good","metrics":[{"interval":{"endTime":"1676407719000","startTime":"1676407719000"},"metricName":"test_metric","sampleType":"Value","thresholds":[{"label":"test_metric_wn","sampleType":"Warning","value":{"doubleValue":85,"valueType":"DoubleType"}},{"label":"test_metric_cr","sampleType":"Critical","value":{"doubleValue":95,"valueType":"DoubleType"}}],"unit":"1","value":{"doubleValue":54,"valueType":"DoubleType"}}],"name":"test-service-5","nextCheckTime":"1676408019000","owner":"test-1","status":"SERVICE_OK","type":"service"},{"lastCheckTime":"1676407719000","lastPluginOutput":"OK - very good","metrics":[{"interval":{"endTime":"1676407719000","startTime":"1676407719000"},"metricName":"test_metric","sampleType":"Value","thresholds":[{"label":"test_metric_wn","sampleType":"Warning","value":{"doubleValue":85,"valueType":"DoubleType"}},{"label":"test_metric_cr","sampleType":"Critical","value":{"doubleValue":95,"valueType":"DoubleType"}}],"unit":"1","value":{"doubleValue":82,"valueType":"DoubleType"}}],"name":"test-service-6","nextCheckTime":"1676408019000","owner":"test-1","status":"SERVICE_OK","type":"service"},{"lastCheckTime":"1676407719000","lastPluginOutput":"OK - very good","metrics":[{"interval":{"endTime":"1676407719000","startTime":"1676407719000"},"metricName":"test_metric","sampleType":"Value","thresholds":[{"label":"test_metric_wn","sampleType":"Warning","value":{"doubleValue":85,"valueType":"DoubleType"}},{"label":"test_metric_cr","sampleType":"Critical","value":{"doubleValue":95,"valueType":"DoubleType"}}],"unit":"1","value":{"doubleValue":61,"valueType":"DoubleType"}}],"name":"test-service-7","nextCheckTime":"1676408019000","owner":"test-1","status":"SERVICE_OK","type":"service"},{"lastCheckTime":"1676407719000","lastPluginOutput":"OK - very good","metrics":[{"interval":{"endTime":"1676407719000","startTime":"1676407719000"},"metricName":"test_metric","sampleType":"Value","thresholds":[{"label":"test_metric_wn","sampleType":"Warning","value":{"doubleValue":85,"valueType":"DoubleType"}},{"label":"test_metric_cr","sampleType":"Critical","value":{"doubleValue":95,"valueType":"DoubleType"}}],"unit":"1","value":{"doubleValue":54,"valueType":"DoubleType"}}],"name":"test-service-8","nextCheckTime":"1676408019000","owner":"test-1","status":"SERVICE_OK","type":"service"},{"lastCheckTime":"1676407719000","lastPluginOutput":"OK - very good","metrics":[{"interval":{"endTime":"1676407719000","startTime":"1676407719000"},"metricName":"test_metric","sampleType":"Value","thresholds":[{"label":"test_metric_wn","sampleType":"Warning","value":{"doubleValue":85,"valueType":"DoubleType"}},{"label":"test_metric_cr","sampleType":"Critical","value":{"doubleValue":95,"valueType":"DoubleType"}}],"unit":"1","value":{"doubleValue":48,"valueType":"DoubleType"}}],"name":"test-service-9","nextCheckTime":"1676408019000","owner":"test-1","status":"SERVICE_OK","type":"service"}],"status":"HOST_UP","type":"host"}]}`),
		}
		mbb.Build(&buf, 1024)
		qq := make([]transit.ResourcesWithServicesRequest, 0)
		var q transit.ResourcesWithServicesRequest
		for _, p := range buf {
			q = transit.ResourcesWithServicesRequest{}
			assert.NoError(t, json.Unmarshal(p, &q))
			qq = append(qq, q)
		}
		assert.Equal(t, 4, len(qq))
		assert.Contains(t, qq[0].Context.TraceToken, "b491b98e-0000")
		assert.Contains(t, qq[1].Context.TraceToken, "b491b98e-0001")
		assert.Equal(t, 2, len(qq[0].Resources[0].Services))
		assert.Equal(t, 2, len(qq[3].Resources[0].Services))

		// bb, err := json.MarshalIndent(qq, "", "  ")
		// assert.NoError(t, err)
		// t.Logf("%#v\n\n%s", qq, bb)
	})

	t.Run("split oversized without metrics", func(t *testing.T) {
		buf := [][]byte{
			[]byte(`{"context":{"agentId":"a3d1a71c-3123-4cf4-bc44-493abe8d693e","appType":"NAGIOS","timeStamp":"1747733605675","traceToken":"b491b98e-c0cf-40c8-9938-3e30cb6a444c","version":"1.0.0"},"resources":[{"description":"atest11","device":"127.1.1.11","lastCheckTime":"1747756069000","lastPluginOutput":"OK - 127.1.1.11 rta 0.032ms lost 0%","name":"atest11","nextCheckTime":"1747756129000","properties":{"Alias":{"stringValue":"atest11","valueType":"StringType"},"CheckType":{"stringValue":"ACTIVE","valueType":"StringType"},"CurrentAttempt":{"integerValue":1,"valueType":"IntegerType"},"CurrentNotificationNumber":{"integerValue":0,"valueType":"IntegerType"},"ExecutionTime":{"doubleValue":0.003,"valueType":"DoubleType"},"LastNotificationTime":{"timeValue":"0","valueType":"TimeType"},"LastStateChange":{"timeValue":"1747683022000","valueType":"TimeType"},"Latency":{"doubleValue":0,"valueType":"DoubleType"},"MaxAttempts":{"integerValue":3,"valueType":"IntegerType"},"PercentStateChange":{"doubleValue":0,"valueType":"DoubleType"},"PerformanceData":{"stringValue":"rta=0.032ms;3000.000;5000.000;0; pl=0%;80;100;0;100 rtmax=0.032ms;;;; rtmin=0.032ms;;;;","valueType":"StringType"},"StateType":{"stringValue":"HARD","valueType":"StringType"},"isAcknowledged":{"boolValue":false,"valueType":"BooleanType"},"isChecksEnabled":{"boolValue":true,"valueType":"BooleanType"},"isEventHandlersEnabled":{"boolValue":true,"valueType":"BooleanType"},"isFlapDetectionEnabled":{"boolValue":true,"valueType":"BooleanType"},"isNotificationsEnabled":{"boolValue":true,"valueType":"BooleanType"},"isPassiveChecksEnabled":{"boolValue":true,"valueType":"BooleanType"}},"services":[{"lastCheckTime":"1747756069000","lastPluginOutput":"OK - 127.1.1.11 rta 0.050ms lost 0%","name":"icmp_ping_alive","nextCheckTime":"1747756129000","properties":{"CheckType":{"stringValue":"ACTIVE","valueType":"StringType"},"CurrentAttempt":{"integerValue":1,"valueType":"IntegerType"},"CurrentNotificationNumber":{"integerValue":0,"valueType":"IntegerType"},"ExecutionTime":{"doubleValue":0.004,"valueType":"DoubleType"},"LastNotificationTime":{"timeValue":"0","valueType":"TimeType"},"Latency":{"doubleValue":0,"valueType":"DoubleType"},"MaxAttempts":{"integerValue":3,"valueType":"IntegerType"},"PercentStateChange":{"doubleValue":0,"valueType":"DoubleType"},"PerformanceData":{"stringValue":"rta=0.050ms;3000.000;5000.000;0; pl=0%;80;100;0;100 rtmax=0.050ms;;;; rtmin=0.050ms;;;;","valueType":"StringType"},"StateType":{"stringValue":"HARD","valueType":"StringType"},"isAcceptPassiveChecks":{"boolValue":true,"valueType":"BooleanType"},"isChecksEnabled":{"boolValue":true,"valueType":"BooleanType"},"isEventHandlersEnabled":{"boolValue":true,"valueType":"BooleanType"},"isFlapDetectionEnabled":{"boolValue":true,"valueType":"BooleanType"},"isNotificationsEnabled":{"boolValue":true,"valueType":"BooleanType"},"isProblemAcknowledged":{"boolValue":false,"valueType":"BooleanType"}},"status":"SERVICE_OK","type":"service"}],"status":"HOST_UP","type":"host"},{"description":"atest22","device":"127.1.1.22","lastCheckTime":"1747756029000","lastPluginOutput":"OK - 127.1.1.22 rta 0.044ms lost 0%","name":"atest22","nextCheckTime":"1747756091000","properties":{"Alias":{"stringValue":"atest22","valueType":"StringType"},"CheckType":{"stringValue":"ACTIVE","valueType":"StringType"},"CurrentAttempt":{"integerValue":1,"valueType":"IntegerType"},"CurrentNotificationNumber":{"integerValue":0,"valueType":"IntegerType"},"ExecutionTime":{"doubleValue":0.003,"valueType":"DoubleType"},"LastNotificationTime":{"timeValue":"0","valueType":"TimeType"},"LastStateChange":{"timeValue":"1747683049000","valueType":"TimeType"},"Latency":{"doubleValue":0,"valueType":"DoubleType"},"MaxAttempts":{"integerValue":3,"valueType":"IntegerType"},"PercentStateChange":{"doubleValue":0,"valueType":"DoubleType"},"PerformanceData":{"stringValue":"rta=0.044ms;3000.000;5000.000;0; pl=0%;80;100;0;100 rtmax=0.044ms;;;; rtmin=0.044ms;;;;","valueType":"StringType"},"StateType":{"stringValue":"HARD","valueType":"StringType"},"isAcknowledged":{"boolValue":false,"valueType":"BooleanType"},"isChecksEnabled":{"boolValue":true,"valueType":"BooleanType"},"isEventHandlersEnabled":{"boolValue":true,"valueType":"BooleanType"},"isFlapDetectionEnabled":{"boolValue":true,"valueType":"BooleanType"},"isNotificationsEnabled":{"boolValue":true,"valueType":"BooleanType"},"isPassiveChecksEnabled":{"boolValue":true,"valueType":"BooleanType"}},"services":[{"lastCheckTime":"1747756060000","lastPluginOutput":"OK - 127.1.1.22 rta 0.031ms lost 0%","name":"icmp_ping_alive","nextCheckTime":"1747756120000","properties":{"CheckType":{"stringValue":"ACTIVE","valueType":"StringType"},"CurrentAttempt":{"integerValue":1,"valueType":"IntegerType"},"CurrentNotificationNumber":{"integerValue":0,"valueType":"IntegerType"},"ExecutionTime":{"doubleValue":0.003,"valueType":"DoubleType"},"LastNotificationTime":{"timeValue":"0","valueType":"TimeType"},"Latency":{"doubleValue":0,"valueType":"DoubleType"},"MaxAttempts":{"integerValue":3,"valueType":"IntegerType"},"PercentStateChange":{"doubleValue":0,"valueType":"DoubleType"},"PerformanceData":{"stringValue":"rta=0.031ms;3000.000;5000.000;0; pl=0%;80;100;0;100 rtmax=0.031ms;;;; rtmin=0.031ms;;;;","valueType":"StringType"},"StateType":{"stringValue":"HARD","valueType":"StringType"},"isAcceptPassiveChecks":{"boolValue":true,"valueType":"BooleanType"},"isChecksEnabled":{"boolValue":true,"valueType":"BooleanType"},"isEventHandlersEnabled":{"boolValue":true,"valueType":"BooleanType"},"isFlapDetectionEnabled":{"boolValue":true,"valueType":"BooleanType"},"isNotificationsEnabled":{"boolValue":true,"valueType":"BooleanType"},"isProblemAcknowledged":{"boolValue":false,"valueType":"BooleanType"}},"status":"SERVICE_OK","type":"service"}],"status":"HOST_UP","type":"host"}]}`),
		}
		mbb.Build(&buf, 1024)
		qq := make([]transit.ResourcesWithServicesRequest, 0)
		var q transit.ResourcesWithServicesRequest
		for _, p := range buf {
			q = transit.ResourcesWithServicesRequest{}
			assert.NoError(t, json.Unmarshal(p, &q))
			qq = append(qq, q)
		}
		assert.Equal(t, 2, len(qq))
		assert.Contains(t, qq[0].Context.TraceToken, "b491b98e-0000")
		assert.Contains(t, qq[1].Context.TraceToken, "b491b98e-0001")
		assert.Equal(t, 1, len(qq[0].Resources[0].Services))
		assert.Equal(t, 1, len(qq[1].Resources[0].Services))

		// bb, err := json.MarshalIndent(qq, "", "  ")
		// assert.NoError(t, err)
		// t.Logf("%#v\n\n%s", qq, bb)
	})

	t.Run("combine small", func(t *testing.T) {
		buf := [][]byte{
			[]byte(`{"context":{"agentId":"2a728b11-3359-49b2-8611-98854927af2c","appType":"NAGIOS","timeStamp":"1676911781735","traceToken":"2a728b11-3359-49b2-8611-98854927af2c","version":"1.0.0"},"groups":[{"groupName":"HHostGroup22","resources":[{"name":"a-test-2","type":"host"}],"type":"HostGroup"}],"resources":[{"name":"a-test-2","services":[{"lastCheckTime":"1676911781000","lastPluginOutput":"CRITICAL - load average: 83.57, 36.40, 18.21","metrics":[{"interval":{"endTime":"1676911781280","startTime":"1676911781181"},"metricName":"load1","sampleType":"Value","thresholds":[{"label":"load1_wn","sampleType":"Warning","value":{"doubleValue":5.0,"integerValue":0,"valueType":"DoubleType"}},{"label":"load1_cr","sampleType":"Critical","value":{"doubleValue":10.0,"integerValue":0,"valueType":"DoubleType"}},{"label":"load1_mn","sampleType":"Min","value":{"doubleValue":0.0,"integerValue":0,"valueType":"DoubleType"}}],"unit":"1","value":{"doubleValue":83.569999999999993,"integerValue":0,"valueType":"DoubleType"}},{"interval":{"endTime":"1676911781280","startTime":"1676911781181"},"metricName":"load5","sampleType":"Value","thresholds":[{"label":"load5_wn","sampleType":"Warning","value":{"doubleValue":4.0,"integerValue":0,"valueType":"DoubleType"}},{"label":"load5_cr","sampleType":"Critical","value":{"doubleValue":8.0,"integerValue":0,"valueType":"DoubleType"}},{"label":"load5_mn","sampleType":"Min","value":{"doubleValue":0.0,"integerValue":0,"valueType":"DoubleType"}}],"unit":"1","value":{"doubleValue":36.399999999999999,"integerValue":0,"valueType":"DoubleType"}},{"interval":{"endTime":"1676911781280","startTime":"1676911781181"},"metricName":"load15","sampleType":"Value","thresholds":[{"label":"load15_wn","sampleType":"Warning","value":{"doubleValue":3.0,"integerValue":0,"valueType":"DoubleType"}},{"label":"load15_cr","sampleType":"Critical","value":{"doubleValue":6.0,"integerValue":0,"valueType":"DoubleType"}},{"label":"load15_mn","sampleType":"Min","value":{"doubleValue":0.0,"integerValue":0,"valueType":"DoubleType"}}],"unit":"1","value":{"doubleValue":18.210000000000001,"integerValue":0,"valueType":"DoubleType"}}],"name":"local_load_6009","nextCheckTime":"1676912381000","properties":{"CheckType":{"doubleValue":0.0,"integerValue":0,"stringValue":"ACTIVE","valueType":"StringType"},"CurrentAttempt":{"doubleValue":0.0,"integerValue":3,"valueType":"IntegerType"},"CurrentNotificationNumber":{"doubleValue":0.0,"integerValue":0,"valueType":"IntegerType"},"ExecutionTime":{"doubleValue":0.098902000000000004,"integerValue":0,"valueType":"DoubleType"},"LastNotificationTime":{"doubleValue":0.0,"integerValue":0,"timeValue":"0","valueType":"TimeType"},"LastStateChange":{"doubleValue":0.0,"integerValue":0,"timeValue":"1676911144000","valueType":"TimeType"},"Latency":{"doubleValue":11.620201110839844,"integerValue":0,"valueType":"DoubleType"},"MaxAttempts":{"doubleValue":0.0,"integerValue":3,"valueType":"IntegerType"},"PercentStateChange":{"doubleValue":31.44736842105263,"integerValue":0,"valueType":"DoubleType"},"ScheduledDowntimeDepth":{"doubleValue":0.0,"integerValue":0,"valueType":"IntegerType"},"StateType":{"doubleValue":0.0,"integerValue":0,"stringValue":"HARD","valueType":"StringType"},"isAcceptPassiveChecks":{"boolValue":true,"doubleValue":0.0,"integerValue":0,"valueType":"BooleanType"},"isChecksEnabled":{"boolValue":true,"doubleValue":0.0,"integerValue":0,"valueType":"BooleanType"},"isEventHandlersEnabled":{"boolValue":true,"doubleValue":0.0,"integerValue":0,"valueType":"BooleanType"},"isFlapDetectionEnabled":{"boolValue":true,"doubleValue":0.0,"integerValue":0,"valueType":"BooleanType"},"isNotificationsEnabled":{"boolValue":true,"doubleValue":0.0,"integerValue":0,"valueType":"BooleanType"},"isProblemAcknowledged":{"doubleValue":0.0,"integerValue":0,"valueType":"BooleanType"}},"status":"SERVICE_UNSCHEDULED_CRITICAL","type":"hypervisor"}],"status":"HOST_UNCHANGED","type":"host"}]}`),
			[]byte(`{"context":{"agentId":"2a728b11-3359-49b2-8611-98854927af2c","appType":"NAGIOS","timeStamp":"1676911783375","traceToken":"2a728b11-3359-49b2-8611-98854927af2c","version":"1.0.0"},"groups":[{"groupName":"HHostGroup11","resources":[{"name":"a-test-1","type":"host"}],"type":"HostGroup"}],"resources":[{"name":"a-test-1","services":[{"lastCheckTime":"1676911783000","lastPluginOutput":"WARNING: Stale Status","name":"linux_load_996","nextCheckTime":"1676911771000","properties":{"CheckType":{"doubleValue":0.0,"integerValue":0,"stringValue":"ACTIVE","valueType":"StringType"},"CurrentAttempt":{"doubleValue":0.0,"integerValue":1,"valueType":"IntegerType"},"CurrentNotificationNumber":{"doubleValue":0.0,"integerValue":0,"valueType":"IntegerType"},"ExecutionTime":{"doubleValue":0.0060609999999999995,"integerValue":0,"valueType":"DoubleType"}},"status":"SERVICE_WARNING","type":"hypervisor"}],"status":"HOST_UNCHANGED","type":"host"}]}`),
			[]byte(`{"context":{"agentId":"2a728b11-3359-49b2-8611-98854927af2c","appType":"NAGIOS","timeStamp":"1676911783386","traceToken":"2a728b11-3359-49b2-8611-98854927af2c","version":"1.0.0"},"groups":[{"groupName":"HHostGroup11","resources":[{"name":"a-test-1","type":"host"}],"type":"HostGroup"}],"resources":[{"name":"a-test-1","services":[{"lastCheckTime":"1676911783000","lastPluginOutput":"WARNING: Stale Status","name":"linux_load_997","nextCheckTime":"1676911771000","properties":{"CheckType":{"doubleValue":0.0,"integerValue":0,"stringValue":"ACTIVE","valueType":"StringType"},"CurrentAttempt":{"doubleValue":0.0,"integerValue":1,"valueType":"IntegerType"},"CurrentNotificationNumber":{"doubleValue":0.0,"integerValue":0,"valueType":"IntegerType"},"ExecutionTime":{"doubleValue":0.0053230000000000005,"integerValue":0,"valueType":"DoubleType"}},"status":"SERVICE_WARNING","type":"hypervisor"}],"status":"HOST_UNCHANGED","type":"host"}]}`),
			[]byte(`{"context":{"agentId":"2a728b11-3359-49b2-8611-98854927af2c","appType":"NAGIOS","timeStamp":"1676911783397","traceToken":"2a728b11-3359-49b2-8611-98854927af2c","version":"1.0.0"},"groups":[{"groupName":"HHostGroup11","resources":[{"name":"a-test-1","type":"host"}],"type":"HostGroup"}],"resources":[{"name":"a-test-1","services":[{"lastCheckTime":"1676911783000","lastPluginOutput":"WARNING: Stale Status","name":"linux_load_998","nextCheckTime":"1676911771000","properties":{"CheckType":{"doubleValue":0.0,"integerValue":0,"stringValue":"ACTIVE","valueType":"StringType"},"CurrentAttempt":{"doubleValue":0.0,"integerValue":1,"valueType":"IntegerType"},"CurrentNotificationNumber":{"doubleValue":0.0,"integerValue":0,"valueType":"IntegerType"},"ExecutionTime":{"doubleValue":0.0064409999999999997,"integerValue":0,"valueType":"DoubleType"}},"status":"SERVICE_WARNING","type":"hypervisor"}],"status":"HOST_UNCHANGED","type":"host"}]}`),
		}

		printMemStats()
		mbb.Build(&buf, 1024)
		printMemStats()

		qq := make([]transit.ResourcesWithServicesRequest, 0)
		var q transit.ResourcesWithServicesRequest
		for _, p := range buf {
			// t.Logf("\n\n%s\n", p)
			q = transit.ResourcesWithServicesRequest{}
			assert.NoError(t, json.Unmarshal(p, &q))
			qq = append(qq, q)
		}
		assert.Equal(t, 3, len(qq))
		assert.Equal(t, 1, len(qq[0].Groups))
		assert.Equal(t, "local_load_6009", qq[0].Resources[0].Services[0].Name)
		assert.Equal(t, "linux_load_996", qq[1].Resources[0].Services[0].Name)
		assert.Equal(t, "linux_load_997", qq[1].Resources[1].Services[0].Name)
		assert.Equal(t, "linux_load_998", qq[2].Resources[0].Services[0].Name)
	})

	t.Run("status change", func(t *testing.T) {
		buf := [][]byte{
			[]byte(`{"context":{"agentId":"37be8b86-3b0f-4408-9a9c-f444d42dcec9","appType":"NAGIOS","timeStamp":"1702659254261","traceToken":"***","version":"1.0.0"},"resources":[{"name":"a1test5","services":[{"lastCheckTime":"1702659254000","lastPluginOutput":"OK - a1test5 rta 0.158ms lost 0%","metrics":[{"interval":{"endTime":"1702659254261","startTime":"1702659254256"},"metricName":"rta","sampleType":"Value","thresholds":[{"label":"rta_wn","sampleType":"Warning","value":{"doubleValue":3000.0,"integerValue":0,"valueType":"DoubleType"}},{"label":"rta_cr","sampleType":"Critical","value":{"doubleValue":5000.0,"integerValue":0,"valueType":"DoubleType"}},{"label":"rta_mn","sampleType":"Min","value":{"doubleValue":0.0,"integerValue":0,"valueType":"DoubleType"}}],"value":{"doubleValue":0.158,"integerValue":0,"valueType":"DoubleType"}},{"interval":{"endTime":"1702659254261","startTime":"1702659254256"},"metricName":"pl","sampleType":"Value","thresholds":[{"label":"pl_wn","sampleType":"Warning","value":{"doubleValue":80.0,"integerValue":0,"valueType":"DoubleType"}},{"label":"pl_cr","sampleType":"Critical","value":{"doubleValue":100.0,"integerValue":0,"valueType":"DoubleType"}},{"label":"pl_mn","sampleType":"Min","value":{"doubleValue":0.0,"integerValue":0,"valueType":"DoubleType"}},{"label":"pl_mx","sampleType":"Max","value":{"doubleValue":100.0,"integerValue":0,"valueType":"DoubleType"}}],"value":{"doubleValue":0.0,"integerValue":0,"valueType":"DoubleType"}},{"interval":{"endTime":"1702659254261","startTime":"1702659254256"},"metricName":"rtmax","sampleType":"Value","value":{"doubleValue":0.158,"integerValue":0,"valueType":"DoubleType"}},{"interval":{"endTime":"1702659254261","startTime":"1702659254256"},"metricName":"rtmin","sampleType":"Value","value":{"doubleValue":0.158,"integerValue":0,"valueType":"DoubleType"}}],"name":"icmp_ping_alive","nextCheckTime":"1702659314000","properties":{"CheckType":{"doubleValue":0.0,"integerValue":0,"stringValue":"ACTIVE","valueType":"StringType"},"CurrentAttempt":{"doubleValue":0.0,"integerValue":1,"valueType":"IntegerType"},"CurrentNotificationNumber":{"doubleValue":0.0,"integerValue":0,"valueType":"IntegerType"},"ExecutionTime":{"doubleValue":0.0049349999999999993,"integerValue":0,"valueType":"DoubleType"},"LastNotificationTime":{"doubleValue":0.0,"integerValue":0,"timeValue":"0","valueType":"TimeType"},"LastStateChange":{"doubleValue":0.0,"integerValue":0,"timeValue":"1702659254000","valueType":"TimeType"},"Latency":{"doubleValue":0.00073600001633167267,"integerValue":0,"valueType":"DoubleType"},"MaxAttempts":{"doubleValue":0.0,"integerValue":3,"valueType":"IntegerType"},"PercentStateChange":{"doubleValue":6.25,"integerValue":0,"valueType":"DoubleType"},"ScheduledDowntimeDepth":{"doubleValue":0.0,"integerValue":0,"valueType":"IntegerType"},"StateType":{"doubleValue":0.0,"integerValue":0,"stringValue":"HARD","valueType":"StringType"},"isAcceptPassiveChecks":{"boolValue":true,"doubleValue":0.0,"integerValue":0,"valueType":"BooleanType"},"isChecksEnabled":{"boolValue":true,"doubleValue":0.0,"integerValue":0,"valueType":"BooleanType"},"isEventHandlersEnabled":{"boolValue":true,"doubleValue":0.0,"integerValue":0,"valueType":"BooleanType"},"isFlapDetectionEnabled":{"boolValue":true,"doubleValue":0.0,"integerValue":0,"valueType":"BooleanType"},"isNotificationsEnabled":{"boolValue":true,"doubleValue":0.0,"integerValue":0,"valueType":"BooleanType"},"isProblemAcknowledged":{"doubleValue":0.0,"integerValue":0,"valueType":"BooleanType"}},"status":"SERVICE_OK","type":"hypervisor"}],"status":"HOST_UNCHANGED","type":"host"}]}`),
			[]byte(`{"context":{"agentId":"37be8b86-3b0f-4408-9a9c-f444d42dcec9","appType":"NAGIOS","timeStamp":"1702659254268","traceToken":"***","version":"1.0.0"},"resources":[{"lastCheckTime":"1702659254000","lastPluginOutput":"OK - a1test5 rta 0.069ms lost 0%","name":"a1test5","nextCheckTime":"1702659314000","properties":{"Alias":{"doubleValue":0.0,"integerValue":0,"stringValue":"a1test5","valueType":"StringType"},"CheckType":{"doubleValue":0.0,"integerValue":0,"stringValue":"ACTIVE","valueType":"StringType"},"CurrentAttempt":{"doubleValue":0.0,"integerValue":1,"valueType":"IntegerType"},"CurrentNotificationNumber":{"doubleValue":0.0,"integerValue":0,"valueType":"IntegerType"},"ExecutionTime":{"doubleValue":0.0052100000000000002,"integerValue":0,"valueType":"DoubleType"},"LastNotificationTime":{"doubleValue":0.0,"integerValue":0,"timeValue":"0","valueType":"TimeType"},"LastStateChange":{"doubleValue":0.0,"integerValue":0,"timeValue":"1702659254000","valueType":"TimeType"},"Latency":{"doubleValue":0.00080799998249858618,"integerValue":0,"valueType":"DoubleType"},"MaxAttempts":{"doubleValue":0.0,"integerValue":3,"valueType":"IntegerType"},"PercentStateChange":{"doubleValue":6.25,"integerValue":0,"valueType":"DoubleType"},"PerformanceData":{"doubleValue":0.0,"integerValue":0,"stringValue":"rta=0.069ms;3000.000;5000.000;0; pl=0%;80;100;0;100 rtmax=0.069ms;;;; rtmin=0.069ms;;;;","valueType":"StringType"},"ScheduledDowntimeDepth":{"doubleValue":0.0,"integerValue":0,"valueType":"IntegerType"},"StateType":{"doubleValue":0.0,"integerValue":0,"stringValue":"HARD","valueType":"StringType"},"isAcknowledged":{"doubleValue":0.0,"integerValue":0,"valueType":"BooleanType"},"isChecksEnabled":{"boolValue":true,"doubleValue":0.0,"integerValue":0,"valueType":"BooleanType"},"isEventHandlersEnabled":{"boolValue":true,"doubleValue":0.0,"integerValue":0,"valueType":"BooleanType"},"isFlapDetectionEnabled":{"boolValue":true,"doubleValue":0.0,"integerValue":0,"valueType":"BooleanType"},"isNotificationsEnabled":{"boolValue":true,"doubleValue":0.0,"integerValue":0,"valueType":"BooleanType"},"isPassiveChecksEnabled":{"boolValue":true,"doubleValue":0.0,"integerValue":0,"valueType":"BooleanType"}},"status":"HOST_UP","type":"host"}]}`),
			[]byte(`{"context":{"agentId":"37be8b86-3b0f-4408-9a9c-f444d42dcec9","appType":"NAGIOS","timeStamp":"1702659374293","traceToken":"***","version":"1.0.0"},"resources":[{"name":"a1test5","services":[{"lastCheckTime":"1702659374000","lastPluginOutput":"check_icmp: Failed to resolve a1test5: Name or service not known","name":"icmp_ping_alive","nextCheckTime":"1702659434000","properties":{"CheckType":{"doubleValue":0.0,"integerValue":0,"stringValue":"ACTIVE","valueType":"StringType"},"CurrentAttempt":{"doubleValue":0.0,"integerValue":1,"valueType":"IntegerType"},"CurrentNotificationNumber":{"doubleValue":0.0,"integerValue":0,"valueType":"IntegerType"},"ExecutionTime":{"doubleValue":0.024393000000000001,"integerValue":0,"valueType":"DoubleType"},"LastNotificationTime":{"doubleValue":0.0,"integerValue":0,"timeValue":"0","valueType":"TimeType"},"LastStateChange":{"doubleValue":0.0,"integerValue":0,"timeValue":"1702659254000","valueType":"TimeType"},"Latency":{"doubleValue":0.00048600000445730984,"integerValue":0,"valueType":"DoubleType"},"MaxAttempts":{"doubleValue":0.0,"integerValue":3,"valueType":"IntegerType"},"PercentStateChange":{"doubleValue":6.1184210526315788,"integerValue":0,"valueType":"DoubleType"},"ScheduledDowntimeDepth":{"doubleValue":0.0,"integerValue":0,"valueType":"IntegerType"},"StateType":{"doubleValue":0.0,"integerValue":0,"stringValue":"SOFT","valueType":"StringType"},"isAcceptPassiveChecks":{"boolValue":true,"doubleValue":0.0,"integerValue":0,"valueType":"BooleanType"},"isChecksEnabled":{"boolValue":true,"doubleValue":0.0,"integerValue":0,"valueType":"BooleanType"},"isEventHandlersEnabled":{"boolValue":true,"doubleValue":0.0,"integerValue":0,"valueType":"BooleanType"},"isFlapDetectionEnabled":{"boolValue":true,"doubleValue":0.0,"integerValue":0,"valueType":"BooleanType"},"isNotificationsEnabled":{"boolValue":true,"doubleValue":0.0,"integerValue":0,"valueType":"BooleanType"},"isProblemAcknowledged":{"doubleValue":0.0,"integerValue":0,"valueType":"BooleanType"}},"status":"SERVICE_OK","type":"hypervisor"}],"status":"HOST_UNCHANGED","type":"host"}]}`),
			[]byte(`{"context":{"agentId":"37be8b86-3b0f-4408-9a9c-f444d42dcec9","appType":"NAGIOS","timeStamp":"1702659377307","traceToken":"***","version":"1.0.0"},"resources":[{"name":"a1test5","services":[{"lastCheckTime":"1702659377000","lastPluginOutput":"check_icmp: Failed to resolve a1test5: Name or service not known","name":"icmp_ping","nextCheckTime":"1702659437000","properties":{"CheckType":{"doubleValue":0.0,"integerValue":0,"stringValue":"ACTIVE","valueType":"StringType"},"CurrentAttempt":{"doubleValue":0.0,"integerValue":1,"valueType":"IntegerType"},"CurrentNotificationNumber":{"doubleValue":0.0,"integerValue":0,"valueType":"IntegerType"},"ExecutionTime":{"doubleValue":0.022239000000000002,"integerValue":0,"valueType":"DoubleType"},"LastNotificationTime":{"doubleValue":0.0,"integerValue":0,"timeValue":"0","valueType":"TimeType"},"LastStateChange":{"doubleValue":0.0,"integerValue":0,"timeValue":"1702659377000","valueType":"TimeType"},"Latency":{"doubleValue":0.0010870000114664435,"integerValue":0,"valueType":"DoubleType"},"MaxAttempts":{"doubleValue":0.0,"integerValue":3,"valueType":"IntegerType"},"PercentStateChange":{"doubleValue":12.236842105263159,"integerValue":0,"valueType":"DoubleType"},"ScheduledDowntimeDepth":{"doubleValue":0.0,"integerValue":0,"valueType":"IntegerType"},"StateType":{"doubleValue":0.0,"integerValue":0,"stringValue":"HARD","valueType":"StringType"},"isAcceptPassiveChecks":{"boolValue":true,"doubleValue":0.0,"integerValue":0,"valueType":"BooleanType"},"isChecksEnabled":{"boolValue":true,"doubleValue":0.0,"integerValue":0,"valueType":"BooleanType"},"isEventHandlersEnabled":{"boolValue":true,"doubleValue":0.0,"integerValue":0,"valueType":"BooleanType"},"isFlapDetectionEnabled":{"boolValue":true,"doubleValue":0.0,"integerValue":0,"valueType":"BooleanType"},"isNotificationsEnabled":{"boolValue":true,"doubleValue":0.0,"integerValue":0,"valueType":"BooleanType"},"isProblemAcknowledged":{"doubleValue":0.0,"integerValue":0,"valueType":"BooleanType"}},"status":"SERVICE_UNKNOWN","type":"hypervisor"}],"status":"HOST_UNCHANGED","type":"host"}]}`),
			[]byte(`{"context":{"agentId":"37be8b86-3b0f-4408-9a9c-f444d42dcec9","appType":"NAGIOS","timeStamp":"1702659494342","traceToken":"***","version":"1.0.0"},"resources":[{"lastCheckTime":"1702659494000","lastPluginOutput":"check_icmp: Failed to resolve a1test5: Name or service not known","name":"a1test5","nextCheckTime":"1702659554000","properties":{"Alias":{"doubleValue":0.0,"integerValue":0,"stringValue":"a1test5","valueType":"StringType"},"CheckType":{"doubleValue":0.0,"integerValue":0,"stringValue":"ACTIVE","valueType":"StringType"},"CurrentAttempt":{"doubleValue":0.0,"integerValue":3,"valueType":"IntegerType"},"CurrentNotificationNumber":{"doubleValue":0.0,"integerValue":0,"valueType":"IntegerType"},"ExecutionTime":{"doubleValue":0.022151000000000001,"integerValue":0,"valueType":"DoubleType"},"LastNotificationTime":{"doubleValue":0.0,"integerValue":0,"timeValue":"0","valueType":"TimeType"},"LastStateChange":{"doubleValue":0.0,"integerValue":0,"timeValue":"1702659374000","valueType":"TimeType"},"Latency":{"doubleValue":5.9000001783715561e-5,"integerValue":0,"valueType":"DoubleType"},"MaxAttempts":{"doubleValue":0.0,"integerValue":3,"valueType":"IntegerType"},"PercentStateChange":{"doubleValue":10.394736842105264,"integerValue":0,"valueType":"DoubleType"},"ScheduledDowntimeDepth":{"doubleValue":0.0,"integerValue":0,"valueType":"IntegerType"},"StateType":{"doubleValue":0.0,"integerValue":0,"stringValue":"HARD","valueType":"StringType"},"isAcknowledged":{"doubleValue":0.0,"integerValue":0,"valueType":"BooleanType"},"isChecksEnabled":{"boolValue":true,"doubleValue":0.0,"integerValue":0,"valueType":"BooleanType"},"isEventHandlersEnabled":{"boolValue":true,"doubleValue":0.0,"integerValue":0,"valueType":"BooleanType"},"isFlapDetectionEnabled":{"boolValue":true,"doubleValue":0.0,"integerValue":0,"valueType":"BooleanType"},"isNotificationsEnabled":{"boolValue":true,"doubleValue":0.0,"integerValue":0,"valueType":"BooleanType"},"isPassiveChecksEnabled":{"boolValue":true,"doubleValue":0.0,"integerValue":0,"valueType":"BooleanType"}},"status":"HOST_UNSCHEDULED_DOWN","type":"host"}]}`),
		}
		mbb.Build(&buf, 10240)
		qq := make([]transit.ResourcesWithServicesRequest, 0)
		var q transit.ResourcesWithServicesRequest
		for _, p := range buf {
			// t.Logf("\n\n%s\n", p)
			q = transit.ResourcesWithServicesRequest{}
			assert.NoError(t, json.Unmarshal(p, &q))
			qq = append(qq, q)
		}
		assert.Equal(t, 4, len(qq))
		assert.Equal(t, 0, len(qq[0].Groups))
		assert.Equal(t, transit.HostUnchanged, qq[0].Resources[0].Status)
		assert.Equal(t, transit.HostUp, qq[1].Resources[0].Status)
		assert.Equal(t, transit.HostUnchanged, qq[2].Resources[0].Status)
		assert.Equal(t, transit.HostUnchanged, qq[2].Resources[1].Status)
		assert.Equal(t, transit.HostUnscheduledDown, qq[3].Resources[0].Status)
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

// inspired by expvar.Handler() implementation
func memstats() any {
	stats := new(runtime.MemStats)
	runtime.ReadMemStats(stats)
	return *stats
}
func printMemStats() {
	println("\n~", time.Now().Format(time.DateTime), "MEM_STATS", fmt.Sprintf("%+v", memstats()))
}
