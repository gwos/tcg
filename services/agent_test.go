package services

import (
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gwos/tcg/config"
	"github.com/gwos/tcg/sdk/clients"
	tcgerr "github.com/gwos/tcg/sdk/errors"
	"github.com/stretchr/testify/assert"
)

func init() {
	config.GetConfig().Connector.ControllerAddr = ":11099"
	config.GetConfig().Connector.NatsStoreType = "MEMORY"
	config.GetConfig().Connector.NatsStoreMaxBytes = 1024000
	config.GetConfig().GWConnections = []config.GWConnection{
		{
			Enabled:         true,
			LocalConnection: false,
			HostName:        "test",
			UserName:        "test",
			Password:        "test",
		},
	}
}

func TestAgentService(t *testing.T) {
	t.Cleanup(func() {
		assert.NoError(t, os.RemoveAll(filepath.Join(GetAgentService().Connector.NatsStoreDir, "jetstream")))
		assert.NoError(t, os.RemoveAll(filepath.Join(GetController().Connector.NatsStoreDir, "inventory.json")))
		assert.NoError(t, os.RemoveAll(filepath.Join(GetController().Connector.NatsStoreDir, "inventory1.json")))
		assert.NoError(t, os.Remove(GetAgentService().Connector.NatsStoreDir))
	})

	t.Run("Controller", func(t *testing.T) {
		addr := "127.0.0.1" + GetController().Connector.ControllerAddr
		for i := 0; i < 5; i++ {
			assert.NoError(t, GetAgentService().StartController())
			assert.NotNil(t, GetController().srv.Load(), "running after start #%d", i)
			if conn, err := net.DialTimeout("tcp", addr, time.Second); assert.NoError(t, err) {
				_ = conn.Close()
			}
			assert.NoError(t, GetAgentService().StopController())
			assert.Nil(t, GetController().srv.Load(), "stopped after stop #%d", i)
		}
	})

	t.Run("NATS", func(t *testing.T) {
		assert.NoError(t, GetAgentService().StartNats())
		assert.NoError(t, GetAgentService().StopNats())
		assert.NoError(t, GetAgentService().StartNats())
		assert.NoError(t, GetAgentService().StopNats())
	})

	t.Run("Transport", func(t *testing.T) {
		GetAgentService().Connector.AgentID = "TESTAGENTID"
		GetAgentService().Connector.AppType = "TESTAPPTYPE"
		assert.NoError(t, GetAgentService().StartNats())
		assert.NoError(t, GetAgentService().StartTransport())
		assert.NoError(t, GetAgentService().StopTransport())
		assert.NoError(t, GetAgentService().StartTransport())
		assert.NoError(t, GetAgentService().StopNats())
	})

	t.Run("DemandConfig", func(t *testing.T) {
		tmpfile, err := os.CreateTemp("", "config")
		assert.NoError(t, err)
		err = tmpfile.Close()
		assert.NoError(t, err)
		defer os.Remove(tmpfile.Name())
		t.Setenv(config.ConfigEnv, tmpfile.Name())
		t.Setenv("TCG_CONNECTOR_NATSSTOREMAXBYTES", "333_222_111_000")

		dto := []byte(`
{
  "agentId": "99998888-7777-6666-a3b0-b14622f7dd39",
  "appName": "test-app-XX",
  "appType": "test-XX",
  "logLevel": 2,
  "tcgUrl": "http://tcg-host:9980/",
  "dalekservicesConnection": {
    "hostName": "gw-host-xxx"
  },
  "groundworkConnections": [{
	"enabled": true,
	"localConnection": false,
    "hostName": "gw-host-xx",
    "userName": "-xx-",
    "password": "xx"
  }]
}`)

		agentService := GetAgentService()
		assert.Equal(t, "TESTAGENTID", agentService.Connector.AgentID)
		assert.NoError(t, agentService.config(dto))
		assert.Equal(t, "99998888-7777-6666-a3b0-b14622f7dd39", agentService.Connector.AgentID)
		assert.Equal(t, "gw-host-xxx", agentService.dsClient.HostName)
		assert.NoError(t, agentService.startNats())
		assert.NoError(t, agentService.startTransport())
		assert.Equal(t, "gw-host-xx", agentService.gwClients[0].HostName)
	})
}

// newDSStub starts a TLS stub that stands in for DalekServices and counts the reload requests it receives.
// makeDalekServicesScheme derives https for any host is not prefixed with "dalekservices", so a TLS server matches.
func newDSStub(t *testing.T, handler http.HandlerFunc) (host string, count *int32) {
	t.Helper()
	count = new(int32)
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(count, 1)
		handler(w, r)
	}))
	prev := clients.HttpClientTransport.TLSClientConfig.InsecureSkipVerify
	clients.HttpClientTransport.TLSClientConfig.InsecureSkipVerify = true
	t.Cleanup(func() {
		clients.HttpClientTransport.TLSClientConfig.InsecureSkipVerify = prev
		srv.Close()
	})
	return strings.TrimPrefix(srv.URL, "https://"), count
}

// TestDemandConfig covers the guard and retry behavior that keeps the connector
// from hammering DalekServices while its agent is not configured.
func TestDemandConfig(t *testing.T) {
	svc := GetAgentService()
	t.Cleanup(func() { _ = svc.StopController() })

	// shrink the retry backoff so the loop can be observed quickly
	prevBackoff := demandConfigBackoff
	demandConfigBackoff = func(int) time.Duration { return 10 * time.Millisecond }
	t.Cleanup(func() { demandConfigBackoff = prevBackoff })

	t.Run("Guard skips reload for empty/placeholder AgentID", func(t *testing.T) {
		host, count := newDSStub(t, func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusCreated)
		})
		for _, id := range []string{"", placeholderAgentID} {
			atomic.StoreInt32(count, 0)
			svc.Connector.AgentID = id
			svc.dsClient.HostName = host
			assert.NoError(t, svc.DemandConfig())
			time.Sleep(200 * time.Millisecond)
			assert.Equal(t, int32(0), atomic.LoadInt32(count),
				"expected no reload request for AgentID=%q", id)
		}
	})

	t.Run("NotFound stops retrying", func(t *testing.T) {
		host, count := newDSStub(t, func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusNotFound)
		})
		svc.Connector.AgentID = "unknown-agent-id"
		svc.dsClient.HostName = host

		assert.NoError(t, svc.DemandConfig())
		// window is much larger than the 10ms backoff: a retrying loop would
		// send many requests, the fixed loop sends exactly one and stops.
		time.Sleep(300 * time.Millisecond)
		assert.Equal(t, int32(1), atomic.LoadInt32(count),
			"expected exactly one reload request on 404, then stop")
	})

	t.Run("Transient error retries until success", func(t *testing.T) {
		var attempts int32
		host, count := newDSStub(t, func(w http.ResponseWriter, _ *http.Request) {
			if atomic.AddInt32(&attempts, 1) < 3 {
				w.WriteHeader(http.StatusInternalServerError) // ErrUndecided -> retry
				return
			}
			w.WriteHeader(http.StatusCreated) // success on the 3rd attempt
		})
		svc.Connector.AgentID = "some-agent-id"
		svc.dsClient.HostName = host

		assert.NoError(t, svc.DemandConfig())
		assert.Eventually(t, func() bool { return atomic.LoadInt32(count) == 3 },
			2*time.Second, 10*time.Millisecond, "expected retries until success")
		// once connected it must stop issuing requests
		time.Sleep(100 * time.Millisecond)
		assert.Equal(t, int32(3), atomic.LoadInt32(count),
			"expected no further reload after success")
	})
}

// TestDemandConfigControllerUnavailable locks in that a controller which
// cannot listen does not stop the connector from running on its local config.
func TestDemandConfigControllerUnavailable(t *testing.T) {
	svc := GetAgentService()
	conn := svc.Connector
	prevAddr, prevTimeout, prevAgentID := conn.ControllerAddr, conn.ControllerStartTimeout, conn.AgentID
	t.Cleanup(func() {
		_ = svc.StopController()
		conn.ControllerAddr, conn.ControllerStartTimeout, conn.AgentID = prevAddr, prevTimeout, prevAgentID
	})

	// hold the port without SO_REUSEPORT so the controller cannot bind it
	busy, err := net.Listen("tcp", "127.0.0.1:0")
	assert.NoError(t, err)
	t.Cleanup(func() { _ = busy.Close() })
	conn.ControllerAddr = busy.Addr().String()
	conn.ControllerStartTimeout = 300 * time.Millisecond
	conn.AgentID = ""

	assert.Error(t, svc.StartController())
	assert.NoError(t, svc.DemandConfig())
}

// TestControllerTLSKeyPairError locks in that the listener is released when
// ServeTLS fails to load the key pair, so no orphan socket shares the port.
func TestControllerTLSKeyPairError(t *testing.T) {
	svc := GetAgentService()
	conn := svc.Connector
	prevAddr, prevCert, prevKey := conn.ControllerAddr, conn.ControllerCertFile, conn.ControllerKeyFile
	t.Cleanup(func() {
		_ = svc.StopController()
		conn.ControllerAddr, conn.ControllerCertFile, conn.ControllerKeyFile = prevAddr, prevCert, prevKey
	})

	conn.ControllerAddr = "127.0.0.1:11098"
	conn.ControllerCertFile = filepath.Join(t.TempDir(), "missing.crt")
	conn.ControllerKeyFile = filepath.Join(t.TempDir(), "missing.key")

	assert.NoError(t, svc.StartController())
	assert.Eventually(t, func() bool { return GetController().srv.Load() == nil },
		2*time.Second, 10*time.Millisecond, "expected server cleared after ServeTLS error")
	// a plain listener cannot share the port with a leaked SO_REUSEPORT socket
	l, err := net.Listen("tcp", conn.ControllerAddr)
	if assert.NoError(t, err, "expected the port to be released") {
		_ = l.Close()
	}
}

// TestStartTransportPlaceholderAgentID locks in that the placeholder AgentID
// is treated as "not configured" everywhere via isConnectorConfigured,
// not only in the DemandConfig guard.
func TestStartTransportPlaceholderAgentID(t *testing.T) {
	svc := GetAgentService()
	prevAgentID, prevAppType := svc.Connector.AgentID, svc.Connector.AppType
	t.Cleanup(func() {
		svc.Connector.AgentID, svc.Connector.AppType = prevAgentID, prevAppType
	})

	svc.Connector.AgentID = placeholderAgentID
	svc.Connector.AppType = "test-XX" // non-NAGIOS so the guard returns an error
	assert.ErrorIs(t, svc.startTransport(), tcgerr.ErrNotConfigured)
}
