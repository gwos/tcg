package services

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/gwos/tcg/config"
	"github.com/stretchr/testify/assert"
)

func init() {
	config.GetConfig().Connector.AppName = "test"
	config.GetConfig().Connector.NatsStoreType = "MEMORY"
	config.GetConfig().GWConnections = []*config.GWConnection{
		{
			Enabled:         true,
			LocalConnection: false,
			HostName:        "test",
			UserName:        "test",
			Password:        "test",
		},
	}
}

func TestController(t *testing.T) {
	t.Cleanup(func() {
		assert.NoError(t, os.RemoveAll(filepath.Join(GetAgentService().Connector.NatsStoreDir, "jetstream")))
		assert.NoError(t, os.Remove(GetAgentService().Connector.NatsStoreDir))
	})

	t.Run("NATS", func(t *testing.T) {
		controller := GetController()
		testServer := httptest.NewServer(http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
			switch req.URL.String() {
			case "/nats/start":
				assert.NoError(t, controller.StartNats())
				assert.Equal(t, StatusRunning, controller.Status().Nats,
					"nats server status should match the expected one [Running]")
			case "/nats/stop":
				assert.NoError(t, controller.StopNats())
				assert.Equal(t, StatusStopped, controller.Status().Nats,
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
		defer res.Body.Close()
		assert.Equal(t, http.StatusOK, res.StatusCode, "status code should match the expected response")

		res, err = http.DefaultClient.Do(stopReq)
		assert.NoError(t, err)
		defer res.Body.Close()
		assert.Equal(t, http.StatusOK, res.StatusCode, "status code should match the expected response")
	})

	t.Run("Transport", func(t *testing.T) {
		controller := GetController()
		testServer := httptest.NewServer(http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
			assert.NoError(t, controller.StartNats())

			switch req.URL.String() {
			case "/nats/transport/start":
				assert.NoError(t, controller.StartTransport())
				assert.Equal(t, StatusRunning, controller.Status().Transport,
					"nats transport status should match the expected one [Running]")
			case "/nats/transport/stop":
				assert.NoError(t, controller.StopTransport())
				assert.Equal(t, StatusStopped, controller.Status().Transport,
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
		defer res.Body.Close()
		assert.Equal(t, http.StatusOK, res.StatusCode, "status code should match the expected response")

		res, err = http.DefaultClient.Do(stopReq)
		assert.NoError(t, err)
		defer res.Body.Close()
		assert.Equal(t, http.StatusOK, res.StatusCode, "status code should match the expected response")
	})
}
