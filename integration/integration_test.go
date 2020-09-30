package integration

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path"
	"reflect"
	"testing"
	"time"

	"github.com/gwos/tcg/clients"
	. "github.com/gwos/tcg/config"
	"github.com/gwos/tcg/log"
	"github.com/gwos/tcg/milliseconds"
	"github.com/gwos/tcg/services"
	"github.com/gwos/tcg/transit"
	"github.com/stretchr/testify/assert"
)

const (
	HostGetAPI        = "http://localhost/api/hosts/"
	HostDeleteAPI     = "http://localhost/api/hosts/"
	HostStatusPending = "PENDING"
	HostStatusUp      = "UP"
	TestHostName      = "GW8_TCG_TEST_HOST"
)

type Response struct {
	HostName      string `json:"hostName"`
	MonitorStatus string `json:"monitorStatus"`
}

var headers map[string]string

func TestIntegration(t *testing.T) {
	var err error
	configNats(t, 5)
	headers, err = config(t)
	defer cleanNats(t)
	defer clean(headers)
	assert.NoError(t, err)

	log.Info("Check for host availability in the database")
	time.Sleep(1 * time.Second)
	assert.NoError(t, existenceCheck(false, "irrelevant"))

	log.Info("Send SynchronizeInventory request to GroundWork Foundation")
	assert.NoError(t, services.GetTransitService().SynchronizeInventory(context.Background(), buildInventoryRequest(t)))

	time.Sleep(5 * time.Second)
	log.Info("Check for host availability in the database")
	time.Sleep(1 * time.Second)
	assert.NoError(t, existenceCheck(true, HostStatusPending))

	log.Info("Send ResourcesWithMetrics request to GroundWork Foundation")
	assert.NoError(t, services.GetTransitService().SendResourceWithMetrics(context.Background(), buildResourceWithMetricsRequest(t)))

	time.Sleep(5 * time.Second)

	log.Info("Check for host availability in the database")
	time.Sleep(1 * time.Second)
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
		Context:   services.GetTransitService().MakeTracerContext(),
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
		LastCheckTime: milliseconds.MillisecondTimestamp{Time: time.Now()},
		NextCheckTime: milliseconds.MillisecondTimestamp{Time: time.Now()},
		Services: []transit.MonitoredService{
			{
				Name:          "test",
				Status:        transit.ServiceOk,
				Owner:         TestHostName,
				LastCheckTime: milliseconds.MillisecondTimestamp{Time: time.Now()},
				NextCheckTime: milliseconds.MillisecondTimestamp{Time: time.Now()},
				Metrics: []transit.TimeSeries{
					{
						MetricName: "testMetric",
						SampleType: transit.Value,
						Interval: &transit.TimeInterval{
							EndTime:   milliseconds.MillisecondTimestamp{Time: time.Now()},
							StartTime: milliseconds.MillisecondTimestamp{Time: time.Now()},
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
		Context:   services.GetTransitService().MakeTracerContext(),
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
		return fmt.Errorf("host from database = (Name: %s, Status: %s), want = (Name: %s, Status: %s)",
			response.HostName, response.MonitorStatus, TestHostName, mustHasStatus)
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
