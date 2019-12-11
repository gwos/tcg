package integration

import (
	"encoding/json"
	"fmt"
	. "github.com/gwos/tng/config"
	"github.com/gwos/tng/log"
	"github.com/gwos/tng/milliseconds"
	"github.com/gwos/tng/nats"
	"github.com/gwos/tng/services"
	"github.com/gwos/tng/transit"
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"testing"
	"time"
)

const (
	TestMessagesCount                  = 3
	PerformanceServicesCount           = 5
	PerformanceResourcesCount          = 1000
	TestAppType                        = "VEMA"
	TestAgentID                        = "3939333393342"
	TestTraceToken                     = "token-99e93"
	GWAccount                          = "RESTAPIACCESS"
	GWPassword                         = "***REMOVED***"
	GWAppName                          = "tng"
	GWValidHost                        = "localhost:80"
	GWInvalidHost                      = "localhost:23"
	TestTngAgentConfigNatsFileStoreDir = "test_datastore"
)

// Test for ensuring that all data is stored in NATS and later resent
// if Groundwork Foundation is unavailable
func TestNatsQueue_1(t *testing.T) {
	assert.NoError(t, configNats(t, 5))
	defer cleanNats(t)
	log.Info("Config has invalid path to Groundwork Foundation, messages will be stored in a datastore:")
	GetConfig().GWConfigs[0].Host = GWInvalidHost

	testMessage, err := parseJSON()
	assert.NoError(t, err)

	for i := 0; i < TestMessagesCount; i++ {
		assert.NoError(t, nats.Publish(services.SubjSendResourceWithMetrics, testMessage))
		time.Sleep(1 * time.Second)
	}

	if services.GetTransitService().Stats().MessagesSent != 0 {
		t.Errorf("Messages shouldn't be delivered, because Groundwork entrypoint is invalid. deliveredCount = %d, want = %d",
			services.GetTransitService().Stats().MessagesSent, 0)
		return
	}

	GetConfig().GWConfigs[0].Host = GWValidHost
	log.Info("Invalid path was changed to valid one")

	time.Sleep(TestMessagesCount * 2 * time.Second)

	if services.GetTransitService().Stats().MessagesSent == 0 {
		t.Errorf("Messages should be delivered, because Groundwork entrypoint is valid. deliveredCount = %d, want = %s",
			services.GetTransitService().Stats().MessagesSent, "'>0'")
	}
}

// Test for ensuring that all data is stored in NATS and later resent
// after NATS streaming server restarting
func TestNatsQueue_2(t *testing.T) {
	assert.NoError(t, configNats(t, 5))

	log.Info("Config has invalid path to Groundwork Foundation, messages will be stored in a datastore:")
	GetConfig().GWConfigs[0].Host = GWInvalidHost

	defer cleanNats(t)

	testMessage, err := parseJSON()
	assert.NoError(t, err)

	for i := 0; i < TestMessagesCount; i++ {
		assert.NoError(t, nats.Publish(services.SubjSendResourceWithMetrics, testMessage))
		time.Sleep(1 * time.Second)
	}

	if services.GetTransitService().Stats().MessagesSent != 0 {
		t.Errorf("Messages shouldn't be delivered, because Groundwork entrypoint is invalid. deliveredCount = %d, want = %d",
			services.GetTransitService().Stats().MessagesSent, 0)
		return
	}

	log.Info("Stopping NATS server ...")
	assert.NoError(t, services.GetTransitService().StopNats())
	log.Info("NATS Server was stopped successfully")

	GetConfig().GWConfigs[0].Host = GWValidHost
	log.Info("Invalid path was changed to valid one")

	log.Info("Starting NATS server ...")
	assert.NoError(t, services.GetTransitService().StartNats())
	assert.NoError(t, services.GetTransitService().StartTransport())

	log.Info("NATS Server was started successfully")
	time.Sleep(TestMessagesCount * 1 * time.Second)

	if services.GetTransitService().Stats().MessagesSent == 0 {
		t.Errorf("Messages should be delivered, because Groundwork entrypoint is valid. deliveredCount = %d, want = %s",
			services.GetTransitService().Stats().MessagesSent, "'>0'")
	}
}

//Test NATS performance
func TestNatsPerformance(t *testing.T) {
	assert.NoError(t, configNats(t, 30))
	defer cleanNats(t)

	var resources []transit.MonitoredResource

	for i := 0; i < PerformanceResourcesCount; i++ {
		resource := transit.MonitoredResource{
			Name:          TestHostName + string(i),
			Type:          transit.Host,
			Status:        transit.HostUp,
			LastCheckTime: milliseconds.MillisecondTimestamp{Time: time.Now()},
			NextCheckTime: milliseconds.MillisecondTimestamp{Time: time.Now()},
			Services:      []transit.MonitoredService{},
		}

		for j := 0; j < PerformanceServicesCount; j++ {
			resource.Services = append(resource.Services, transit.MonitoredService{
				Name:          fmt.Sprintf("%s_%d_%s", TestHostName, i, "SERVICE"),
				Status:        transit.ServiceOk,
				Owner:         TestHostName + string(i),
				LastCheckTime: milliseconds.MillisecondTimestamp{Time: time.Now()},
				NextCheckTime: milliseconds.MillisecondTimestamp{Time: time.Now()},
				Metrics: []transit.TimeSeries{
					{
						MetricName: "Test",
						SampleType: transit.Value,
						Interval: &transit.TimeInterval{
							EndTime:   milliseconds.MillisecondTimestamp{Time: time.Now()},
							StartTime: milliseconds.MillisecondTimestamp{Time: time.Now()},
						},
						Value: &transit.TypedValue{
							ValueType:    transit.IntegerType,
							IntegerValue: 1000,
						},
						Unit: transit.MB,
					},
				},
			})
		}
		resources = append(resources, resource)
	}

	for _, res := range resources {
		request := transit.ResourcesWithServicesRequest{
			Context: transit.TracerContext{
				AppType:    TestAppType,
				AgentID:    TestAgentID,
				TraceToken: TestTraceToken,
				TimeStamp:  milliseconds.MillisecondTimestamp{Time: time.Now()},
			},
			Resources: []transit.MonitoredResource{res},
		}
		jsonBytes, err := json.Marshal(request)
		assert.NoError(t, err)
		assert.NoError(t, services.GetTransitService().SendResourceWithMetrics(jsonBytes))
	}

	time.Sleep(1 * time.Minute)

	if services.GetTransitService().Stats().MessagesSent != PerformanceResourcesCount {
		t.Errorf("Messages should be delivered. deliveredCount = %d, want = %s",
			services.GetTransitService().Stats().MessagesSent, "'1000'")
	}
}

func configNats(t *testing.T, natsAckWait int64) error {
	assert.NoError(t, os.Setenv(ConfigEnv, path.Join("..", ConfigName)))

	GetConfig().GWConfigs = []*GWConfig{
		{
			Host:     GWValidHost,
			Account:  GWAccount,
			Password: GWPassword,
			AppName:  GWAppName,
		},
	}

	service := services.GetTransitService()

	service.AgentConfig.NatsFilestoreDir = TestTngAgentConfigNatsFileStoreDir
	service.AgentConfig.NatsAckWait = natsAckWait

	assert.NoError(t, service.StartNats())
	assert.NoError(t, service.StartTransport())

	return nil
}

func cleanNats(t *testing.T) {
	services.GetTransitService().Stats().MessagesSent = 0

	assert.NoError(t, services.GetTransitService().StopNats())

	cmd := exec.Command("rm", "-rf", "test_datastore")
	_, err := cmd.Output()
	assert.NoError(t, err)
}

func parseJSON() ([]byte, error) {
	jsonFile, err := os.Open("fixtures/sendResourceWithMetrics.json")
	if err != nil {
		return nil, err
	}
	defer func() { _ = jsonFile.Close() }()

	byteValue, err := ioutil.ReadAll(jsonFile)
	if err != nil {
		return nil, err
	}

	return byteValue, nil
}
