package k8s

import (
	"os"
	"testing"

	"github.com/gwos/tcg/config"
	"github.com/stretchr/testify/assert"
)

// TestConfigHandlerReadsRetryConnection feeds the real monitorConnection payload and
// checks the UI "Retries" setting drives how many failed cycles mark the connector stopped.
func TestConfigHandlerReadsRetryConnection(t *testing.T) {
	tmpFile, _ := os.CreateTemp("", "config")
	t.Setenv(config.ConfigEnv, tmpFile.Name())
	defer os.Remove(tmpFile.Name())

	data := []byte(`{
	  "monitorConnection": {
	    "server": "",
	    "userName": "",
	    "password": "",
	    "sslEnabled": true,
	    "id": 5,
	    "url": "",
	    "extensions": {
	      "mapHostname": [],
	      "mapHostgroup": [],
	      "mapService": [],
	      "checkIntervalMinutes": 1,
	      "checkTimeoutSeconds": 20,
	      "retryConnection": 2,
	      "authType": "BearerToken",
	      "kubernetesClusterEndpoint": "https://cluster.example:6443",
	      "kubernetesBearerToken": "some-token"
	    }
	  },
	  "metricsProfile": {"metrics": []}
	}`)

	stub := &transportStub{running: true}
	stub.install(t)

	_, _ = config.GetConfig().LoadConnectorDTO(data)
	configHandler(data)

	assert.Equal(t, 2, connector.ExtConfig.RetryConnection, "retryConnection must be parsed")
	assert.Equal(t, 2, maxFailures, "the parsed value must drive the stop threshold")

	/* two failed cycles, as configured, and the connector must go stopped */
	markCollectFailed(otherErr())
	assert.Equal(t, 0, stub.stops, "one failure is below the configured retries")

	markCollectFailed(otherErr())
	assert.Equal(t, 1, stub.stops, "the second failure must stop the connector")
	assert.False(t, stub.running)
}
