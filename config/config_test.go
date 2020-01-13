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
connector:
  agentId: "3dcd9a52-949d-4531-a3b0-b14622f7dd39"
  appName: "test-app"
  appType: "test"
  controllerAddr: ":8081"
  natsAckWait: 15
dsConnection:
  hostName: "localhost:3001"
  userName: "RESTAPIACCESS"
  password: ""
gwConnections:
  -
    hostName: "localhost:80"
    userName: "RESTAPIACCESS"
    password: ""
`)

	tmpfile, err := ioutil.TempFile("", "config")
	defer os.Remove(tmpfile.Name())
	assert.NoError(t, err)
	_, err = tmpfile.Write(configYAML)
	assert.NoError(t, err)
	err = tmpfile.Close()
	assert.NoError(t, err)

	os.Setenv(string(ConfigEnv), tmpfile.Name())
	os.Setenv("TNG_CONNECTOR_NATSSTORETYPE", "MEMORY")
	os.Setenv("TNG_DSCONNECTION", "{\"password\":\"SECRET\"}")
	os.Setenv("TNG_GWCONNECTIONS", "[{\"password\":\"SEC RET\"},{\"hostName\":\"localhost:3001\"}]")

	expected := Config{
		Connector: &Connector{
			AgentID:          "3dcd9a52-949d-4531-a3b0-b14622f7dd39",
			AppName:          "test-app",
			AppType:          "test",
			ControllerAddr:   ":8081",
			LogLevel:         1,
			NatsAckWait:      15,
			NatsMaxInflight:  2147483647,
			NatsFilestoreDir: "natsstore",
			NatsStoreType:    "MEMORY",
			NatsHost:         ":4222",
		},
		DSConnection: &DSConnection{"localhost:3001", "RESTAPIACCESS", "SECRET"},
		GWConnections: GWConnections{
			&GWConnection{"localhost:80", "RESTAPIACCESS", "SEC RET"},
			&GWConnection{HostName: "localhost:3001"},
		},
	}

	got := GetConfig()

	if !reflect.DeepEqual(got.Connector, expected.Connector) {
		t.Errorf("got: %v, expected: %v", got.Connector, expected.Connector)
	}

	if !reflect.DeepEqual(got.DSConnection, expected.DSConnection) {
		t.Errorf("got: %v, expected: %v", got.DSConnection, expected.DSConnection)
	}

	if len(got.GWConnections) != len(expected.GWConnections) {
		t.Errorf("got: %v, expected: %v", len(got.GWConnections), len(expected.GWConnections))
	}

	for k, v := range got.GWConnections {
		if !reflect.DeepEqual(v, expected.GWConnections[k]) {
			t.Errorf("key: %v, got: %v, expected: %v", k, v, expected.GWConnections[k])
		}
	}
}
