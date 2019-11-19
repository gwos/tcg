package integration

import (
	"github.com/gwos/tng/nats"
	"github.com/gwos/tng/services"
	"github.com/stretchr/testify/assert"
	"io/ioutil"
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
	TestLoggingLevel                   = "LEVEL_TEST"
	LoggingLevel                       = "LEVEL"
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

	testMessage, err := parseJson()
	assert.NoError(t, err)

	for i := 0; i < TestMessagesCount; i++ {
		err := nats.Publish(services.SubjSendResourceWithMetrics, testMessage)
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
	assert.NoError(t, err)

	log.Println("Config have invalid path to Groundwork Foundation, messages will be stored in a datastore:")
	services.GetTransitService().Config.GWConfig.Host = GWInvalidHost

	defer cleanNats(t)

	testMessage, err := parseJson()
	assert.NoError(t, err)

	for i := 0; i < TestMessagesCount; i++ {
		err := nats.Publish(services.SubjSendResourceWithMetrics, testMessage)
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
	assert.NoError(t, err)
	log.Println("NATS Server was stopped successfully")

	services.GetTransitService().Config.GWConfig.Host = GWValidHost
	log.Println("Invalid path was changed to valid one")

	log.Println("Starting NATS server ...")
	err = services.GetTransitService().StartNats()
	assert.NoError(t, err)

	log.Println("NATS Server was started successfully")
	time.Sleep(TestMessagesCount * 2 * time.Second)

	if services.GetTransitService().Stats().MessagesSent == 0 {
		t.Errorf("Messages should be delivered, because Groundwork entrypoint is valid. deliveredCount = %d, want = %s",
			services.GetTransitService().Stats().MessagesSent, "'>0'")
	}
}

func configNats(t *testing.T) error {
	assert.NoError(t, os.Setenv(ConfigEnv, path.Join("..", ConfigName)))
	assert.NoError(t, os.Setenv(LoggingLevel, TestLoggingLevel))

	service := services.GetTransitService()

	service.AgentConfig.NatsFilestoreDir = TestTngAgentConfigNatsFileStoreDir
	service.AgentConfig.NatsAckWait = TestNatsAckWait

	assert.NoError(t, service.StartNats())
	assert.NoError(t, service.StartTransport())
	assert.NoError(t, service.Connect())

	return nil
}

func cleanNats(t *testing.T) {
	services.GetTransitService().Stats().MessagesSent = 0

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

func parseJson() ([]byte, error) {
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
