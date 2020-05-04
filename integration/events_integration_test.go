package integration

import (
	"github.com/gwos/tcg/nats"
	"github.com/gwos/tcg/services"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func TestEvents(t *testing.T) {
	testMessage, err := parseJSON("fixtures/sendEvents.json")
	assert.NoError(t, err)

	configNats(t, 5)
	defer cleanNats(t)

	assert.NoError(t, nats.Publish(services.SubjSendEvents, testMessage))

	time.Sleep(1 * time.Second)

	if services.GetTransitService().Stats().MessagesSent == 0 {
		t.Errorf("Message should be delivered. deliveredCount = %d, want = %d",
			services.GetTransitService().Stats().MessagesSent, 1)
		return
	}

	if len(services.GetTransitService().Stats().LastErrors) != 0 {
		t.Errorf("LastErrors should be empty. lastError = %s",
			services.GetTransitService().Stats().LastErrors)
		return
	}
}
