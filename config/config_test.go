package config

import (
	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"
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
  hostName: "localhost"
gwConnections:
  -
    enabled: true
    localConnection: false
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
	os.Setenv("TNG_DSCONNECTION", "{\"hostName\":\"localhost:3001\"}")
	os.Setenv("TNG_GWCONNECTIONS", "[{\"password\":\"SEC RET\"},{\"hostName\":\"localhost:3001\"}]")

	expected := &Config{
		Connector: &Connector{
			AgentID:          "3dcd9a52-949d-4531-a3b0-b14622f7dd39",
			AppName:          "test-app",
			AppType:          "test",
			ControllerAddr:   ":8099",
			LogConsPeriod:    0,
			LogLevel:         1,
			NatsAckWait:      15,
			NatsMaxInflight:  2147483647,
			NatsFilestoreDir: "natsstore",
			NatsStoreType:    "MEMORY",
			NatsHost:         "127.0.0.1:4222",
		},
		DSConnection: &DSConnection{"localhost:3001"},
		GWConnections: GWConnections{
			&GWConnection{Enabled: true, LocalConnection: false, HostName: "localhost:80", UserName: "RESTAPIACCESS", Password: "SEC RET"},
			&GWConnection{HostName: "localhost:3001"},
		},
	}

	checkExpected(t, expected)
}

func TestLoadConnectorDTO(t *testing.T) {
	tmpfile, err := ioutil.TempFile("", "config")
	assert.NoError(t, err)
	err = tmpfile.Close()
	assert.NoError(t, err)
	defer os.Remove(tmpfile.Name())
	os.Setenv(string(ConfigEnv), tmpfile.Name())

	expected := &Config{
		Connector: &Connector{
			AgentID:          "11112222-3333-4444-a3b0-b14622f7dd39",
			AppName:          "test-app",
			AppType:          "test",
			ControllerAddr:   "0.0.0.0:8099",
			LogConsPeriod:    0,
			LogLevel:         0,
			NatsAckWait:      15,
			NatsMaxInflight:  2147483647,
			NatsFilestoreDir: "natsstore",
			NatsStoreType:    "MEMORY",
			NatsHost:         ":4222",
		},
		DSConnection: &DSConnection{"localhost:3001"},
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
  "logConsPeriod": 30,
  "logLevel": 2,
  "tngUrl": "http://tng-host/",
  "dalekservicesConnection": {
    "hostName": "gw-host-xxx"
  },
  "groundworkConnections": [{
    "id": 101,
    "enabled": true,
    "localConnection": false,
    "hostName": "gw-host-xx",
    "userName": "-xx-",
    "password": "xx"
  }],
  "advanced": {
    "prefixes": [{
      "groundworkConnectionId": 101,
      "prefix": "c1"
    }]
  }
}`)

	_, err = cfg.LoadConnectorDTO(dto)
	assert.NoError(t, err)

	expected = &Config{
		Connector: &Connector{
			AgentID:          "99998888-7777-6666-a3b0-b14622f7dd39",
			AppName:          "test-app-XX",
			AppType:          "test-XX",
			ControllerAddr:   "0.0.0.0:8099",
			LogConsPeriod:    30,
			LogLevel:         2,
			NatsAckWait:      15,
			NatsMaxInflight:  2147483647,
			NatsFilestoreDir: "natsstore",
			NatsStoreType:    "MEMORY",
			NatsHost:         ":4222",
		},
		DSConnection: &DSConnection{"gw-host-xxx"},
		GWConnections: GWConnections{
			&GWConnection{
				ID:                  101,
				Enabled:             true,
				LocalConnection:     false,
				HostName:            "gw-host-xx",
				UserName:            "-xx-",
				Password:            "xx",
				PrefixResourceNames: true,
				ResourceNamePrefix:  "c1",
			},
		},
	}

	checkExpected(t, expected)
	// ss, err := json.Marshal(GetConfig())
	// t.Logf("%v", ss)

	data, err := ioutil.ReadFile(tmpfile.Name())
	assert.Contains(t, string(data), "99998888-7777-6666-a3b0-b14622f7dd39")
}

func TestMarshaling(t *testing.T) {
	configYAML := []byte(`
connector:
  agentId: 3dcd9a52-949d-4531-a3b0-b14622f7dd39
dsConnection:
  hostName: localhost
gwConnections:
  -
    enabled: true
    hostName: localhost
    userName: RESTAPIACCESS
    password: _v1_fc0546f02af1c34298d207468a78bc38cda6bd480d3357c8220580883747505d7971c3c43610cea1bc1df9e3292cb935
`)

	os.Setenv(string(SecKeyEnv), "SECRET")
	defer os.Unsetenv(string(SecKeyEnv))

	var cfg Config
	assert.NoError(t, yaml.Unmarshal(configYAML, &cfg))
	assert.Equal(t, "P@SSW0RD", cfg.GWConnections[0].Password)

	res, err := yaml.Marshal(cfg)
	assert.NoError(t, err)
	assert.Contains(t, string(res), "password: _v1_")
	assert.NotContains(t, string(res), "password: _v1_fc0546f02")
	// t.Logf("$$\n%v", string(data))
}
