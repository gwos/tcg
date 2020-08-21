package services

import (
	"github.com/gwos/tcg/config"
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"os"
	"testing"
)

func init() {
	config.GetConfig().Connector.NatsStoreType = "MEMORY"
	config.GetConfig().GWConnections = []*config.GWConnection{
		{
			Enabled:         true,
			LocalConnection: false,
			HostName:        "test",
			UserName:        "test",
			Password:        "test",
		},
	}
}

func TestAgentService_StartStopNats(t *testing.T) {
	assert.NoError(t, GetAgentService().StartNats())
	assert.NoError(t, GetAgentService().StopNats())
}

func TestAgentService_StartStopController(t *testing.T) {
	assert.NoError(t, GetAgentService().StartController())
	assert.NoError(t, GetAgentService().StopController())
}

func TestAgentService_StartStopTransport(t *testing.T) {
	assert.NoError(t, GetAgentService().StartNats())
	assert.NoError(t, GetAgentService().StartTransport())
	assert.NoError(t, GetAgentService().StopTransport())
	assert.NoError(t, GetAgentService().StartTransport())
	assert.NoError(t, GetAgentService().StopNats())
}

func TestAgentService_DemandConfig(t *testing.T) {
	tmpfile, err := ioutil.TempFile("", "config")
	assert.NoError(t, err)
	err = tmpfile.Close()
	assert.NoError(t, err)
	defer os.Remove(tmpfile.Name())
	_ = os.Setenv(string(config.ConfigEnv), tmpfile.Name())

	dto := []byte(`
{
  "agentId": "99998888-7777-6666-a3b0-b14622f7dd39",
  "appName": "test-app-XX",
  "appType": "test-XX",
  "logLevel": 2,
  "tcgUrl": "http://tcg-host:9980/",
  "dalekservicesConnection": {
    "hostName": "gw-host-xxx"
  },
  "groundworkConnections": [{
	"enabled": true,
	"localConnection": false,
    "hostName": "gw-host-xx",
    "userName": "-xx-",
    "password": "xx"
  }]
}`)

	agentService := GetAgentService()
	assert.Equal(t, "", agentService.Connector.AgentID)
	_ = agentService.config(dto)
	assert.Equal(t, "99998888-7777-6666-a3b0-b14622f7dd39", agentService.Connector.AgentID)
	assert.Equal(t, "gw-host-xxx", agentService.dsClient.HostName)
}
