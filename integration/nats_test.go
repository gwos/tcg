package integration

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/gwos/tcg/sdk/clients"
	"github.com/gwos/tcg/services"
	"github.com/stretchr/testify/assert"
)

// Test for ensuring that all data is stored in NATS and later resent
// if Groundwork Foundation is unavailable
// TODO: TCG connects to Foundation as local connection
func TestNatsQueue1(t *testing.T) {
	setupIntegration(t, OV{natsAckWait, 5*time.Second}, OV{dynInventory, true})
	apiClient.RemoveAgent(services.GetTransitService().AgentID)
	defer apiClient.RemoveAgent(services.GetTransitService().AgentID)
	defer cleanNats(t)

	t.Log("Timeout all requests, messages will be stored in the queue")
	httpClientGWTimeout0 := clients.HttpClientGW.Timeout
	clients.HttpClientGW.Timeout = 1 * time.Nanosecond

	assert.NoError(t, services.GetTransitService().StopTransport())
	m0 := services.GetTransitService().Stats().MessagesSent.Value()
	assert.NoError(t, services.GetTransitService().StartTransport())

	testMessage, err := os.ReadFile("fixtures/sendResourceWithMetrics.json")
	assert.NoError(t, err)

	for i := 0; i < TestMessagesCount; i++ {
		assert.NoError(t, services.GetTransitService().SendResourceWithMetrics(context.Background(), testMessage))
	}
	time.Sleep(1 * time.Second)

	if dc := services.GetTransitService().Stats().MessagesSent.Value() - m0; dc != 0 {
		t.Errorf("Messages shouldn't be delivered, because cancelling all requests. deliveredCount = %d, want = %d",
			dc, 0)
		return
	}

	t.Log("Allow all requests")
	clients.HttpClientGW.Timeout = httpClientGWTimeout0
	assert.NoError(t, services.GetTransitService().StopTransport())
	assert.NoError(t, services.GetTransitService().StartTransport())

	time.Sleep(4 * time.Second)

	if dc := services.GetTransitService().Stats().MessagesSent.Value() - m0; dc == 0 {
		t.Errorf("Messages should be delivered. deliveredCount = %d, want = %s",
			dc, "'>0'")
	}
}

// Test for ensuring that all data is stored in NATS and later resent
// after NATS streaming server restarting
// TODO: TCG connects to Foundation as remote connection
func TestNatsQueue2(t *testing.T) {
	setupIntegration(t, OV{natsAckWait, 30*time.Second}, OV{dynInventory, true})
	apiClient.RemoveAgent(services.GetTransitService().AgentID)
	defer apiClient.RemoveAgent(services.GetTransitService().AgentID)
	defer cleanNats(t)

	t.Log("Timeout all requests, messages will be stored in the queue")
	httpClientGWTimeout0 := clients.HttpClientGW.Timeout
	clients.HttpClientGW.Timeout = 1 * time.Nanosecond

	assert.NoError(t, services.GetTransitService().StopTransport())
	m0 := services.GetTransitService().Stats().MessagesSent.Value()
	assert.NoError(t, services.GetTransitService().StartTransport())

	testMessage, err := os.ReadFile("fixtures/sendResourceWithMetrics.json")
	assert.NoError(t, err)

	for i := 0; i < TestMessagesCount; i++ {
		assert.NoError(t, services.GetTransitService().SendResourceWithMetrics(context.Background(), testMessage))
		time.Sleep(1 * time.Second)
	}

	if dc := services.GetTransitService().Stats().MessagesSent.Value() - m0; dc != 0 {
		t.Errorf("Messages shouldn't be delivered, because cancelling all requests. deliveredCount = %d, want = %d",
			dc, 0)
		return
	}

	t.Log("Stopping NATS server ...")
	assert.NoError(t, services.GetTransitService().StopNats())
	t.Log("NATS Server was stopped successfully")

	t.Log("Allow all requests")
	clients.HttpClientGW.Timeout = httpClientGWTimeout0

	t.Log("Starting NATS server ...")
	assert.NoError(t, services.GetTransitService().StartNats())
	assert.NoError(t, services.GetTransitService().StartTransport())

	t.Log("NATS Server was started successfully")
	time.Sleep(time.Duration(TestMessagesCount) * time.Second)

	if dc := services.GetTransitService().Stats().MessagesSent.Value() - m0; dc == 0 {
		t.Errorf("Messages should be delivered. deliveredCount = %d, want = %s",
			dc, "'>0'")
	}
}
