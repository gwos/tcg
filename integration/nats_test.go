package integration

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"testing"
	"time"

	"github.com/gwos/tcg/config"
	"github.com/gwos/tcg/sdk/clients"
	"github.com/gwos/tcg/sdk/transit"
	"github.com/gwos/tcg/services"
	"github.com/stretchr/testify/assert"
)

const (
	TestHostName = "GW8_TCG_TEST_HOST"

	TestMessagesCount         = 4
	PerformanceServicesCount  = 20
	PerformanceResourcesCount = 50
	PerformanceLoopMetrics    = 4

	dynInventoryFalse = false
	dynInventoryTrue  = true
	natsAckWait5s     = time.Second * 5
	natsAckWait30s    = time.Second * 30
)

var TestConfigDefaults = map[string]string{
	"TCG_CONNECTOR_AGENTID":          "INTEGRATION-TEST",
	"TCG_CONNECTOR_APPNAME":          "INTEGRATION-TEST",
	"TCG_CONNECTOR_APPTYPE":          "VEMA",
	"TCG_CONNECTOR_ENABLED":          "true",
	"TCG_CONNECTOR_NATSFILESTOREDIR": "natsstore.test",
	"TCG_GWCONNECTIONS_0_ENABLED":    "true",
	"TCG_GWCONNECTIONS_0_HOSTNAME":   "https://localhost",
	"TCG_GWCONNECTIONS_0_PASSWORD":   "",
	"TCG_GWCONNECTIONS_0_USERNAME":   "",
}

var apiClient = new(APIClient)

// Test for ensuring that all data is stored in NATS and later resent
// if Groundwork Foundation is unavailable
// TODO: TCG connects to Foundation as local connection
func TestNatsQueue1(t *testing.T) {
	defer cleanNats(t)
	setupIntegration(t, natsAckWait5s, dynInventoryTrue)

	t.Log("Timeout all requests, messages will be stored in the queue")
	httpClientTimeout0 := clients.HttpClient.Timeout
	clients.HttpClient.Timeout = 1 * time.Nanosecond

	assert.NoError(t, services.GetTransitService().StopTransport())
	m0 := services.GetTransitService().Stats().MessagesSent.Value()
	assert.NoError(t, services.GetTransitService().StartTransport())

	testMessage, err := readFile("fixtures/sendResourceWithMetrics.json")
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

	clients.HttpClient.Timeout = httpClientTimeout0
	t.Log("Allow all requests")
	assert.NoError(t, services.GetTransitService().StopTransport())
	assert.NoError(t, services.GetTransitService().StartTransport())

	time.Sleep(40 * time.Second)

	if dc := services.GetTransitService().Stats().MessagesSent.Value() - m0; dc != TestMessagesCount {
		t.Errorf("Messages should be delivered. deliveredCount = %v, want = %v",
			dc, TestMessagesCount)
	}
}

// Test for ensuring that all data is stored in NATS and later resent
// after NATS streaming server restarting
// TODO: TCG connects to Foundation as remote connection
func TestNatsQueue2(t *testing.T) {
	defer cleanNats(t)
	setupIntegration(t, natsAckWait30s, dynInventoryTrue)

	t.Log("Timeout all requests, messages will be stored in the queue")
	httpClientTimeout0 := clients.HttpClient.Timeout
	clients.HttpClient.Timeout = 1 * time.Nanosecond

	assert.NoError(t, services.GetTransitService().StopTransport())
	m0 := services.GetTransitService().Stats().MessagesSent.Value()
	assert.NoError(t, services.GetTransitService().StartTransport())

	testMessage, err := readFile("fixtures/sendResourceWithMetrics.json")
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

	clients.HttpClient.Timeout = httpClientTimeout0
	t.Log("Allow all requests")

	t.Log("Starting NATS server ...")
	assert.NoError(t, services.GetTransitService().StartNats())
	assert.NoError(t, services.GetTransitService().StartTransport())

	t.Log("NATS Server was started successfully")
	time.Sleep(TestMessagesCount * 1 * time.Second)

	if dc := services.GetTransitService().Stats().MessagesSent.Value() - m0; dc == 0 {
		t.Errorf("Messages should be delivered. deliveredCount = %d, want = %s",
			dc, "'>0'")
	}
}

// Test NATS performance
func TestNatsPerformance(t *testing.T) {
	defer cleanNats(t)
	defer apiClient.RemoveHost(TestHostName)

	setupIntegration(t, natsAckWait30s, dynInventoryFalse)
	m0 := services.GetTransitService().Stats().MessagesSent.Value()

	resources := make([]transit.MonitoredResource, 0, PerformanceResourcesCount)
	inventory := new(transit.InventoryRequest)
	inventory.SetContext(*services.GetTransitService().MakeTracerContext())

	for i := 0; i < PerformanceResourcesCount; i++ {
		rs := makeResource(i, PerformanceServicesCount)
		rs.SetProperty("__seq__", i)

		resources = append(resources, *rs)
		inventory.AddResource(rs.ToInventoryResource())
	}

	payload, err := json.Marshal(inventory)
	assert.NoError(t, err)
	assert.NoError(t, services.GetTransitService().SynchronizeInventory(context.Background(), payload))

	time.Sleep(5 * time.Second)

	t1 := time.Now()
	go func(t *testing.T) {
		for i := 0; i < PerformanceLoopMetrics; i++ {
			for _, res := range resources {
				request := transit.ResourcesWithServicesRequest{
					Context:   services.GetTransitService().MakeTracerContext(),
					Resources: []transit.MonitoredResource{res},
				}
				payload, err := json.Marshal(request)
				assert.NoError(t, err)
				assert.NoError(t, services.GetTransitService().SendResourceWithMetrics(context.Background(), payload))
			}
			time.Sleep(5 * time.Second)
		}
		t.Logf("--nats published %v %v", time.Since(t1).Round(time.Millisecond).String(), len(resources))
	}(t)

	time.Sleep(20 * time.Millisecond)
	_ = services.GetTransitService().PauseNats()
	_ = services.GetTransitService().StopNats()
	_ = services.GetTransitService().StartNats()
	_ = services.GetTransitService().UnpauseNats()
	t.Logf("--nats paused/re-started/unpaused %v", time.Since(t1).Round(time.Millisecond).String())

	_ = services.GetTransitService().StartTransport()
	time.Sleep(40 * time.Second)

	if cnt, dc := PerformanceLoopMetrics*len(resources), services.GetTransitService().Stats().MessagesSent.Value()-m0; dc != int64(cnt) {
		t.Errorf("Messages should be delivered. deliveredCount = %d, want = %d",
			dc, cnt)
	}
}

func setupIntegration(t *testing.T, natsAckWait time.Duration, isDynamicInventory bool) {
	for k, v := range TestConfigDefaults {
		if _, ok := os.LookupEnv(k); !ok {
			t.Setenv(k, v)
		}
	}
	if len(os.Getenv("TCG_GWCONNECTIONS_0_USERNAME")) == 0 ||
		len(os.Getenv("TCG_GWCONNECTIONS_0_PASSWORD")) == 0 {
		t.Errorf("[setupIntegration]: Provide environment variables for Groundwork Connection: %s and %s",
			"TCG_GWCONNECTIONS_0_USERNAME", "TCG_GWCONNECTIONS_0_PASSWORD")
		t.SkipNow()
	}

	cfg := config.GetConfig()
	cfg.Connector.NatsAckWait = natsAckWait
	cfg.GWConnections[0].IsDynamicInventory = isDynamicInventory

	service := services.GetTransitService()
	assert.NoError(t, service.StopNats())
	assert.NoError(t, service.StartNats())
	assert.NoError(t, service.StartTransport())
	t.Log("[setupIntegration]: ", service.Status())
	t.Logf("cfg.Connector: %+v", cfg.Connector)
	t.Logf("cfg.GWConnections[0]: %+v", cfg.GWConnections[0])
}

func cleanNats(t *testing.T) {
	assert.NoError(t, services.GetTransitService().StopNats())
	assert.NoError(t, services.GetTransitService().ResetNats())
	assert.NoError(t, os.Remove(config.GetConfig().Connector.NatsStoreDir))
	t.Log("[cleanNats]: ", services.GetTransitService().Status())
}

func readFile(filePath string) ([]byte, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	bb, err := io.ReadAll(f)
	if err != nil {
		return nil, err
	}

	return bb, nil
}

func makeResource(rsIdx, svcCount int) *transit.MonitoredResource {
	rs := new(transit.MonitoredResource)
	rs.Status = transit.HostUp
	rs.Type = transit.ResourceTypeHost
	rs.LastCheckTime = transit.NewTimestamp()
	rs.NextCheckTime = transit.NewTimestamp()
	*rs.NextCheckTime = rs.NextCheckTime.Add(time.Minute * 60)
	rs.Name = TestHostName
	if rsIdx > 0 {
		rs.Name = fmt.Sprintf("%v_%v", TestHostName, rsIdx)
	}

	for i := 0; i < svcCount; i++ {
		svc := new(transit.MonitoredService)
		svc.Name = fmt.Sprintf("%v_SERVICE_%v", rs.Name, i)
		svc.Owner = rs.Name
		svc.Status = transit.ServiceOk
		svc.Type = transit.ResourceTypeService
		svc.LastCheckTime = transit.NewTimestamp()
		svc.NextCheckTime = transit.NewTimestamp()
		*svc.NextCheckTime = svc.NextCheckTime.Add(time.Minute * 60)

		m := new(transit.TimeSeries)
		m.Interval = new(transit.TimeInterval)
		m.Interval.StartTime = transit.NewTimestamp()
		m.Interval.EndTime = transit.NewTimestamp()
		m.MetricName = "test_metric"
		m.SampleType = transit.Value
		m.Value = transit.NewTypedValue(i)
		m.Unit = transit.MB

		svc.Metrics = append(svc.Metrics, *m)
		rs.Services = append(rs.Services, *svc)
	}

	return rs
}
