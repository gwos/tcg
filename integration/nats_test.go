package integration

import (
	. "github.com/gwos/tng/config"
	"github.com/gwos/tng/log"
	"github.com/gwos/tng/nats"
	"github.com/gwos/tng/services"
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
	log.Info("Config has invalid path to Groundwork Foundation, messages will be stored in a datastore:")
	GetConfig().GWConfigs[0].Host = GWInvalidHost

	testMessage, err := parseJSON()
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
	err := configNats(t)
	assert.NoError(t, err)

	log.Info("Config has invalid path to Groundwork Foundation, messages will be stored in a datastore:")
	GetConfig().GWConfigs[0].Host = GWInvalidHost

	defer cleanNats(t)

	testMessage, err := parseJSON()
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

	log.Info("Stopping NATS server ...")
	err = services.GetTransitService().StopNats()
	assert.NoError(t, err)
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

func configNats(t *testing.T) error {
	assert.NoError(t, os.Setenv(ConfigEnv, path.Join("..", ConfigName)))

	GetConfig().GWConfigs = []*GWConfig{
		{
			Host:     "localhost:80",
			Account:  "RESTAPIACCESS",
			Password: "63c5BtYDNAPANvNqAkh9quYszwVrvLaruxmzvM4P1FSw",
			AppName:  "tng",
		},
	}

	service := services.GetTransitService()

	service.AgentConfig.NatsFilestoreDir = TestTngAgentConfigNatsFileStoreDir
	service.AgentConfig.NatsAckWait = TestNatsAckWait

	assert.NoError(t, service.StartNats())
	assert.NoError(t, service.StartTransport())

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
