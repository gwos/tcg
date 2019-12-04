package services

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestAgentService_StopNats(t *testing.T) {
	assert.NoError(t, GetTransitService().StopNats())
}

func TestAgentService_StopController(t *testing.T) {
	assert.NoError(t, GetTransitService().StopController())
}

func TestAgentService_StopTransport(t *testing.T) {
	assert.NoError(t, GetTransitService().StopTransport())
}
