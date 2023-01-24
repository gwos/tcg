package integration

import (
	"context"
	"testing"
	"time"

	"github.com/gwos/tcg/services"
	"github.com/stretchr/testify/assert"
)

func TestEvents(t *testing.T) {
	testMessage, err := parseJSON("fixtures/sendEvents.json")
	assert.NoError(t, err)

	setupIntegration(t, 5*time.Second)
	defer cleanNats(t)

	l0 := len(services.GetTransitService().Stats().LastErrors)
	m0 := services.GetTransitService().Stats().MessagesSent.Value()
	assert.NoError(t, services.GetController().SendEvents(context.Background(), testMessage))
	time.Sleep(1 * time.Second)

	if dc := services.GetTransitService().Stats().MessagesSent.Value() - m0; dc != 1 {
		t.Errorf("Message should be delivered. deliveredCount = %d, want = %d", dc, 1)
		return
	}

	if len(services.GetTransitService().Stats().LastErrors) != l0 {
		t.Errorf("LastErrors should be not changed")
		t.Log("lastErrors:", services.GetTransitService().Stats().LastErrors)
		return
	}
}
