package integration

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
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
	PerformanceServicesCount  = 2
	PerformanceResourcesCount = 1000
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
	setupIntegration(t, 5*time.Second)

	t.Log("Timeout all requests, messages will be stored in the queue")
	defaultNetClientTimeout := *clients.NetClientTimeout
	*clients.NetClientTimeout = 1 * time.Nanosecond

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

	*clients.NetClientTimeout = defaultNetClientTimeout
	t.Log("Allow all requests")
	assert.NoError(t, services.GetTransitService().StopTransport())
	assert.NoError(t, services.GetTransitService().StartTransport())

	time.Sleep(1 * time.Second)

	if dc := services.GetTransitService().Stats().MessagesSent.Value() - m0; dc == 0 {
		t.Errorf("Messages should be delivered. deliveredCount = %d, want = %s",
			dc, "'>0'")
	}
}

// Test for ensuring that all data is stored in NATS and later resent
// after NATS streaming server restarting
// TODO: TCG connects to Foundation as remote connection
func TestNatsQueue2(t *testing.T) {
	defer cleanNats(t)
	setupIntegration(t, 30*time.Second)

	t.Log("Timeout all requests, messages will be stored in the queue")
	defaultNetClientTimeout := *clients.NetClientTimeout
	*clients.NetClientTimeout = 1 * time.Nanosecond

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

	*clients.NetClientTimeout = defaultNetClientTimeout
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

	setupIntegration(t, 30*time.Second)
	m0 := services.GetTransitService().Stats().MessagesSent.Value()

	resources := make([]transit.MonitoredResource, 0, PerformanceResourcesCount)
	inventory := new(transit.InventoryRequest)
	inventory.SetContext(*services.GetTransitService().MakeTracerContext())

	for i := 0; i < PerformanceResourcesCount; i++ {
		rs := makeResource(PerformanceServicesCount)

		resources = append(resources, *rs)
		inventory.AddResource(rs.ToInventoryResource())
	}

	payload, err := json.Marshal(inventory)
	assert.NoError(t, err)
	assert.NoError(t, services.GetTransitService().SynchronizeInventory(context.Background(), payload))

	time.Sleep(5 * time.Second)

	for _, res := range resources {
		request := transit.ResourcesWithServicesRequest{
			Context:   services.GetTransitService().MakeTracerContext(),
			Resources: []transit.MonitoredResource{res},
		}
		payload, err := json.Marshal(request)
		assert.NoError(t, err)
		assert.NoError(t, services.GetTransitService().SendResourceWithMetrics(context.Background(), payload))
	}

	time.Sleep(10 * time.Second)

	if dc := services.GetTransitService().Stats().MessagesSent.Value() - m0; dc != PerformanceResourcesCount+1 {
		t.Errorf("Messages should be delivered. deliveredCount = %d, want = %d",
			dc, PerformanceResourcesCount+1)
	}
}

func setupIntegration(t *testing.T, natsAckWait time.Duration) {
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
	assert.NoError(t, os.RemoveAll(filepath.Join(config.GetConfig().Connector.NatsStoreDir, "jetstream")))
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

func makeResource(svcCount int) *transit.MonitoredResource {
	rs := new(transit.MonitoredResource)
	rs.Name = TestHostName
	rs.Status = transit.HostUp
	rs.Type = transit.ResourceTypeHost
	rs.LastCheckTime = transit.NewTimestamp()
	rs.NextCheckTime = transit.NewTimestamp()
	*rs.NextCheckTime = rs.NextCheckTime.Add(time.Minute * 60)

	for i := 0; i < svcCount; i++ {
		svc := new(transit.MonitoredService)
		svc.Name = fmt.Sprintf("%s_SERVICE_%v", TestHostName, i)
		svc.Owner = TestHostName
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
