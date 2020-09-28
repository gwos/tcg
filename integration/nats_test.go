package integration

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"path"
	"reflect"
	"testing"
	"time"

	"github.com/gwos/tcg/cache"
	"github.com/gwos/tcg/clients"
	. "github.com/gwos/tcg/config"
	"github.com/gwos/tcg/log"
	"github.com/gwos/tcg/milliseconds"
	"github.com/gwos/tcg/services"
	"github.com/gwos/tcg/transit"
	"github.com/stretchr/testify/assert"
)

const (
	TestMessagesCount                  = 3
	PerformanceServicesCount           = 1
	PerformanceResourcesCount          = 1000
	TestAppType                        = "VEMA"
	TestAgentID                        = "3939333393342"
	TestTraceToken                     = "token-99e93"
	GWAccountEnvVar                    = "TEST_GW_USERNAME"
	GWPasswordEnvVar                   = "TEST_GW_PASSWORD"
	GWAccountRemote                    = "admin"
	GWPasswordRemote                   = "admin"
	GWValidHost                        = "http://localhost:80"
	GWInvalidHost                      = "http://localhost:23"
	TestTcgAgentConfigNatsFileStoreDir = "test_datastore"
)

// Test for ensuring that all data is stored in NATS and later resent
// if Groundwork Foundation is unavailable
// TCG connects to Foundation as local connection
func TestNatsQueue_1(t *testing.T) {
	defer cleanNats(t)
	configNats(t, 5)
	log.Info("Config has invalid path to Groundwork Foundation, messages will be stored in a datastore:")
	GetConfig().GWConnections[0].HostName = GWInvalidHost
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

	GetConfig().GWConnections[0].HostName = GWValidHost
	log.Info("Invalid path was changed to valid one")
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
	configNats(t, 30)
	log.Info("Config has invalid path to Groundwork Foundation, messages will be stored in a datastore:")
	GetConfig().GWConnections[0].HostName = GWInvalidHost
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

	log.Info("Stopping NATS server ...")
	assert.NoError(t, services.GetTransitService().StopNats())
	log.Info("NATS Server was stopped successfully")

	GetConfig().GWConnections[0].LocalConnection = false
	GetConfig().GWConnections[0].UserName = GWAccountRemote
	GetConfig().GWConnections[0].Password = GWPasswordRemote
	GetConfig().GWConnections[0].HostName = GWValidHost
	log.Info("Invalid path was changed to valid one")

	log.Info("Starting NATS server ...")
	assert.NoError(t, services.GetTransitService().StartNats())
	assert.NoError(t, services.GetTransitService().StartTransport())

	log.Info("NATS Server was started successfully")
	time.Sleep(TestMessagesCount * 1 * time.Second)

	if services.GetTransitService().Stats().MessagesSent-m0 == 0 {
		t.Errorf("Messages should be delivered, because Groundwork entrypoint is valid. deliveredCount = %d, want = %s",
			services.GetTransitService().Stats().MessagesSent-m0, "'>0'")
	}
}

//Test NATS performance
func TestNatsPerformance(t *testing.T) {
	defer cleanNats(t)
	configNats(t, 30)
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

func configNats(t *testing.T, natsAckWait int64) {
	time.Sleep(3 * time.Second)

	assert.NoError(t, os.Setenv(string(ConfigEnv), path.Join("..", ConfigName)))
	testGroundworkUserName := os.Getenv(GWAccountEnvVar)
	testGroundworkPassword := os.Getenv(GWPasswordEnvVar)

	if testGroundworkUserName == "" || testGroundworkPassword == "" {
		t.Errorf("|nats_test.go| [configNats]: Provide environment variables for Groundwork Connection username('%s') and password('%s')",
			GWAccountEnvVar, GWPasswordEnvVar)
		t.SkipNow()
	}

	GetConfig().GWConnections = []*GWConnection{
		{
			Enabled:         true,
			LocalConnection: true,
			HostName:        GWValidHost,
			UserName:        testGroundworkUserName,
			Password:        testGroundworkPassword,
		},
	}

	service := services.GetTransitService()
	service.Connector.NatsFilestoreDir = TestTcgAgentConfigNatsFileStoreDir
	service.Connector.NatsAckWait = natsAckWait

	assert.NoError(t, service.StopNats())
	assert.NoError(t, service.StartNats())
	assert.NoError(t, service.StartTransport())
	log.Info("#configNats status: ", service.Status())
}

func cleanNats(t *testing.T) {
	assert.NoError(t, services.GetTransitService().StopNats())
	cmd := exec.Command("rm", "-rf", TestTcgAgentConfigNatsFileStoreDir)
	_, err := cmd.Output()
	assert.NoError(t, err)
	cache.DispatcherDoneCache.Flush()
	log.Info("#cleanNats status: ", services.GetTransitService().Status())
}

func parseJSON(filePath string) ([]byte, error) {
	jsonFile, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer jsonFile.Close()

	byteValue, err := ioutil.ReadAll(jsonFile)
	if err != nil {
		return nil, err
	}

	return byteValue, nil
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
