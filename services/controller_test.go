package services

import (
	"github.com/stretchr/testify/assert"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path"
	"testing"
)

const (
	ConfigEnv  = "TNG_CONFIG"
	ConfigName = "config.yml"
)

func TestController_StartStopNats(t *testing.T) {
	testServer := httptest.NewServer(http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		assert.NoError(t, os.Setenv(ConfigEnv, path.Join("..", ConfigName)))
		service := GetTransitService()

		switch req.URL.String() {
		case "/nats/start":
			err := service.StartNats()
			assert.NoError(t, err)
			assert.Equal(t, string(service.Status().Nats), "Running", "nats server status should"+
				" match the expected one(\"Running\"")
		case "/nats/stop":
			err := service.StopNats()
			assert.NoError(t, err)
			assert.Equal(t, string(service.Status().Nats), "Stopped", "nats server status should"+
				" match the expected one(\"Stopped\"")
		}
		res.WriteHeader(http.StatusOK)
	}))

	defer func() {
		testServer.Close()
		cmd := exec.Command("rm", "-rf", "src")
		_, err := cmd.Output()
		if err != nil {
			log.Println(err)
		}
	}()

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
	assert.NoError(t, os.Setenv(ConfigEnv, path.Join("..", ConfigName)))
	service := GetTransitService()
	testServer := httptest.NewServer(http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		err := service.StartNats()
		assert.NoError(t, err)

		switch req.URL.String() {
		case "/nats/transport/start":
			err := service.StartTransport()
			assert.NoError(t, err)
			assert.Equal(t, string(service.Status().Transport), "Running", "nats transport status"+
				" should match the expected one(\"Running\"")
		case "/nats/transport/stop":
			err := service.StopTransport()
			assert.NoError(t, err)
			assert.Equal(t, string(service.Status().Transport), "Stopped", "nats transport status"+
				" should match the expected one(\"Stopped\"")
		}
		res.WriteHeader(http.StatusOK)
	}))

	defer func() {
		testServer.Close()
		_ = service.StopNats()
		cmd := exec.Command("rm", "-rf", "src")
		_, err := cmd.Output()
		if err != nil {
			log.Println(err)
		}
	}()

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
