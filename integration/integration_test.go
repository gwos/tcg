package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"runtime"
	"testing"
	"time"

	"github.com/gwos/tcg/config"
	"github.com/gwos/tcg/sdk/clients"
	"github.com/gwos/tcg/sdk/transit"
	"github.com/gwos/tcg/services"
	"github.com/stretchr/testify/assert"
)

func TestIntegration(t *testing.T) {
	setupIntegration(t, natsAckWait5s, dynInventoryFalse)
	apiClient.RemoveAgent(services.GetTransitService().AgentID)
	defer apiClient.RemoveAgent(services.GetTransitService().AgentID)
	defer cleanNats(t)

	group := transit.ResourceGroup{
		Description: testName,
		GroupName:   testName,
		Type:        transit.HostGroup,
	}
	rs := makeResource(0, 3)
	resources := new(transit.ResourcesWithServicesRequest)
	resources.SetContext(*services.GetTransitService().MakeTracerContext())
	resources.AddResource(rs)
	inventory := new(transit.InventoryRequest)
	inventory.SetContext(*services.GetTransitService().MakeTracerContext())
	inventory.AddResource(rs.ToInventoryResource())
	group.AddResource(rs.ToResourceRef())
	inventory.AddResourceGroup(group)

	inventoryPayload, err := json.Marshal(inventory)
	assert.NoError(t, err)
	resourcesPayload, err := json.Marshal(resources)
	assert.NoError(t, err)

	t.Log("Check for host availability in the database")
	assert.NoError(t, apiClient.CheckHostExist(rs.Name, false, "irrelevant"))

	t.Log("Send SynchronizeInventory request to GroundWork Foundation")
	assert.NoError(t, services.GetTransitService().SynchronizeInventory(context.Background(), inventoryPayload))

	time.Sleep(5 * time.Second)
	t.Log("Check for host availability in the database")
	assert.NoError(t, apiClient.CheckHostExist(rs.Name, true, "PENDING"))

	t.Log("Send ResourcesWithMetrics request to GroundWork Foundation")
	assert.NoError(t, services.GetTransitService().SendResourceWithMetrics(context.Background(), resourcesPayload))

	time.Sleep(5 * time.Second)
	t.Log("Check for host availability in the database")
	assert.NoError(t, apiClient.CheckHostExist(rs.Name, true, "UP"))

	t.Log("Send bad ResourcesWithMetrics payload to GroundWork Foundation")
	/* expect foundation error, processing should not stop */
	badPayload := bytes.ReplaceAll(resourcesPayload, []byte(`context`), []byte(`*ontex*`))
	assert.NoError(t, services.GetTransitService().SendResourceWithMetrics(context.Background(), badPayload))
	assert.Equal(t, services.StatusRunning, services.GetTransitService().Status().Nats.Value())
	assert.Equal(t, services.StatusRunning, services.GetTransitService().Status().Transport.Value())

	cfg := config.GetConfig()
	gwClient := clients.GWClient{
		AppName:      cfg.Connector.AppName,
		AppType:      cfg.Connector.AppType,
		GWConnection: cfg.GWConnections[0].AsClient(),
	}

	t.Log("Test GWClient.GetServicesByAgent")
	gwServices := new(clients.GWServices)
	assert.NoError(t, gwClient.GetServicesByAgent(cfg.Connector.AgentID, gwServices))
	assert.Equal(t, 3, len(gwServices.Services))
	assert.Equal(t, "host0.test.tcg.gw8", gwServices.Services[0].HostName)

	t.Log("Test GWClient.GetHostGroupsByAppTypeAndHostNames")
	gwHostGroups := new(clients.GWHostGroups)
	assert.NoError(t, gwClient.GetHostGroupsByAppTypeAndHostNames(cfg.Connector.AppType, []string{"host0.test.tcg.gw8"}, gwHostGroups))
	assert.Equal(t, 1, len(gwHostGroups.HostGroups))
	assert.Equal(t, 1, len(gwHostGroups.HostGroups[0].Hosts))
	assert.Equal(t, "host0.test.tcg.gw8", gwHostGroups.HostGroups[0].Hosts[0].HostName)
	assert.Equal(t, testName, gwHostGroups.HostGroups[0].Name)
}

func BenchmarkE2E(b *testing.B) {
	setupIntegration(b, natsAckWait30s, dynInventoryFalse)
	apiClient.RemoveAgent(services.GetTransitService().AgentID)

	defer printMemStats()
	defer cleanNats(b)
	defer apiClient.RemoveAgent(services.GetTransitService().AgentID)

	cfg := config.GetConfig()
	gwClient := clients.GWClient{
		AppName:      cfg.Connector.AppName,
		AppType:      cfg.Connector.AppType,
		GWConnection: cfg.GWConnections[0].AsClient(),
	}
	transitService := services.GetTransitService()

	// Benchmark sending data to Backend with/without NATS: TestFlagClient
	b.Run("send.data", func(b *testing.B) {
		printMemStats()
		payload, err := inventoryRequest(TestResourcesCount, TestServicesCount)
		assert.NoError(b, err)

		if TestFlagClient {
			_, err := gwClient.SynchronizeInventory(context.Background(), payload)
			assert.NoError(b, err)
		} else {
			assert.NoError(b, transitService.SynchronizeInventory(context.Background(), payload))
		}
		time.Sleep(5 * time.Second)
		m0 := transitService.Stats().MessagesSent.Value()

		for _, res := range resources(TestResourcesCount, TestServicesCount) {
			request := transit.ResourcesWithServicesRequest{
				Context:   transitService.MakeTracerContext(),
				Resources: []transit.MonitoredResource{res},
			}
			payload, err := json.Marshal(request)
			assert.NoError(b, err)

			if TestFlagClient {
				_, err := gwClient.SendResourcesWithMetrics(context.Background(), payload)
				assert.NoError(b, err)
			} else {
				assert.NoError(b, transitService.SendResourceWithMetrics(context.Background(), payload))
			}
		}
		time.Sleep(5 * time.Second) // time for batcher + dispatcher
		runtime.GC()
		printMemStats()
		printTcgStats()

		if transitService.BatchMaxBytes == 0 {
			if cnt, dc := b.N*TestResourcesCount*TestResourcesCount, transitService.Stats().MessagesSent.Value()-m0; dc != int64(cnt) {
				b.Errorf("Messages should be delivered. deliveredCount = %d, want = %d  %v",
					dc, cnt, m0)
			}
		}
	})
}

func resources(countHst, countSvc int) []transit.MonitoredResource {
	rr := make([]transit.MonitoredResource, 0, countHst)
	for i := 0; i < countHst; i++ {
		rs := makeResource(i, countSvc)
		rr = append(rr, rs)
	}
	return rr
}

func inventoryRequest(countHst, countSvc int) ([]byte, error) {
	inventory := new(transit.InventoryRequest)
	inventory.SetContext(*services.GetTransitService().MakeTracerContext())
	group := transit.ResourceGroup{
		Description: testName,
		GroupName:   testName,
		Type:        transit.HostGroup,
	}
	for _, rs := range resources(countHst, countSvc) {
		inventory.AddResource(rs.ToInventoryResource())
		group.AddResource(rs.ToResourceRef())
	}
	inventory.AddResourceGroup(group)

	return json.MarshalIndent(inventory, " ", " ") // enlarge payload
}

// inspired by expvar.Handler() implementation
func memstats() any {
	stats := new(runtime.MemStats)
	runtime.ReadMemStats(stats)
	return *stats
}
func printMemStats() {
	println("\n~", time.Now().Format(time.DateTime), "MEM_STATS", fmt.Sprintf("%+v", memstats()))
}
func printTcgStats() {
	println("\n~", time.Now().Format(time.DateTime), "TCG_STATS", fmt.Sprintf("%+v", services.GetTransitService().Stats()))
}
