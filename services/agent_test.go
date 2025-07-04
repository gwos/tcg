package services

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/gwos/tcg/config"
	"github.com/stretchr/testify/assert"
)

func init() {
	config.GetConfig().Connector.ControllerAddr = ":11099"
	config.GetConfig().Connector.NatsStoreType = "MEMORY"
	config.GetConfig().Connector.NatsStoreMaxBytes = 1024000
	config.GetConfig().GWConnections = []config.GWConnection{
		{
			Enabled:         true,
			LocalConnection: false,
			HostName:        "test",
			UserName:        "test",
			Password:        "test",
		},
	}
}

func TestAgentService(t *testing.T) {
	t.Cleanup(func() {
		assert.NoError(t, os.RemoveAll(filepath.Join(GetAgentService().Connector.NatsStoreDir, "jetstream")))
		assert.NoError(t, os.RemoveAll(filepath.Join(GetController().Connector.NatsStoreDir, "inventory.json")))
		assert.NoError(t, os.RemoveAll(filepath.Join(GetController().Connector.NatsStoreDir, "inventory1.json")))
		assert.NoError(t, os.Remove(GetAgentService().Connector.NatsStoreDir))
	})

	t.Run("Controller", func(t *testing.T) {
		assert.NoError(t, GetAgentService().StartController())
		assert.NoError(t, GetAgentService().StopController())
		assert.NoError(t, GetAgentService().StartController())
		assert.NoError(t, GetAgentService().StopController())
	})

	t.Run("NATS", func(t *testing.T) {
		assert.NoError(t, GetAgentService().StartNats())
		assert.NoError(t, GetAgentService().StopNats())
		assert.NoError(t, GetAgentService().StartNats())
		assert.NoError(t, GetAgentService().StopNats())
	})

	t.Run("Transport", func(t *testing.T) {
		GetAgentService().Connector.AgentID = "TESTAGENTID"
		GetAgentService().Connector.AppType = "TESTAPPTYPE"
		assert.NoError(t, GetAgentService().StartNats())
		assert.NoError(t, GetAgentService().StartTransport())
		assert.NoError(t, GetAgentService().StopTransport())
		assert.NoError(t, GetAgentService().StartTransport())
		assert.NoError(t, GetAgentService().StopNats())
	})

	t.Run("DemandConfig", func(t *testing.T) {
		tmpfile, err := os.CreateTemp("", "config")
		assert.NoError(t, err)
		err = tmpfile.Close()
		assert.NoError(t, err)
		defer os.Remove(tmpfile.Name())
		t.Setenv(config.ConfigEnv, tmpfile.Name())
		t.Setenv("TCG_CONNECTOR_NATSSTOREMAXBYTES", "333_222_111_000")

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
		assert.Equal(t, "TESTAGENTID", agentService.Connector.AgentID)
		assert.NoError(t, agentService.config(dto))
		assert.Equal(t, "99998888-7777-6666-a3b0-b14622f7dd39", agentService.Connector.AgentID)
		assert.Equal(t, "gw-host-xxx", agentService.dsClient.HostName)
		assert.NoError(t, agentService.startNats())
		assert.NoError(t, agentService.startTransport())
		assert.Equal(t, "gw-host-xx", agentService.gwClients[0].HostName)
	})
}
