package integration

import (
	"encoding/json"
	"fmt"
	"github.com/gwos/tng/clients"
	"github.com/gwos/tng/log"
	"github.com/gwos/tng/services"
	. "github.com/gwos/tng/config"
	"github.com/gwos/tng/subseconds"
	"github.com/gwos/tng/transit"
	"github.com/stretchr/testify/assert"
	"net/http"
	"os"
	"os/exec"
	"path"
	"reflect"
	"testing"
	"time"
)

const (
	HostGetAPI        = "http://localhost/api/hosts/"
	HostDeleteAPI     = "http://localhost/api/hosts/"
	HostStatusPending = "PENDING"
	HostStatusUp      = "UP"
	TestHostName      = "GW8_TNG_TEST_HOST"
)

type Response struct {
	HostName      string `json:"hostName"`
	MonitorStatus string `json:"monitorStatus"`
}

var headers map[string]string

func TestIntegration(t *testing.T) {
	var err error
	assert.NoError(t, configNats(t, 5))
	headers, err = config(t)
	defer cleanNats(t)
	defer clean(headers)
	assert.NoError(t, err)

	log.Info("Check for host availability in the database")
	time.Sleep(1 * time.Second)
	assert.NoError(t, existenceCheck(false, "irrelevant"))

	log.Info("Send SynchronizeInventory request to GroundWork Foundation")
	assert.NoError(t, services.GetTransitService().SynchronizeInventory(buildInventoryRequest(t)))

	time.Sleep(5 * time.Second)
	log.Info("Check for host availability in the database")
	time.Sleep(1 * time.Second)
	assert.NoError(t, existenceCheck(true, HostStatusPending))

	log.Info("Send ResourcesWithMetrics request to GroundWork Foundation")
	assert.NoError(t, services.GetTransitService().SendResourceWithMetrics(buildResourceWithMetricsRequest(t)))

	time.Sleep(5 * time.Second)

	log.Info("Check for host availability in the database")
	time.Sleep(1 * time.Second)
	assert.NoError(t, existenceCheck(true, HostStatusUp))
}

func BenchmarkWithJavaIntegration(t *testing.B) {
	var err error
	headers, err = config(t)
	defer clean(headers)
	assert.NoError(t, err)

	assert.NoError(t, existenceCheck(false, "irrelevant"))

	assert.NoError(t, installDependencies())

	assert.NoError(t, runJavaSynchronizeInventoryTest())

	assert.NoError(t, existenceCheck(true, HostStatusPending))

	assert.NoError(t, runJavaSendResourceWithMetricsTest())

	assert.NoError(t, existenceCheck(true, HostStatusUp))
}

func buildInventoryRequest(t *testing.T) []byte {
	inventoryResource := transit.InventoryResource{
		Name: TestHostName,
		Type: "HOST",
		Services: []transit.InventoryService{
			{
				Name:  "test",
				Type:  transit.Hypervisor,
				Owner: TestHostName,
			},
		},
	}

	inventoryRequest := transit.InventoryRequest{
		Context: transit.TracerContext{
			AppType:    TestAppType,
			AgentID:    TestAgentID,
			TraceToken: TestTraceToken,
			TimeStamp:  subseconds.MillisecondTimestamp{Time: time.Now()},
		},
		Resources: []transit.InventoryResource{inventoryResource},
		Groups:    nil,
	}

	b, err := json.Marshal(inventoryRequest)
	assert.NoError(t, err)

	return b
}

func buildResourceWithMetricsRequest(t *testing.T) []byte {
	monitoredResource := transit.MonitoredResource{
		Name:          TestHostName,
		Type:          transit.Host,
		Status:        transit.HostUp,
		LastCheckTime: subseconds.MillisecondTimestamp{Time: time.Now()},
		NextCheckTime: subseconds.MillisecondTimestamp{Time: time.Now()},
		Services: []transit.MonitoredService{
			{
				Name:          "test",
				Status:        transit.ServiceOk,
				Owner:         TestHostName,
				LastCheckTime: subseconds.MillisecondTimestamp{Time: time.Now()},
				NextCheckTime: subseconds.MillisecondTimestamp{Time: time.Now()},
				Metrics: []transit.TimeSeries{
					{
						MetricName: "testMetric",
						SampleType: transit.Value,
						Interval: &transit.TimeInterval{
							EndTime:   subseconds.MillisecondTimestamp{Time: time.Now()},
							StartTime: subseconds.MillisecondTimestamp{Time: time.Now()},
						},
						Value: &transit.TypedValue{
							ValueType:    transit.IntegerType,
							IntegerValue: 1000,
						},
						Unit: transit.MB,
					},
				},
			},
		},
	}

	request := transit.ResourcesWithServicesRequest{
		Context: transit.TracerContext{
			AppType:    TestAppType,
			AgentID:    TestAgentID,
			TraceToken: TestTraceToken,
			TimeStamp:  subseconds.MillisecondTimestamp{Time: time.Now()},
		},
		Resources: []transit.MonitoredResource{monitoredResource},
	}

	b, err := json.Marshal(request)
	assert.NoError(t, err)
	return b
}

func config(t assert.TestingT) (map[string]string, error) {
	err := os.Setenv(string(ConfigEnv), path.Join("..", ConfigName))
	assert.NoError(t, err)

	gwClient := &clients.GWClient{
		AppName:      GetConfig().Connector.AppName,
		GWConnection: GetConfig().GWConnections[0],
	}
	err = gwClient.Connect()
	assert.NoError(t, err)

	token := reflect.ValueOf(gwClient).Elem().FieldByName("token").String()
	headers := map[string]string{
		"Accept":         "application/json",
		"GWOS-APP-NAME":  gwClient.AppName,
		"GWOS-API-TOKEN": token,
	}

	return headers, nil
}

func existenceCheck(mustExist bool, mustHasStatus string) error {
	statusCode, byteResponse, err := clients.SendRequest(http.MethodGet, HostGetAPI+TestHostName, headers, nil, nil)
	if err != nil {
		return err
	}
	if statusCode == 200 {
		log.Info(" -> Host exists")
	} else {
		log.Info(" -> Host doesn't exist")
	}

	if !mustExist && statusCode == 404 {
		return nil
	}
	if !(mustExist && statusCode == 200) {
		return fmt.Errorf("Status code = %d (Details: %s), want = %d ", statusCode, string(byteResponse), 404)
	}

	var response Response

	err = json.Unmarshal(byteResponse, &response)
	if err != nil {
		return err
	}

	if mustExist && (response.HostName != TestHostName || response.MonitorStatus != mustHasStatus) {
		return fmt.Errorf("Host from database = (Name: %s, Status: %s), want = (Name: %s, Status: %s)",
			response.HostName, response.MonitorStatus, TestHostName, mustHasStatus)
	}

	return nil
}

func installDependencies() error {
	workDir, err := os.Getwd()
	if err != nil {
		return err
	}

	cmd := exec.Command("mvn", "install:install-file", "-Dfile=lib/collagerest-common-8.0.0-SNAPSHOT.jar", "-DgroupId=org.groundwork", "-DartifactId=collagerest-common", "-Dversion=8.0.0-SNAPSHOT")
	cmd.Dir = path.Join(workDir, "../gw-transit")
	_, err = cmd.Output()
	if err != nil {
		return err
	}

	return nil
}

func runJavaSynchronizeInventoryTest() error {
	workDir, err := os.Getwd()
	if err != nil {
		return err
	}

	cmd := exec.Command("mvn", "-Dtest=AppTest#shouldSynchronizeInventory", "test")
	cmd.Dir = path.Join(workDir, "../gw-transit")
	_, err = cmd.Output()
	if err != nil {
		return err
	}

	return nil
}

func runJavaSendResourceWithMetricsTest() error {
	workDir, err := os.Getwd()
	if err != nil {
		return err
	}

	cmd := exec.Command("mvn", "-Dtest=AppTest#shouldSendResourceAndMetrics", "test")
	cmd.Dir = path.Join(workDir, "../gw-transit")
	_, err = cmd.Output()
	if err != nil {
		return err
	}

	return nil
}

func clean(headers map[string]string) {
	_, _, err := clients.SendRequest(http.MethodDelete, HostDeleteAPI+TestHostName, headers, nil, nil)
	if err != nil {
		log.Error(err.Error())
	}

	cmd := exec.Command("rm", "-rf", "../gw-transit/src/main/resources/datastore")
	_, err = cmd.Output()
	if err != nil {
		log.Error(err.Error())
	}
}
