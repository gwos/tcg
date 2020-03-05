package services

import (
	"github.com/gwos/tng/config"
	"github.com/stretchr/testify/assert"
	"net/http"
	"net/http/httptest"
	"testing"
)

func init() {
	config.GetConfig().Connector.AppName = "test"
	config.GetConfig().Connector.NatsStoreType = "MEMORY"
	config.GetConfig().GWConnections = []*config.GWConnection{
		{
			HostName: "test",
			UserName: "test",
			Password: "test",
			Enabled: true,
		},
	}
}

func TestController_StartStopNats(t *testing.T) {
	controller := GetController()
	testServer := httptest.NewServer(http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		switch req.URL.String() {
		case "/nats/start":
			assert.NoError(t, controller.StartNats())
			assert.Equal(t, string(controller.Status().Nats), Running,
				"nats server status should match the expected one [Running]")
		case "/nats/stop":
			assert.NoError(t, controller.StopNats())
			assert.Equal(t, string(controller.Status().Nats), Stopped,
				"nats server status should match the expected one [Stopped]")
		}
		res.WriteHeader(http.StatusOK)
	}))

	startReq, err := http.NewRequest(http.MethodGet, testServer.URL+"/nats/start", nil)
	assert.NoError(t, err)

	stopReq, err := http.NewRequest(http.MethodGet, testServer.URL+"/nats/stop", nil)
	assert.NoError(t, err)

	res, err := http.DefaultClient.Do(startReq)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, res.StatusCode, "status code should match the expected response")

	res, err = http.DefaultClient.Do(stopReq)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, res.StatusCode, "status code should match the expected response")
}

func TestController_StartStopTransport(t *testing.T) {
	controller := GetController()
	testServer := httptest.NewServer(http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		assert.NoError(t, controller.StartNats())

		switch req.URL.String() {
		case "/nats/transport/start":
			assert.NoError(t, controller.StartTransport())
			assert.Equal(t, string(controller.Status().Transport), Running,
				"nats transport status should match the expected one [Running]")
		case "/nats/transport/stop":
			assert.NoError(t, controller.StopTransport())
			assert.Equal(t, string(controller.Status().Transport), Stopped,
				"nats transport status should match the expected one [Stopped]")
		}
		res.WriteHeader(http.StatusOK)
	}))

	startReq, err := http.NewRequest(http.MethodGet, testServer.URL+"/nats/transport/start", nil)
	assert.NoError(t, err)

	stopReq, err := http.NewRequest(http.MethodGet, testServer.URL+"/nats/transport/stop", nil)
	assert.NoError(t, err)

	res, err := http.DefaultClient.Do(startReq)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, res.StatusCode, "status code should match the expected response")

	res, err = http.DefaultClient.Do(stopReq)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, res.StatusCode, "status code should match the expected response")
}
