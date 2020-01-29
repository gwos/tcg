package integration

import (
	"encoding/json"
	"fmt"
	"github.com/gwos/tng/clients"
	"github.com/gwos/tng/log"
	"github.com/gwos/tng/nats"
	"github.com/gwos/tng/services"
	. "github.com/gwos/tng/config"
	"github.com/gwos/tng/milliseconds"
	"github.com/gwos/tng/transit"
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"path"
	"reflect"
	"testing"
	"time"
)

const (
	TestMessagesCount                  = 3
	PerformanceServicesCount           = 1
	PerformanceResourcesCount          = 1000
	TestAppType                        = "VEMA"
	TestAgentID                        = "3939333393342"
	TestTraceToken                     = "token-99e93"
	GWAccount                          = "RESTAPIACCESS"
	GWPassword                         = "63c5BtYDNAPANvNqAkh9quYszwVrvLaruxmzvM4P1FSw"
	GWValidHost                        = "localhost:80"
	GWInvalidHost                      = "localhost:23"
	TestTngAgentConfigNatsFileStoreDir = "test_datastore"
)

// Test for ensuring that all data is stored in NATS and later resent
// if Groundwork Foundation is unavailable
func TestNatsQueue_1(t *testing.T) {
	assert.NoError(t, configNats(t, 5))
	defer cleanNats(t)
	log.Info("Config has invalid path to Groundwork Foundation, messages will be stored in a datastore:")
	GetConfig().GWConnections[0].HostName = GWInvalidHost

	testMessage, err := parseJSON("fixtures/sendResourceWithMetrics.json")
	assert.NoError(t, err)

	for i := 0; i < TestMessagesCount; i++ {
		assert.NoError(t, nats.Publish(services.SubjSendResourceWithMetrics, testMessage))
		time.Sleep(1 * time.Second)
	}

	if services.GetTransitService().Stats().MessagesSent != 0 {
		t.Errorf("Messages shouldn't be delivered, because Groundwork entrypoint is invalid. deliveredCount = %d, want = %d",
			services.GetTransitService().Stats().MessagesSent, 0)
		return
	}

	GetConfig().GWConnections[0].HostName = GWValidHost
	log.Info("Invalid path was changed to valid one")

	time.Sleep(TestMessagesCount * 2 * time.Second)

	if services.GetTransitService().Stats().MessagesSent == 0 {
		t.Errorf("Messages should be delivered, because Groundwork entrypoint is valid. deliveredCount = %d, want = %s",
			services.GetTransitService().Stats().MessagesSent, "'>0'")
	}
}

// Test for ensuring that all data is stored in NATS and later resent
// after NATS streaming server restarting
func TestNatsQueue_2(t *testing.T) {
	assert.NoError(t, configNats(t, 5))

	log.Info("Config has invalid path to Groundwork Foundation, messages will be stored in a datastore:")
	GetConfig().GWConnections[0].HostName = GWInvalidHost

	defer cleanNats(t)

	testMessage, err := parseJSON("fixtures/sendResourceWithMetrics.json")
	assert.NoError(t, err)

	for i := 0; i < TestMessagesCount; i++ {
		assert.NoError(t, nats.Publish(services.SubjSendResourceWithMetrics, testMessage))
		time.Sleep(1 * time.Second)
	}

	if services.GetTransitService().Stats().MessagesSent != 0 {
		t.Errorf("Messages shouldn't be delivered, because Groundwork entrypoint is invalid. deliveredCount = %d, want = %d",
			services.GetTransitService().Stats().MessagesSent, 0)
		return
	}

	log.Info("Stopping NATS server ...")
	assert.NoError(t, services.GetTransitService().StopNats())
	log.Info("NATS Server was stopped successfully")

	GetConfig().GWConnections[0].HostName = GWValidHost
	log.Info("Invalid path was changed to valid one")

	log.Info("Starting NATS server ...")
	assert.NoError(t, services.GetTransitService().StartNats())
	assert.NoError(t, services.GetTransitService().StartTransport())

	log.Info("NATS Server was started successfully")
	time.Sleep(TestMessagesCount * 1 * time.Second)

	if services.GetTransitService().Stats().MessagesSent == 0 {
		t.Errorf("Messages should be delivered, because Groundwork entrypoint is valid. deliveredCount = %d, want = %s",
			services.GetTransitService().Stats().MessagesSent, "'>0'")
	}
}

//Test NATS performance
func TestNatsPerformance(t *testing.T) {
	assert.NoError(t, configNats(t, 30))
	defer cleanNats(t)

	var resources []transit.MonitoredResource

	inventoryRes := inventoryResource()

	for i := 0; i < PerformanceServicesCount; i++ {
		inventoryRes.Services = append(inventoryRes.Services, inventoryService(i))
	}

	request := transit.InventoryRequest{
		Context:   context(),
		Resources: []transit.InventoryResource{inventoryRes},
	}
	jsonBytes, err := json.Marshal(request)
	assert.NoError(t, err)
	assert.NoError(t, services.GetTransitService().SynchronizeInventory(jsonBytes))

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
			Context:   context(),
			Resources: []transit.MonitoredResource{res},
		}
		jsonBytes, err := json.Marshal(request)
		assert.NoError(t, err)
		assert.NoError(t, services.GetTransitService().SendResourceWithMetrics(jsonBytes))
	}

	time.Sleep(10 * time.Second)

	if services.GetTransitService().Stats().MessagesSent != PerformanceResourcesCount+1 {
		t.Errorf("Messages should be delivered. deliveredCount = %d, want = %d",
			services.GetTransitService().Stats().MessagesSent, PerformanceResourcesCount+1)
	}

	defer removeHost(t)
}

func configNats(t *testing.T, natsAckWait int64) error {
	assert.NoError(t, os.Setenv(string(ConfigEnv), path.Join("..", ConfigName)))

	GetConfig().GWConnections = []*GWConnection{
		{
			HostName: GWValidHost,
			UserName: GWAccount,
			Password: GWPassword,
		},
	}

	service := services.GetTransitService()

	service.Connector.NatsFilestoreDir = TestTngAgentConfigNatsFileStoreDir
	service.Connector.NatsAckWait = natsAckWait

	assert.NoError(t, service.StartNats())
	assert.NoError(t, service.StartTransport())

	return nil
}

func cleanNats(t *testing.T) {
	services.GetTransitService().Stats().MessagesSent = 0

	assert.NoError(t, services.GetTransitService().StopNats())

	cmd := exec.Command("rm", "-rf", "test_datastore")
	_, err := cmd.Output()
	assert.NoError(t, err)
}

func parseJSON(filePath string) ([]byte, error) {
	jsonFile, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer func() { _ = jsonFile.Close() }()

	byteValue, err := ioutil.ReadAll(jsonFile)
	if err != nil {
		return nil, err
	}

	return byteValue, nil
}

func context() transit.TracerContext {
	return transit.TracerContext{
		AppType:    TestAppType,
		AgentID:    TestAgentID,
		TraceToken: TestTraceToken,
		TimeStamp:  milliseconds.MillisecondTimestamp{Time: time.Now()},
		Version:    transit.TransitModelVersion,
	}
}

func resource() transit.MonitoredResource {
	return transit.MonitoredResource{
		Name:          TestHostName,
		Type:          transit.Host,
		Status:        transit.HostUp,
		LastCheckTime: milliseconds.MillisecondTimestamp{Time: time.Now()},
		NextCheckTime: milliseconds.MillisecondTimestamp{Time: time.Now().Add(time.Minute * 60)},
		Services:      []transit.MonitoredService{},
	}
}

func service(i int) transit.MonitoredService {
	return transit.MonitoredService{
		Name:          fmt.Sprintf("%s_%s_0", TestHostName, "SERVICE"),
		Status:        transit.ServiceOk,
		Owner:         TestHostName,
		LastCheckTime: milliseconds.MillisecondTimestamp{Time: time.Now()},
		NextCheckTime: milliseconds.MillisecondTimestamp{Time: time.Now().Add(time.Minute * 60)},
		Metrics: []transit.TimeSeries{
			{
				MetricName: "Test",
				SampleType: transit.Value,
				Interval: &transit.TimeInterval{
					EndTime:   milliseconds.MillisecondTimestamp{Time: time.Now()},
					StartTime: milliseconds.MillisecondTimestamp{Time: time.Now()},
				},
				Value: &transit.TypedValue{
					ValueType:    transit.IntegerType,
					IntegerValue: int64(i),
				},
				Unit: transit.MB,
			},
		},
	}
}

func inventoryResource() transit.InventoryResource {
	return transit.InventoryResource{
		Name:   TestHostName,
		Device: "",
		Type:   "HOST",
	}
}

func inventoryService(i int) transit.InventoryService {
	return transit.InventoryService{
		Name:  fmt.Sprintf("%s_%s_%d", TestHostName, "SERVICE", i),
		Type:  "network-device",
		Owner: TestHostName,
	}
}

func removeHost(t *testing.T) {
	gwClient := &clients.GWClient{
		AppName:      GetConfig().Connector.AppName,
		GWConnection: GetConfig().GWConnections[0],
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
