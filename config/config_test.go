package config

import (
	"io/ioutil"
	"os"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"
)

func TestGetConfig(t *testing.T) {
	once = sync.Once{}
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

	tmpFile, err := ioutil.TempFile("", "config")
	assert.NoError(t, err)
	defer os.Remove(tmpFile.Name())
	_, err = tmpFile.Write(configYAML)
	assert.NoError(t, err)
	err = tmpFile.Close()
	assert.NoError(t, err)

	_ = os.Setenv(string(ConfigEnv), tmpFile.Name())
	_ = os.Setenv("TCG_CONNECTOR_NATSSTORETYPE", "MEMORY")
	_ = os.Setenv("TCG_DSCONNECTION", "{\"hostName\":\"localhost:3001\"}")
	_ = os.Setenv("TCG_GWCONNECTIONS", "[{\"password\":\"SEC RET\"},{\"hostName\":\"localhost:3001\"}]")
	defer os.Unsetenv(string(ConfigEnv))
	defer os.Unsetenv("TCG_CONNECTOR_NATSSTORETYPE")
	defer os.Unsetenv("TCG_DSCONNECTION")
	defer os.Unsetenv("TCG_GWCONNECTIONS")

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
		},
		DSConnection: &DSConnection{"localhost:3001"},
		GWConnections: GWConnections{
			&GWConnection{Enabled: true, LocalConnection: false, HostName: "localhost:80", UserName: "RESTAPIACCESS", Password: "SEC RET"},
			&GWConnection{HostName: "localhost:3001"},
		},
	}

	cfg := GetConfig()
	assert.Equal(t, *expected.Connector, *cfg.Connector)
	assert.Equal(t, *expected.DSConnection, *cfg.DSConnection)
	assert.Equal(t, expected.GWConnections, cfg.GWConnections)
}

func TestLoadConnectorDTO(t *testing.T) {
	once = sync.Once{}
	configYAML := []byte(`
connector:
  agentId:
  appName: "test-app"
  appType: "test"
  controllerAddr: ":8011"
dsConnection:
  hostName: "localhost"
`)

	tmpFile, err := ioutil.TempFile("", "config")
	assert.NoError(t, err)
	defer os.Remove(tmpFile.Name())
	_, err = tmpFile.Write(configYAML)
	assert.NoError(t, err)

	_ = os.Setenv(string(ConfigEnv), tmpFile.Name())
	_ = os.Setenv("TCG_CONNECTOR_CONTROLLERADDR", ":8022")
	defer os.Unsetenv(string(ConfigEnv))
	defer os.Unsetenv("TCG_CONNECTOR_CONTROLLERADDR")

	expected := &Config{
		Connector: &Connector{
			AgentID:          "",
			AppName:          "test-app",
			AppType:          "test",
			ControllerAddr:   ":8022",
			LogConsPeriod:    0,
			LogLevel:         1,
			NatsAckWait:      30,
			NatsMaxInflight:  2147483647,
			NatsFilestoreDir: "natsstore",
			NatsStoreType:    "FILE",
		},
		DSConnection: &DSConnection{"localhost"},
	}

	cfg := GetConfig()
	assert.Equal(t, *expected.Connector, *cfg.Connector)
	assert.Equal(t, *expected.DSConnection, *cfg.DSConnection)

	dto := []byte(`
{
  "agentId": "99998888-7777-6666-a3b0-b14622f7dd39",
  "appName": "test-app-XX",
  "appType": "test-XX",
  "logConsPeriod": 30,
  "logLevel": 2,
  "tcgUrl": "http://tcg-host/",
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
			ControllerAddr:   ":8022",
			LogConsPeriod:    30,
			LogLevel:         2,
			NatsAckWait:      30,
			NatsMaxInflight:  2147483647,
			NatsFilestoreDir: "natsstore",
			NatsStoreType:    "FILE",
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

	assert.Equal(t, *expected.Connector, *cfg.Connector)
	assert.Equal(t, *expected.DSConnection, *cfg.DSConnection)
	assert.Equal(t, expected.GWConnections, cfg.GWConnections)

	data, err := ioutil.ReadFile(tmpFile.Name())
	assert.Contains(t, string(data), "99998888-7777-6666-a3b0-b14622f7dd39")
	assert.Contains(t, string(data), "controllerAddr: :8011")
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

	_ = os.Setenv(SecKeyEnv, "SECRET")
	defer os.Unsetenv(SecKeyEnv)

	var cfg Config
	assert.NoError(t, yaml.Unmarshal(configYAML, &cfg))
	assert.Equal(t, "P@SSW0RD", cfg.GWConnections[0].Password)

	res, err := yaml.Marshal(cfg)
	assert.NoError(t, err)
	assert.Contains(t, string(res), "password: _v1_")
	assert.NotContains(t, string(res), "password: _v1_fc0546f02")
	// t.Logf("$$\n%v", string(data))
}
