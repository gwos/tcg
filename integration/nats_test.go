package integration

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/gwos/tcg/config"
	"github.com/gwos/tcg/sdk/clients"
	"github.com/gwos/tcg/sdk/transit"
	"github.com/gwos/tcg/services"
	"github.com/stretchr/testify/assert"
)

const (
	TestMessagesCount         = 3
	PerformanceServicesCount  = 1
	PerformanceResourcesCount = 1000
	TestAgentID               = "INTEGRATION-TEST"
	TestAppName               = "INTEGRATION-TEST"
	TestAppType               = "VEMA"
	GWAccountEnvVar           = "TEST_GW_USERNAME"
	GWPasswordEnvVar          = "TEST_GW_PASSWORD"
	GWValidHost               = "http://localhost:80"
	GWInvalidHost             = "http://localhost:23"
	TestConfigNatsStoreDir    = "natsstore.test"
)

// Test for ensuring that all data is stored in NATS and later resent
// if Groundwork Foundation is unavailable
// TCG connects to Foundation as local connection
func TestNatsQueue_1(t *testing.T) {
	defer cleanNats(t)
	setupIntegration(t, 5*time.Second)
	t.Log("Config has invalid path to Groundwork Foundation, messages will be stored in the queue:")
	config.GetConfig().GWConnections[0].HostName = GWInvalidHost
	assert.NoError(t, services.GetTransitService().StopTransport())
	m0 := services.GetTransitService().Stats().MessagesSent
	assert.NoError(t, services.GetTransitService().StartTransport())

	testMessage, err := parseJSON("fixtures/sendResourceWithMetrics.json")
	assert.NoError(t, err)

	for i := 0; i < TestMessagesCount; i++ {
		assert.NoError(t, services.GetTransitService().SendResourceWithMetrics(context.Background(), testMessage))
	}
	time.Sleep(1 * time.Second)

	if services.GetTransitService().Stats().MessagesSent-m0 != 0 {
		t.Errorf("Messages shouldn't be delivered, because Groundwork entrypoint is invalid. deliveredCount = %d, want = %d",
			services.GetTransitService().Stats().MessagesSent-m0, 0)
		return
	}

	config.GetConfig().GWConnections[0].HostName = GWValidHost
	t.Log("Invalid path was changed to valid one")
	assert.NoError(t, services.GetTransitService().StopTransport())
	assert.NoError(t, services.GetTransitService().StartTransport())

	time.Sleep(1 * time.Second)

	if services.GetTransitService().Stats().MessagesSent-m0 == 0 {
		t.Errorf("Messages should be delivered, because Groundwork entrypoint is valid. deliveredCount = %d, want = %s",
			services.GetTransitService().Stats().MessagesSent-m0, "'>0'")
	}
}

// Test for ensuring that all data is stored in NATS and later resent
// after NATS streaming server restarting
// TCG connects to Foundation as remote connection
func TestNatsQueue_2(t *testing.T) {
	defer cleanNats(t)
	setupIntegration(t, 30*time.Second)
	t.Log("Config has invalid path to Groundwork Foundation, messages will be stored in the queue:")
	config.GetConfig().GWConnections[0].HostName = GWInvalidHost
	assert.NoError(t, services.GetTransitService().StopTransport())
	m0 := services.GetTransitService().Stats().MessagesSent
	assert.NoError(t, services.GetTransitService().StartTransport())

	testMessage, err := parseJSON("fixtures/sendResourceWithMetrics.json")
	assert.NoError(t, err)

	for i := 0; i < TestMessagesCount; i++ {
		assert.NoError(t, services.GetTransitService().SendResourceWithMetrics(context.Background(), testMessage))
		time.Sleep(1 * time.Second)
	}

	if services.GetTransitService().Stats().MessagesSent-m0 != 0 {
		t.Errorf("Messages shouldn't be delivered, because Groundwork entrypoint is invalid. deliveredCount = %d, want = %d",
			services.GetTransitService().Stats().MessagesSent-m0, 0)
		return
	}

	t.Log("Stopping NATS server ...")
	assert.NoError(t, services.GetTransitService().StopNats())
	t.Log("NATS Server was stopped successfully")

	config.GetConfig().GWConnections[0].HostName = GWValidHost
	t.Log("Invalid path was changed to valid one")

	t.Log("Starting NATS server ...")
	assert.NoError(t, services.GetTransitService().StartNats())
	assert.NoError(t, services.GetTransitService().StartTransport())

	t.Log("NATS Server was started successfully")
	time.Sleep(TestMessagesCount * 1 * time.Second)

	if services.GetTransitService().Stats().MessagesSent-m0 == 0 {
		t.Errorf("Messages should be delivered, because Groundwork entrypoint is valid. deliveredCount = %d, want = %s",
			services.GetTransitService().Stats().MessagesSent-m0, "'>0'")
	}
}

//Test NATS performance
func TestNatsPerformance(t *testing.T) {
	defer cleanNats(t)
	setupIntegration(t, 30*time.Second)
	m0 := services.GetTransitService().Stats().MessagesSent

	var resources []transit.MonitoredResource

	inventoryRes := inventoryResource()

	for i := 0; i < PerformanceServicesCount; i++ {
		inventoryRes.Services = append(inventoryRes.Services, inventoryService(i))
	}

	request := transit.InventoryRequest{
		Context:   services.GetTransitService().MakeTracerContext(),
		Resources: []transit.InventoryResource{inventoryRes},
	}
	jsonBytes, err := json.Marshal(request)
	assert.NoError(t, err)
	assert.NoError(t, services.GetTransitService().SynchronizeInventory(context.Background(), jsonBytes))

	for i := 0; i < PerformanceResourcesCount; i++ {
		res := resource()

		for j := 0; j < PerformanceServicesCount; j++ {
			res.Services = append(res.Services, service(i))
		}

		resources = append(resources, res)
	}

	time.Sleep(5 * time.Second)

	for _, res := range resources {
		request := transit.ResourcesWithServicesRequest{
			Context:   services.GetTransitService().MakeTracerContext(),
			Resources: []transit.MonitoredResource{res},
		}
		jsonBytes, err := json.Marshal(request)
		assert.NoError(t, err)
		assert.NoError(t, services.GetTransitService().SendResourceWithMetrics(context.Background(), jsonBytes))
	}

	time.Sleep(10 * time.Second)

	if services.GetTransitService().Stats().MessagesSent-m0 != PerformanceResourcesCount+1 {
		t.Errorf("Messages should be delivered. deliveredCount = %d, want = %d",
			services.GetTransitService().Stats().MessagesSent-m0, PerformanceResourcesCount+1)
	}

	defer removeHost(t)
}

func setupIntegration(t *testing.T, natsAckWait time.Duration) {
	testGroundworkUserName := os.Getenv(GWAccountEnvVar)
	testGroundworkPassword := os.Getenv(GWPasswordEnvVar)
	if testGroundworkUserName == "" || testGroundworkPassword == "" {
		t.Errorf("[setupIntegration]: Provide environment variables for Groundwork Connection username('%s') and password('%s')",
			GWAccountEnvVar, GWPasswordEnvVar)
		t.SkipNow()
	}

	cfg := config.GetConfig()
	cfg.Connector.AgentID = TestAgentID
	cfg.Connector.AppName = TestAppName
	cfg.Connector.AppType = TestAppType
	cfg.Connector.LogLevel = 2
	cfg.Connector.NatsAckWait = natsAckWait
	cfg.Connector.NatsStoreDir = TestConfigNatsStoreDir
	cfg.GWConnections = []*config.GWConnection{
		{
			Enabled:         true,
			LocalConnection: false,
			HostName:        GWValidHost,
			UserName:        testGroundworkUserName,
			Password:        testGroundworkPassword,

			IsDynamicInventory: cfg.Connector.IsDynamicInventory,
			HTTPEncode:         func() bool { return strings.ToLower(cfg.Connector.GWEncode) == "force" }(),
		},
	}

	service := services.GetTransitService()
	assert.NoError(t, service.StopNats())
	assert.NoError(t, service.StartNats())
	assert.NoError(t, service.StartTransport())
	t.Log("[setupIntegration]: ", service.Status())
}

func cleanNats(t *testing.T) {
	assert.NoError(t, services.GetTransitService().StopNats())
	cmd := exec.Command("rm", "-rf", TestConfigNatsStoreDir)
	_, err := cmd.Output()
	assert.NoError(t, err)
	t.Log("[cleanNats]: ", services.GetTransitService().Status())
}

func parseJSON(filePath string) ([]byte, error) {
	jsonFile, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer jsonFile.Close()

	byteValue, err := io.ReadAll(jsonFile)
	if err != nil {
		return nil, err
	}

	return byteValue, nil
}

func resource() transit.MonitoredResource {
	lastCheckTime := *transit.NewTimestamp()
	nextCheckTime := lastCheckTime.Add(time.Minute * 60)
	return transit.MonitoredResource{
		BaseResource: transit.BaseResource{
			BaseInfo: transit.BaseInfo{
				Name: TestHostName,
				Type: transit.ResourceTypeHost,
			},
		},
		MonitoredInfo: transit.MonitoredInfo{
			Status:        transit.HostUp,
			LastCheckTime: &lastCheckTime,
			NextCheckTime: &nextCheckTime,
		},
		Services: []transit.MonitoredService{},
	}
}

func service(i int) transit.MonitoredService {
	lastCheckTime := *transit.NewTimestamp()
	nextCheckTime := lastCheckTime.Add(time.Minute * 60)
	return transit.MonitoredService{
		BaseInfo: transit.BaseInfo{
			Name:  fmt.Sprintf("%s_%s_0", TestHostName, "SERVICE"),
			Type:  transit.ResourceTypeService,
			Owner: TestHostName,
		},
		MonitoredInfo: transit.MonitoredInfo{
			Status:        transit.ServiceOk,
			LastCheckTime: &lastCheckTime,
			NextCheckTime: &nextCheckTime,
		},
		Metrics: []transit.TimeSeries{
			{
				MetricName: "Test",
				SampleType: transit.Value,
				Interval: &transit.TimeInterval{
					EndTime:   &lastCheckTime,
					StartTime: &lastCheckTime,
				},
				Value: transit.NewTypedValue(i),
				Unit:  transit.MB,
			},
		},
	}
}

func inventoryResource() transit.InventoryResource {
	return transit.InventoryResource{
		BaseResource: transit.BaseResource{
			BaseInfo: transit.BaseInfo{
				Name: TestHostName,
				Type: transit.ResourceTypeHost,
			},
		},
	}
}

func inventoryService(i int) transit.InventoryService {
	return transit.InventoryService{
		BaseInfo: transit.BaseInfo{
			Name:  fmt.Sprintf("%s_%s_%d", TestHostName, "SERVICE", i),
			Type:  "network-device",
			Owner: TestHostName,
		},
	}
}

func removeHost(t *testing.T) {
	gwClient := &clients.GWClient{
		AppName:      config.GetConfig().Connector.AppName,
		GWConnection: (*clients.GWConnection)(config.GetConfig().GWConnections[0]),
	}
	err := gwClient.Connect()
	assert.NoError(t, err)

	token := reflect.ValueOf(gwClient).Elem().FieldByName("token").String()
	headers := map[string]string{
		"Accept":         "application/json",
		"GWOS-APP-NAME":  gwClient.AppName,
		"GWOS-API-TOKEN": token,
	}

	_, _, err = clients.SendRequest(http.MethodDelete, HostDeleteAPI+TestHostName, headers, nil, nil)
	assert.NoError(t, err)
}
