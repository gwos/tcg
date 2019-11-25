package config

import (
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"os"
	"reflect"
	"testing"
)

func TestGetConfig(t *testing.T) {
	configYAML := []byte(`
agentConfig:
  controllerAddr: ":8081"
  natsAckWait: 15
  natsFilestoreDir: "datastore"
  natsStoreType: "FILE"
  natsHost: ":4222"
  startNats: True
  startTransport: True
  startController: True
  loggingLevel: 0
gwConfig:
  host: "localhost:80"
  account: "RESTAPIACCESS"
  password: ""
  appName: "gw8"
`)

	tmpfile, err := ioutil.TempFile("", "config")
	defer os.Remove(tmpfile.Name())
	assert.NoError(t, err)
	_, err = tmpfile.Write(configYAML)
	assert.NoError(t, err)
	err = tmpfile.Close()
	assert.NoError(t, err)

	os.Setenv(ConfigEnv, tmpfile.Name())
	os.Setenv("TNG_AGENTCONFIG_NATSSTORETYPE", "MEMORY")
	os.Setenv("TNG_GWCONFIG_PASSWORD", "SECRET")

	expected := Config{
		AgentConfig: &AgentConfig{":8081", "", "", 15, "datastore", "MEMORY", ":4222", true, true, true, 0},
		GWConfig:    &GWConfig{"localhost:80", "RESTAPIACCESS", "SECRET", "gw8"},
	}

	got := GetConfig()
	if !reflect.DeepEqual(got.AgentConfig, expected.AgentConfig) {
		t.Errorf("got %v, expected %v", got.AgentConfig, expected.AgentConfig)
	}
	if !reflect.DeepEqual(got.GWConfig, expected.GWConfig) {
		t.Errorf("got %v, expected %v", got.GWConfig, expected.GWConfig)
	}
}
