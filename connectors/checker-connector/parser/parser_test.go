package parser

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNscaParser(t *testing.T) {
	data := []byte(
		`Server1;Disks1;0;CRITICAL - load average: 2.45, 2.32, 2.22|load1=2.450;0.000;0.000;0;
Server2;Disks2;1;CRITICAL - load average: 2.45, 2.32, 2.22|load1=2.450;0.000;0.000;0; load5=2.320;0.000;0.000;0; load15=2.220;0.000;0.000;0;
Server3;Disks3;2;CRITICAL - load average: 2.45, 2.32, 2.22|load1=2.450;0.000;0.000;0; load5=2.320;0.000;0.000;0;`,
	)

	monitoredResources, err := parse(data, NSCA)
	assert.NoError(t, err)

	if len(*monitoredResources) != 3 {
		assert.Fail(t, "invalid count of monitored resources")
	}

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
		default:
			assert.Fail(t, "invalid service in monitored resources")
		}
	}
}

func TestBronxParser(t *testing.T) {
	data := []byte(
		`S;1596719076;Server1;Disk1;0;OK| test-metric=44;85;95
S;1596719076;Server2;Disk2;1;WARNING| test-metric=44;85;95
S;1596719076;Server3;Disk3;2;CRITICAL| test-metric=44;85;95`,
	)

	monitoredResources, err := parse(data, Bronx)
	assert.NoError(t, err)

	if len(*monitoredResources) != 3 {
		assert.Fail(t, "invalid count of monitored resources")
	}

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
		default:
			assert.Fail(t, "invalid service in monitored resources")
		}
	}
}
