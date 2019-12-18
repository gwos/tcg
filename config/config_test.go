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
gwConfigs:
  -
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

	os.Setenv(string(ConfigEnv), tmpfile.Name())
	os.Setenv("TNG_AGENTCONFIG_NATSSTORETYPE", "MEMORY")
	os.Setenv("TNG_GWCONFIGS", "[{\"password\":\"SEC RET\"},{\"appName\":\"gw8\"}]")

	expected := Config{
		AgentConfig: &AgentConfig{
			ControllerAddr:   ":8081",
			LogLevel:         1,
			NatsAckWait:      15,
			NatsMaxInflight:  2147483647,
			NatsFilestoreDir: "natsstore",
			NatsStoreType:    "MEMORY",
			NatsHost:         ":4222",
		},
		GWConfigs: GWConfigs{
			&GWConfig{"localhost:80", "RESTAPIACCESS", "SEC RET", "gw8"},
			&GWConfig{AppName: "gw8"},
		},
	}

	got := GetConfig()

	if !reflect.DeepEqual(got.AgentConfig, expected.AgentConfig) {
		t.Errorf("got: %v, expected: %v", got.AgentConfig, expected.AgentConfig)
	}

	if len(got.GWConfigs) != len(expected.GWConfigs) {
		t.Errorf("got: %v, expected: %v", len(got.GWConfigs), len(expected.GWConfigs))
	}

	for k, v := range got.GWConfigs {
		if !reflect.DeepEqual(v, expected.GWConfigs[k]) {
			t.Errorf("key: %v, got: %v, expected: %v", k, v, expected.GWConfigs[k])
		}
	}
}
