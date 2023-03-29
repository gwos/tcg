package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/gwos/tcg/sdk/transit"
	"github.com/gwos/tcg/services"
	"github.com/stretchr/testify/assert"
)

func TestIntegration(t *testing.T) {
	defer cleanNats(t)
	defer apiClient.RemoveHost(TestHostName)

	setupIntegration(t, 5*time.Second)

	rs := makeResource(3)
	resources := new(transit.ResourcesWithServicesRequest)
	resources.SetContext(*services.GetTransitService().MakeTracerContext())
	resources.AddResource(*rs)
	inventory := new(transit.InventoryRequest)
	inventory.SetContext(*services.GetTransitService().MakeTracerContext())
	inventory.AddResource(rs.ToInventoryResource())

	inventoryPayload, err := json.Marshal(inventory)
	assert.NoError(t, err)
	resourcesPayload, err := json.Marshal(resources)
	assert.NoError(t, err)

	t.Log("Check for host availability in the database")
	time.Sleep(1 * time.Second)
	assert.NoError(t, apiClient.CheckHostExist(TestHostName, false, "irrelevant"))

	t.Log("Send SynchronizeInventory request to GroundWork Foundation")
	assert.NoError(t, services.GetTransitService().SynchronizeInventory(context.Background(), inventoryPayload))

	time.Sleep(5 * time.Second)
	t.Log("Check for host availability in the database")
	time.Sleep(1 * time.Second)
	assert.NoError(t, apiClient.CheckHostExist(TestHostName, true, "PENDING"))

	t.Log("Send ResourcesWithMetrics request to GroundWork Foundation")
	assert.NoError(t, services.GetTransitService().SendResourceWithMetrics(context.Background(), resourcesPayload))

	time.Sleep(5 * time.Second)

	t.Log("Check for host availability in the database")
	time.Sleep(1 * time.Second)
	assert.NoError(t, apiClient.CheckHostExist(TestHostName, true, "UP"))

	t.Log("Send bad ResourcesWithMetrics payload to GroundWork Foundation")
	/* expect foundation error, processing should not stop */
	badPayload := bytes.ReplaceAll(resourcesPayload,
		[]byte(`context`), []byte(`*ontex*`))
	assert.NoError(t, services.GetTransitService().SendResourceWithMetrics(context.Background(), badPayload))
	assert.Equal(t, services.StatusRunning, services.GetTransitService().Status().Nats)
	assert.Equal(t, services.StatusRunning, services.GetTransitService().Status().Transport)
}
