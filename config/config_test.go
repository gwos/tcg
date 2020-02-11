package config

import (
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"os"
	"reflect"
	"testing"
)

func checkExpected(t *testing.T, expected *Config) {
	t.Helper()

	got := GetConfig()
	if !reflect.DeepEqual(got.Connector, expected.Connector) {
		t.Errorf("Connector got: %v, expected: %v", got.Connector, expected.Connector)
	}
	if !reflect.DeepEqual(got.DSConnection, expected.DSConnection) {
		t.Errorf("DSConnection got: %v, expected: %v", got.DSConnection, expected.DSConnection)
	}
	if len(got.GWConnections) != len(expected.GWConnections) {
		t.Errorf("GWConnections got: %v, expected: %v", len(got.GWConnections), len(expected.GWConnections))
	}
	for k, v := range got.GWConnections {
		if !reflect.DeepEqual(v, expected.GWConnections[k]) {
			t.Errorf("GWConnections key: %v, got: %v, expected: %v", k, v, expected.GWConnections[k])
		}
	}
}

func TestGetConfig(t *testing.T) {
	configYAML := []byte(`
connector:
  agentId: "3dcd9a52-949d-4531-a3b0-b14622f7dd39"
  appName: "test-app"
  appType: "test"
  controllerAddr: ":8099"
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

	expected := &Config{
		Connector: &Connector{
			AgentID:          "3dcd9a52-949d-4531-a3b0-b14622f7dd39",
			AppName:          "test-app",
			AppType:          "test",
			ControllerAddr:   ":8099",
			LogLevel:         1,
			NatsAckWait:      15,
			NatsMaxInflight:  2147483647,
			NatsFilestoreDir: "natsstore",
			NatsStoreType:    "MEMORY",
			NatsHost:         ":4222",
		},
		DSConnection: &DSConnection{"localhost:3001", "RESTAPIACCESS", "SECRET"},
		GWConnections: GWConnections{
			&GWConnection{HostName: "localhost:80", UserName: "RESTAPIACCESS", Password: "SEC RET"},
			&GWConnection{HostName: "localhost:3001"},
		},
	}

	checkExpected(t, expected)
}

func TestLoadConnectorDTO(t *testing.T) {
	expected := &Config{
		Connector: &Connector{
			AgentID:          "11112222-3333-4444-a3b0-b14622f7dd39",
			AppName:          "test-app",
			AppType:          "test",
			ControllerAddr:   "0.0.0.0:8099",
			LogLevel:         0,
			NatsAckWait:      15,
			NatsMaxInflight:  2147483647,
			NatsFilestoreDir: "natsstore",
			NatsStoreType:    "MEMORY",
			NatsHost:         ":4222",
		},
		DSConnection: &DSConnection{"localhost:3001", "RESTAPIACCESS", "SECRET"},
		GWConnections: GWConnections{
			&GWConnection{HostName: "gw-host1"},
			&GWConnection{HostName: "gw-host2"},
		},
	}
	cfg := GetConfig()
	cfg.Connector = expected.Connector
	cfg.GWConnections = expected.GWConnections

	checkExpected(t, expected)

	dto := []byte(`
{
  "agentId": "99998888-7777-6666-a3b0-b14622f7dd39",
  "appName": "test-app-XX",
  "appType": "test-XX",
  "logLevel": 2,
  "tngUrl": "http://tng-host:9980/",
  "groundworkConnections": [{
    "hostName": "gw-host-xx",
    "userName": "-xx-",
    "password": "xx"
  }]
}`)

	_, err := cfg.LoadConnectorDTO(dto)
	assert.NoError(t, err)

	expected = &Config{
		Connector: &Connector{
			AgentID:          "99998888-7777-6666-a3b0-b14622f7dd39",
			AppName:          "test-app-XX",
			AppType:          "test-XX",
			ControllerAddr:   "0.0.0.0:9980",
			LogLevel:         2,
			NatsAckWait:      15,
			NatsMaxInflight:  2147483647,
			NatsFilestoreDir: "natsstore",
			NatsStoreType:    "MEMORY",
			NatsHost:         ":4222",
		},
		DSConnection: &DSConnection{"localhost:3001", "RESTAPIACCESS", "SECRET"},
		GWConnections: GWConnections{
			&GWConnection{HostName: "gw-host-xx", UserName: "-xx-", Password: "xx"},
		},
	}

	checkExpected(t, expected)
	// ss, err := json.Marshal(GetConfig())
	// t.Logf("%v", ss)
}
