package services

import (
	"github.com/gwos/tng/config"
	"github.com/stretchr/testify/assert"
	"net/http"
	"net/http/httptest"
	"os"
	"path"
	"testing"
)

const (
	ConfigEnv  = "TNG_CONFIG"
	ConfigName = "tng_config.yaml"
)

func TestController_StartStopNats(t *testing.T) {
	testServer := httptest.NewServer(http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		assert.NoError(t, os.Setenv(ConfigEnv, path.Join("..", ConfigName)))
		service := GetTransitService()
		service.NatsStoreType = "MEMORY"

		switch req.URL.String() {
		case "/nats/start":
			assert.NoError(t, service.StartNats())
			assert.Equal(t, string(service.Status().Nats), "Running", "nats server status should"+
				" match the expected one(\"Running\"")
		case "/nats/stop":
			assert.NoError(t, service.StopNats())
			assert.Equal(t, string(service.Status().Nats), "Stopped", "nats server status should"+
				" match the expected one(\"Stopped\"")
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
	if os.Getenv(ConfigEnv) == "" {
		assert.NoError(t, os.Setenv(ConfigEnv, path.Join("..", ConfigName)))
	}
	service := GetTransitService()
	service.NatsAckWait = 15
	service.NatsStoreType = "MEMORY"
	config.GetConfig().GWConfigs = []*config.GWConfig{
		{
			Host:     "test",
			Account:  "test",
			Password: "test",
			AppName:  "test",
		},
	}
	testServer := httptest.NewServer(http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		assert.NoError(t, service.StartNats())

		switch req.URL.String() {
		case "/nats/transport/start":
			assert.NoError(t, service.StartTransport())
			assert.Equal(t, string(service.Status().Transport), "Running", "nats transport status"+
				" should match the expected one(\"Running\"")
		case "/nats/transport/stop":
			assert.NoError(t, service.StopTransport())
			assert.Equal(t, string(service.Status().Transport), "Stopped", "nats transport status"+
				" should match the expected one(\"Stopped\"")
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
