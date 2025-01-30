//go:build !codeanalysis

package parser

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNscaParser(t *testing.T) {
	data := []byte(
		`Server1;Disks1;0;CRITICAL - load average: 2.45, 2.32, 2.22|load1=2.450;0.000;0.000;0;
Server2;Disks2;1;CRITICAL - load average: 2.45, 2.32, 2.22|load1=2.450;0.000;0.000;0; load5=2.320;0.000;0.000;0; load15=2.220;0.000;0.000;0;
Server3;Disks3;2;CRITICAL - load average: 2.45, 2.32, 2.22|load1=2.450;0.000;0.000;0; load5=2.320;0.000;0.000;0;
awips-demo-4;example-service-10;0;OK - example-service-10 (2021-08-09 15:47:38 :: 1628524058) | result=147ms;;;0;`,
	)

	monitoredResources, err := Parse(data, NSCA)
	assert.NoError(t, err)

	assert.Equal(t, 4, len(*monitoredResources), "invalid count of monitored resources")

	for _, res := range *monitoredResources {
		switch res.Name {
		case "Server1":
			assert.Equal(t, 1, len(res.Services), "invalid count of services for monitored resource")
			assert.Equal(t, "Disks1", res.Services[0].Name, "invalid service in monitored resources")
			assert.Equal(t, 1, len(res.Services[0].Metrics), "invalid count of metrics for service")
		case "Server2":
			assert.Equal(t, 1, len(res.Services), "invalid count of services for monitored resource")
			assert.Equal(t, "Disks2", res.Services[0].Name, "invalid service in monitored resources")
			assert.Equal(t, 3, len(res.Services[0].Metrics), "invalid count of metrics for service")
		case "Server3":
			assert.Equal(t, 1, len(res.Services), "invalid count of services for monitored resource")
			assert.Equal(t, "Disks3", res.Services[0].Name, "invalid service in monitored resources")
			assert.Equal(t, 2, len(res.Services[0].Metrics), "invalid count of metrics for service")
		case "awips-demo-4":
			assert.Equal(t, 1, len(res.Services), "invalid count of services for monitored resource")
			assert.Equal(t, "example-service-10", res.Services[0].Name, "invalid service in monitored resources")
			assert.Equal(t, 1, len(res.Services[0].Metrics), "invalid count of metrics for service")
		default:
			assert.Fail(t, "invalid service in monitored resources")
		}
	}
}

func TestBronxParser(t *testing.T) {
	data := []byte(
		`S;1596719076;Server1;Disk1;0;OK| test-metric=44;85;95;
S;1596719076;Server2;Disk2;1;WARNING| test-metric=44;85;95;
S;1596719076;Server3;Disk3;2;CRITICAL| test-metric=44;85;95;
S;1628530909;awips-demo-2;example-service-2;0;OK - example-service-2 (2021-08-09 17:41:49 :: 1628530909) | result=63ms;;;0;
S;1628530909;awips-demo-2;example-service-3;0;OK - example-service-3 (2021-08-09 17:41:49 :: 1628530909) | result=210ms;;;0;
S;1628530909;awips-demo-2;example-service-4;0;OK - example-service-4 (2021-08-09 17:41:49 :: 1628530909) | result=126ms;;;0;
S;1628530909;awips-demo-2;example-service-5;0;OK - example-service-5 (2021-08-09 17:41:49 :: 1628530909) | result=42ms;;;0;
S;1628530909;awips-demo-2;example-service-6;0;OK - example-service-6 (2021-08-09 17:41:49 :: 1628530909) | result=42ms;;;0;
S;1628530909;awips-demo-2;example-service-7;0;OK - example-service-7 (2021-08-09 17:41:49 :: 1628530909) | result=63ms;;;0;`,
	)

	monitoredResources, err := Parse(data, Bronx)
	assert.NoError(t, err)

	assert.Equal(t, 4, len(*monitoredResources), "invalid count of monitored resources")

	for _, res := range *monitoredResources {
		switch res.Name {
		case "Server1":
			assert.Equal(t, 1, len(res.Services), "invalid count of services for monitored resource")
			assert.Equal(t, "Disk1", res.Services[0].Name, "invalid service in monitored resources")
			assert.Equal(t, "test-metric", res.Services[0].Metrics[0].MetricName, "invalid metric name")
		case "Server2":
			assert.Equal(t, 1, len(res.Services), "invalid count of services for monitored resource")
			assert.Equal(t, "Disk2", res.Services[0].Name, "invalid service in monitored resources")
		case "Server3":
			assert.Equal(t, 1, len(res.Services), "invalid count of services for monitored resource")
			assert.Equal(t, "Disk3", res.Services[0].Name, "invalid service in monitored resources")
		case "awips-demo-2":
			assert.Equal(t, 6, len(res.Services), "invalid count of services for monitored resource")
			assert.Equal(t, "example-service-2", res.Services[0].Name, "invalid service in monitored resources")
			assert.Equal(t, 1, len(res.Services[0].Metrics), "invalid count of metrics for service")
		default:
			assert.Fail(t, "invalid service in monitored resources")
		}
	}
}

func TestBronxParser2(t *testing.T) {
	data := []byte(
`H;1628546296;awips-demo-2;0;UP - awips-demo-2 (2021-08-09 21:58:16 :: 1628546296) | result=21ms;;;0;
S;1628546296;awips-demo-2;example-service;0;OK - example-service (2021-08-09 21:58:16 :: 1628546296) | result=168ms;;;0;
S;1628546296;awips-demo-2;example-service-1;0;OK - example-service-1 (2021-08-09 21:58:16 :: 1628546296) | result=168ms;;;0;
S;1628546296;awips-demo-2;example-service-2;0;OK - example-service-2 (2021-08-09 21:58:16 :: 1628546296) | result=84ms;;;0;
S;1628546296;awips-demo-2;example-service-3;0;OK - example-service-3 (2021-08-09 21:58:16 :: 1628546296) | result=42ms;;;0;
S;1628546296;awips-demo-2;example-service-4;0;OK - example-service-4 (2021-08-09 21:58:16 :: 1628546296) | result=147ms;;;0;
S;1628546296;awips-demo-2;example-service-5;0;OK - example-service-5 (2021-08-09 21:58:16 :: 1628546296) | result=63ms;;;0;`,
	)

	monitoredResources, err := Parse(data, Bronx)
	assert.NoError(t, err)

	assert.Equal(t, 1, len(*monitoredResources), "invalid count of monitored resources")

	for _, res := range *monitoredResources {
		switch res.Name {
		case "awips-demo-2":
			assert.Equal(t, 6, len(res.Services), "invalid count of services for monitored resource")
			assert.Equal(t, 1, len(res.Services[0].Metrics), "invalid count of metrics for service")
		default:
			assert.Fail(t, "invalid service in monitored resources")
		}
	}
}
