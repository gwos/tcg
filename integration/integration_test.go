package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os/exec"
	"reflect"
	"testing"
	"time"

	"github.com/gwos/tcg/config"
	"github.com/gwos/tcg/sdk/clients"
	"github.com/gwos/tcg/sdk/milliseconds"
	"github.com/gwos/tcg/sdk/transit"
	"github.com/gwos/tcg/services"
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
	setupIntegration(t, 5*time.Second)
	headers, err = connect(t)
	defer cleanNats(t)
	defer clean(t, headers)
	assert.NoError(t, err)

	t.Log("Check for host availability in the database")
	time.Sleep(1 * time.Second)
	assert.NoError(t, existenceCheck(t, false, "irrelevant"))

	t.Log("Send SynchronizeInventory request to GroundWork Foundation")
	assert.NoError(t, services.GetTransitService().SynchronizeInventory(context.Background(), buildInventoryRequest(t)))

	time.Sleep(5 * time.Second)
	t.Log("Check for host availability in the database")
	time.Sleep(1 * time.Second)
	assert.NoError(t, existenceCheck(t, true, HostStatusPending))

	t.Log("Send ResourcesWithMetrics request to GroundWork Foundation")
	assert.NoError(t, services.GetTransitService().SendResourceWithMetrics(context.Background(), buildResourceWithMetricsRequest(t)))

	time.Sleep(5 * time.Second)

	t.Log("Check for host availability in the database")
	time.Sleep(1 * time.Second)
	assert.NoError(t, existenceCheck(t, true, HostStatusUp))

	t.Log("Send bad ResourcesWithMetrics payload to GroundWork Foundation")
	/* expect foundation error, processing should not stop */
	badPayload := bytes.ReplaceAll(buildResourceWithMetricsRequest(t),
		[]byte(`context`), []byte(`*ontex*`))
	assert.NoError(t, services.GetTransitService().SendResourceWithMetrics(context.Background(), badPayload))
	assert.Equal(t, services.StatusRunning, services.GetTransitService().Status().Nats)
	assert.Equal(t, services.StatusRunning, services.GetTransitService().Status().Transport)
}

func buildInventoryRequest(t *testing.T) []byte {
	inventoryResource := transit.InventoryResource{
		BaseResource: transit.BaseResource{
			BaseTransitData: transit.BaseTransitData{
				Name: TestHostName,
				Type: transit.ResourceTypeHost,
			},
		},
		Services: []transit.InventoryService{
			{
				BaseTransitData: transit.BaseTransitData{
					Name:  "test",
					Type:  transit.ResourceTypeHypervisor,
					Owner: TestHostName,
				},
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
		BaseResource: transit.BaseResource{
			BaseTransitData: transit.BaseTransitData{
				Name: TestHostName,
				Type: transit.ResourceTypeHost,
			},
		},
		Status:        transit.HostUp,
		LastCheckTime: milliseconds.MillisecondTimestamp{Time: time.Now()},
		NextCheckTime: milliseconds.MillisecondTimestamp{Time: time.Now()},
		Services: []transit.MonitoredService{
			{
				BaseTransitData: transit.BaseTransitData{
					Name:  "test",
					Type:  transit.ResourceTypeService,
					Owner: TestHostName,
				},
				Status:        transit.ServiceOk,
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

func connect(t assert.TestingT) (map[string]string, error) {
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

	return headers, nil
}

func existenceCheck(t *testing.T, mustExist bool, mustHasStatus string) error {
	statusCode, byteResponse, err := clients.SendRequest(http.MethodGet, HostGetAPI+TestHostName, headers, nil, nil)
	if err != nil {
		return err
	}
	if statusCode == 200 {
		t.Log(" -> Host exists")
	} else {
		t.Log(" -> Host doesn't exist")
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

func clean(t *testing.T, headers map[string]string) {
	_, _, err := clients.SendRequest(http.MethodDelete, HostDeleteAPI+TestHostName, headers, nil, nil)
	assert.NoError(t, err)

	cmd := exec.Command("rm", "-rf", config.GetConfig().Connector.NatsStoreDir)
	_, err = cmd.Output()
	assert.NoError(t, err)
}
