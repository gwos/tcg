package integration

import (
	"github.com/gwos/tng/nats"
	"github.com/gwos/tng/services"
	"github.com/stretchr/testify/assert"
	"log"
	"os"
	"os/exec"
	"path"
	"testing"
	"time"
)

const (
	TestMessagesCount                  = 3
	TestNatsAckWait                    = 5
	GWValidHost                        = "localhost:80"
	GWInvalidHost                      = "localhost:23"
	TestTngAgentConfigNatsFileStoreDir = "test_datastore"
)

// Test for ensuring that all data is stored in NATS and later resent
// if Groundwork Foundation is unavailable
func TestNatsQueue_1(t *testing.T) {
	err := configNats(t)
	defer cleanNats(t)
	if err != nil {
		t.Error(err)
		return
	}
	log.Println("Config have invalid path to Groundwork Foundation, messages will be stored in a datastore:")
	services.GetTransitService().Config.GWConfig.Host = GWInvalidHost

	for i := 0; i < TestMessagesCount; i++ {
		err := nats.Publish(services.SubjSendResourceWithMetrics, []byte(testMessage))
		if err != nil {
			t.Error(err)
			return
		}
		time.Sleep(1 * time.Second)
	}

	if services.GetTransitService().Stats().MessagesSent != 0 {
		t.Errorf("Messages shouldn't be delivered, because Groundwork entrypoint is invalid. deliveredCount = %d, want = %d",
			services.GetTransitService().Stats().MessagesSent, 0)
		return
	}

	services.GetTransitService().Config.GWConfig.Host = GWValidHost
	log.Println("Invalid path was changed to valid one")

	time.Sleep(TestMessagesCount * 2 * time.Second)

	if services.GetTransitService().Stats().MessagesSent == 0 {
		t.Errorf("Messages should be delivered, because Groundwork entrypoint is valid. deliveredCount = %d, want = %s",
			services.GetTransitService().Stats().MessagesSent, "'>0'")
	}
}

// Test for ensuring that all data is stored in NATS and later resent
// after NATS streaming server restarting
func TestNatsQueue_2(t *testing.T) {
	err := configNats(t)
	if err != nil {
		t.Error(err)
		return
	}

	log.Println("Config have invalid path to Groundwork Foundation, messages will be stored in a datastore:")
	services.GetTransitService().Config.GWConfig.Host = GWInvalidHost

	defer cleanNats(t)

	for i := 0; i < TestMessagesCount; i++ {
		err := nats.Publish(services.SubjSendResourceWithMetrics, []byte(testMessage))
		if err != nil {
			t.Error(err)
		}
		time.Sleep(1 * time.Second)
	}

	if services.GetTransitService().Stats().MessagesSent != 0 {
		t.Errorf("Messages shouldn't be delivered, because Groundwork entrypoint is invalid. deliveredCount = %d, want = %d",
			services.GetTransitService().Stats().MessagesSent, 0)
		return
	}

	log.Println("Stopping NATS server ...")
	err = services.GetTransitService().StopNats()
	if err != nil {
		t.Error(err)
		return
	}
	log.Println("NATS Server was stopped successfully")

	services.GetTransitService().Config.GWConfig.Host = GWValidHost
	log.Println("Invalid path was changed to valid one")

	log.Println("Starting NATS server ...")
	err = services.GetTransitService().StartNats()
	if err != nil {
		t.Error(err)
		return
	}

	log.Println("NATS Server was started successfully")
	time.Sleep(TestMessagesCount * 2 * time.Second)

	if services.GetTransitService().Stats().MessagesSent == 0 {
		t.Errorf("Messages should be delivered, because Groundwork entrypoint is valid. deliveredCount = %d, want = %s",
			services.GetTransitService().Stats().MessagesSent, "'>0'")
	}
}

func configNats(t *testing.T) error {
	assert.NoError(t, os.Setenv(ConfigEnv, path.Join("..", ConfigName)))

	service := services.GetTransitService()

	service.AgentConfig.NatsFilestoreDir = TestTngAgentConfigNatsFileStoreDir
	service.AgentConfig.NatsAckWait = TestNatsAckWait

	assert.NoError(t, service.StartNats())
	assert.NoError(t, service.StartTransport())
	assert.NoError(t, service.Connect())

	return nil
}

func cleanNats(t *testing.T) {
	err := services.GetTransitService().StopNats()
	if err != nil {
		t.Error(err)
	}

	cmd := exec.Command("rm", "-rf", "test_datastore")
	_, err = cmd.Output()
	if err != nil {
		t.Error(err)
	}
}

var testMessage = `{"context": {"appType":"VEMA","agentId":"3939333393342","traceToken":"token-99e93",
	"timeStamp":"2019-10-21T21:00:00.000+0000"},"resources":[{"resource":{"properties":{},
	"name":"GW8_TNG_TEST_HOST_1","type":"HOST","status":"HOST_UP",
	"lastCheckTime":"2019-10-21T21:00:00.000+0000"}},{"resource":{"properties":{},
	"name":"GW8_TNG_TEST_SERVICE_0","type":"SERVICE","owner":"GW8_TNG_TEST_HOST_1",
	"status":"SERVICE_OK","lastCheckTime":"2019-10-21T21:00:00.000+0000"},
	"metrics":[{"tags":{},"metricName":"GW8_TNG_TEST_SERVICE","sampleType":"Warning",
	"interval":{"startTime":"2019-10-20T21:00:00.000+0000","endTime":"2019-10-22T21:00:00.000+0000"},
	"value":{"valueType":"IntegerType","integerValue":1}}]}]}`
