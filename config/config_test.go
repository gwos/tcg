package config

import (
	"os"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"
)

func TestApplyEnv(t *testing.T) {
	once = sync.Once{}
	configYAML := []byte(`
connector:
  agentId:  # null field
  appName: test-app
  appType: test
  enabled: true
  batchMaxBytes: 10000
dsConnection:  # empty set
#gwConnections:  # missing
`)

	tmpFile, err := os.CreateTemp("", "config")
	assert.NoError(t, err)
	defer os.Remove(tmpFile.Name())
	_, err = tmpFile.Write(configYAML)
	assert.NoError(t, err)
	err = tmpFile.Close()
	assert.NoError(t, err)

	_ = os.Setenv(ConfigEnv, tmpFile.Name())
	_ = os.Setenv("TCG_CONNECTOR_AGENTID", `AGENT-ID`)
	_ = os.Setenv("TCG_CONNECTOR_ENABLED", `false`)
	_ = os.Setenv("TCG_CONNECTOR_BATCHMAXBYTES", `111`)
	_ = os.Setenv("TCG_DSCONNECTION_HOSTNAME", "localhost:3001")
	_ = os.Setenv("TCG_GWCONNECTIONS_0_PASSWORD", "SEC RET")
	_ = os.Setenv("TCG_GWCONNECTIONS_1_PASSWORD", "SEC_RET")
	defer os.Unsetenv("TCG_CONNECTOR_AGENTID")
	defer os.Unsetenv("TCG_CONNECTOR_ENABLED")
	defer os.Unsetenv("TCG_CONNECTOR_BATCHMAXBYTES")
	defer os.Unsetenv("TCG_DSCONNECTION_HOSTNAME")
	defer os.Unsetenv("TCG_GWCONNECTIONS_0_PASSWORD")
	defer os.Unsetenv("TCG_GWCONNECTIONS_1_PASSWORD")
	defer os.Unsetenv(ConfigEnv)

	_ = GetConfig()
	assert.Equal(t, "AGENT-ID", cfg.Connector.AgentID)
	assert.Equal(t, false, cfg.Connector.Enabled)
	assert.Equal(t, 111, cfg.Connector.BatchMaxBytes)
	// assert.Equal(t, "localhost:3001", cfg.DSConnection.HostName)
	assert.Equal(t, "SEC RET", cfg.GWConnections[0].Password)
	assert.Equal(t, "SEC_RET", cfg.GWConnections[1].Password)
}

func TestGetConfig(t *testing.T) {
	once = sync.Once{}
	configYAML := []byte(`
connector:
  agentId: "3dcd9a52-949d-4531-a3b0-b14622f7dd39"
  appName: "test-app"
  appType: "test"
  controllerAddr: ":9999"
  logCondense: "30s"
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

	tmpFile, err := os.CreateTemp("", "config")
	assert.NoError(t, err)
	defer os.Remove(tmpFile.Name())
	_, err = tmpFile.Write(configYAML)
	assert.NoError(t, err)
	err = tmpFile.Close()
	assert.NoError(t, err)

	_ = os.Setenv(ConfigEnv, tmpFile.Name())
	_ = os.Setenv("TCG_CONNECTOR_NATSSTOREMAXAGE", "1h")
	_ = os.Setenv("TCG_CONNECTOR_NATSSTORETYPE", "MEMORY")
	_ = os.Setenv("TCG_DSCONNECTION_HOSTNAME", "localhost:3001")
	_ = os.Setenv("TCG_GWCONNECTIONS_0_PASSWORD", "SEC RET")
	_ = os.Setenv("TCG_GWCONNECTIONS_1_HOSTNAME", "localhost:3001")
	defer os.Unsetenv(ConfigEnv)
	defer os.Unsetenv("TCG_CONNECTOR_NATSSTOREMAXAGE")
	defer os.Unsetenv("TCG_CONNECTOR_NATSSTORETYPE")
	defer os.Unsetenv("TCG_DSCONNECTION_HOSTNAME")
	defer os.Unsetenv("TCG_GWCONNECTIONS_0_PASSWORD")
	defer os.Unsetenv("TCG_GWCONNECTIONS_1_HOSTNAME")

	expected := defaults()
	expected.Connector.AgentID = "3dcd9a52-949d-4531-a3b0-b14622f7dd39"
	expected.Connector.AppName = "test-app"
	expected.Connector.AppType = "test"
	expected.Connector.ControllerAddr = ":9999"
	expected.Connector.LogCondense = 30000000000
	expected.Connector.NatsStoreType = "MEMORY"
	expected.Connector.NatsStoreMaxAge = 3600000000000
	expected.DSConnection = DSConnection{"localhost:3001"}
	expected.GWConnections = GWConnections{
		{Enabled: true, LocalConnection: false, HostName: "localhost:80", UserName: "RESTAPIACCESS", Password: "SEC RET"},
		{HostName: "localhost:3001"},
		{},
		{},
	}

	cfg := GetConfig()
	assert.Equal(t, expected.Connector, cfg.Connector)
	assert.Equal(t, expected.DSConnection, cfg.DSConnection)
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

	tmpFile, err := os.CreateTemp("", "config")
	assert.NoError(t, err)
	defer os.Remove(tmpFile.Name())
	_, err = tmpFile.Write(configYAML)
	assert.NoError(t, err)

	_ = os.Setenv(ConfigEnv, tmpFile.Name())
	_ = os.Setenv("TCG_CONNECTOR_CONTROLLERADDR", ":8022")
	defer os.Unsetenv(ConfigEnv)
	defer os.Unsetenv("TCG_CONNECTOR_CONTROLLERADDR")

	expected := defaults()
	expected.Connector.AppName = "test-app"
	expected.Connector.AppType = "test"
	expected.Connector.ControllerAddr = ":8022"
	expected.DSConnection = DSConnection{"localhost"}

	cfg := GetConfig()
	assert.Equal(t, expected.Connector, cfg.Connector)
	assert.Equal(t, expected.DSConnection, cfg.DSConnection)

	dto := []byte(`
{
  "agentId": "99998888-7777-6666-a3b0-b14622f7dd39",
  "appName": "test-app-XX",
  "appType": "test-XX",
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

	expected = defaults()
	expected.Connector.AgentID = "99998888-7777-6666-a3b0-b14622f7dd39"
	expected.Connector.AppName = "test-app-XX"
	expected.Connector.AppType = "test-XX"
	expected.Connector.ControllerAddr = ":8022"
	expected.Connector.LogLevel = 2
	expected.DSConnection = DSConnection{"gw-host-xxx"}
	expected.GWConnections = GWConnections{
		{
			ID:                  101,
			Enabled:             true,
			LocalConnection:     false,
			HostName:            "gw-host-xx",
			UserName:            "-xx-",
			Password:            "xx",
			PrefixResourceNames: true,
			ResourceNamePrefix:  "c1",
		},
	}

	assert.Equal(t, expected.Connector, cfg.Connector)
	assert.Equal(t, expected.DSConnection, cfg.DSConnection)
	assert.Equal(t, expected.GWConnections, cfg.GWConnections)

	data, err := os.ReadFile(tmpFile.Name())
	assert.NoError(t, err)
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
