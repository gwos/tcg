package services

import (
	"github.com/gwos/tng/setup"
	"github.com/stretchr/testify/assert"
	"testing"
)

func init() {
	setup.GetConfig().AgentConfig.NatsStoreType = "MEMORY"
	setup.GetConfig().GWConfigs = []*setup.GWConfig{
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
