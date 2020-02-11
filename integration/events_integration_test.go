package integration

import (
	"github.com/gwos/tng/nats"
	"github.com/gwos/tng/services"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func TestEvents(t *testing.T) {
	testMessage, err := parseJSON("fixtures/sendEvents.json")
	assert.NoError(t, err)

	configNats(t, 5)
	defer cleanNats(t)

	assert.NoError(t, nats.Publish(services.SubjSendEvent, testMessage))

	time.Sleep(1 * time.Second)

	if services.GetTransitService().Stats().MessagesSent == 0 {
		t.Errorf("Message should be delivered. deliveredCount = %d, want = %d",
			services.GetTransitService().Stats().MessagesSent, 1)
		return
	}

	if services.GetTransitService().Stats().LastError != "" {
		t.Errorf("Erorr should be empty. lastError = %s, want = %s",
			services.GetTransitService().Stats().LastError, "\"empty string\"")
		return
	}
}
