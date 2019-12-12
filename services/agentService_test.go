package services

import (
	"github.com/gwos/tng/config"
	"github.com/stretchr/testify/assert"
	"testing"
)

func init() {
	config.GetConfig().AgentConfig.NatsStoreType = "MEMORY"
	config.GetConfig().GWConfigs = []*config.GWConfig{
		{
			Host:     "test",
			Account:  "test",
			Password: "test",
			AppName:  "test",
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
