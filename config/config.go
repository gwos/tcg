package config

import (
	"encoding/json"
	"fmt"
	"github.com/gwos/tng/log"
	"github.com/kelseyhightower/envconfig"
	"gopkg.in/yaml.v3"
	"io/ioutil"
	"math"
	"net/url"
	"os"
	"path"
	"strings"
	"sync"
)

var once sync.Once
var cfg *Config

// ConfigStringConstant defines string constant type
type ConfigStringConstant string

// ConfigEnv defines environment variable for config file path, overrides the ConfigName
// ConfigName defines default filename for look in work directory if ConfigEnv is empty
// EnvConfigPrefix defines name prefix for environment variables
//   for example: TNG_CONNECTOR_NATSSTORETYPE
const (
	ConfigEnv       ConfigStringConstant = "TNG_CONFIG"
	ConfigName                           = "tng_config.yaml"
	EnvConfigPrefix                      = "TNG"
)

// LogLevel defines levels for logrus
type LogLevel int

// Enum levels
const (
	Error LogLevel = iota
	Warn
	Info
	Debug
)

func (l LogLevel) String() string {
	return [...]string{"Error", "Warn", "Info", "Debug"}[l]
}

// Connector defines TNG Connector configuration
// see GetConfig() for defaults
type Connector struct {
	AgentID string `yaml:"agentId"`
	AppName string `yaml:"appName"`
	AppType string `yaml:"appType"`
	// ControllerAddr accepts value for combined "host:port"
	// used as `http.Server{Addr}`
	ControllerAddr     string `yaml:"controllerAddr"`
	ControllerCertFile string `yaml:"controllerCertFile"`
	ControllerKeyFile  string `yaml:"controllerKeyFile"`
	// ControllerPin accepts value from environment
	// provides local access for debug
	ControllerPin string `yaml:"-"`
	// NatsAckWait accepts number of seconds
	// should be greater then the GWClient request duration
	NatsAckWait int64 `yaml:"natsAckWait"`
	// NatsMaxInflight accepts number of unacknowledged messages
	// that a publisher may have in-flight at any given time.
	// When this maximum is reached, further async publish calls will block
	// until the number of unacknowledged messages falls below the specified limit
	NatsMaxInflight  int    `yaml:"natsMaxInflight"`
	NatsFilestoreDir string `yaml:"natsFilestoreDir"`
	// NatsStoreType accepts "FILE"|"MEMORY"
	NatsStoreType string `yaml:"natsStoreType"`
	// NatsHost accepts value for combined "host:port"
	// used as `strings.Split(natsHost, ":")`
	NatsHost string `yaml:"natsHost"`
	// LogFile accepts file path to log in addition to stdout
	LogFile  string   `yaml:"logFile"`
	LogLevel LogLevel `yaml:"logLevel"`
	Enabled  bool     `yaml:"enabled"`
}

// ConnectorDTO defines TNG Connector configuration
type ConnectorDTO struct {
	AgentID       string        `json:"agentId"`
	AppName       string        `json:"appName"`
	AppType       string        `json:"appType"`
	TngURL        string        `json:"tngUrl"`
	LogLevel      LogLevel      `json:"logLevel"`
	Enabled       bool          `json:"enabled"`
	DSConnection  DSConnection  `json:"dalekservicesConnection"`
	GWConnections GWConnections `json:"groundworkConnections"`
	// TODO: extend LoadConnectorDTO to handle more fields
	// MonitorConnection MonitorConnectionDto
	// MetricsProfile    MetricsProfileDto
}

// GWConnection defines Groundwork Connection configuration
type GWConnection struct {
	// HostName accepts value for combined "host:port"
	// used as `url.URL{HostName}`
	HostName            string `yaml:"hostName"`
	UserName            string `yaml:"userName"`
	Password            string `yaml:"password"`
	Enabled             bool   `yaml:"enabled"`
	IsChild             bool   `yaml:"isChild"`
	DisplayName         string `yaml:"displayName"`
	MergeHosts          bool   `yaml:"mergeHosts"`
	LocalConnection     bool   `yaml:"localConnection"`
	DeferOwnership      string `yaml:"deferOwnership"`
	PrefixResourceNames bool   `yaml:"prefixResourceNames"`
	ResourceNamePrefix  string `yaml:"resourceNamePrefix"`
	SendAllInventory    bool   `yaml:"sendAllInventory"`
}

// Decode implements envconfig.Decoder interface
// merges incoming value with existed structure
func (con *GWConnection) Decode(value string) error {
	var overrides GWConnection
	if err := yaml.Unmarshal([]byte(value), &overrides); err != nil {
		return err
	}
	if overrides.HostName != "" {
		con.HostName = overrides.HostName
	}
	if overrides.UserName != "" {
		con.UserName = overrides.UserName
	}
	if overrides.Password != "" {
		con.Password = overrides.Password
	}
	if overrides.DisplayName != "" {
		con.DisplayName = overrides.DisplayName
	}
	if overrides.DeferOwnership != "" {
		con.DeferOwnership = overrides.DeferOwnership
	}
	if overrides.ResourceNamePrefix != "" {
		con.ResourceNamePrefix = overrides.ResourceNamePrefix
	}
	return nil
}

// GWConnections defines a set of configurations
type GWConnections []*GWConnection

// Decode implements envconfig.Decoder interface
// merges incoming value with existing structure
func (cons *GWConnections) Decode(value string) error {
	var overrides GWConnections
	if err := yaml.Unmarshal([]byte(value), &overrides); err != nil {
		return err
	}
	if len(overrides) > len(*cons) {
		buf := GWConnections(make([]*GWConnection, len(overrides)))
		copy(buf, overrides)
		copy(buf, *cons)
		*cons = *(&buf)
	}
	for i, v := range overrides {
		if v.HostName != "" {
			(*cons)[i].HostName = v.HostName
		}
		if v.UserName != "" {
			(*cons)[i].UserName = v.UserName
		}
		if v.Password != "" {
			(*cons)[i].Password = v.Password
		}
	}
	return nil
}

// DSConnection defines DalekServices Connection configuration
type DSConnection struct {
	// HostName accepts value for combined "host:port"
	// used as `url.URL{HostName}`
	HostName string `yaml:"hostName"`
}

// Decode implements envconfig.Decoder interface
// merges incoming value with existed structure
func (con *DSConnection) Decode(value string) error {
	var overrides GWConnection
	if err := yaml.Unmarshal([]byte(value), &overrides); err != nil {
		return err
	}
	if overrides.HostName != "" {
		con.HostName = overrides.HostName
	}
	return nil
}

// Config defines TNG Agent configuration
type Config struct {
	Connector     *Connector    `yaml:"connector"`
	DSConnection  *DSConnection `yaml:"dsConnection"`
	GWConnections GWConnections `yaml:"gwConnections"`
}

// GetConfig implements Singleton pattern
func GetConfig() *Config {
	once.Do(func() {
		// set defaults
		cfg = &Config{
			Connector: &Connector{
				ControllerAddr:   ":8099",
				LogLevel:         1,
				NatsAckWait:      30,
				NatsMaxInflight:  math.MaxInt32,
				NatsFilestoreDir: "natsstore",
				NatsStoreType:    "FILE",
				NatsHost:         "127.0.0.1:4222",
			},
			DSConnection: &DSConnection{},
		}

		configPath := os.Getenv(string(ConfigEnv))
		if configPath == "" {
			wd, err := os.Getwd()
			if err != nil {
				log.Warn(err)
				wd = ""
			}
			configPath = path.Join(wd, ConfigName)
		}

		if data, err := ioutil.ReadFile(configPath); err != nil {
			log.Warn(err)
		} else {
			if err := yaml.Unmarshal(data, cfg); err != nil {
				log.Warn(err)
			}
		}

		if err := envconfig.Process(EnvConfigPrefix, cfg); err != nil {
			log.Warn(err)
		}

		log.Config(cfg.Connector.LogFile, int(cfg.Connector.LogLevel))
	})
	return cfg
}

// LoadConnectorDTO loads ConnectorDTO into Config
func (cfg *Config) LoadConnectorDTO(data []byte) (*ConnectorDTO, error) {
	var dto ConnectorDTO
	var tempURLString string

	if err := json.Unmarshal(data, &dto); err != nil {
		return nil, err
	}

	tempURLString = dto.TngURL
	if !strings.HasPrefix(tempURLString, "http://") && !strings.HasPrefix(tempURLString, "https://") {
		tempURLString = "https://" + tempURLString
	}
	if tngURL, err := url.Parse(tempURLString); err == nil {
		// TODO: Improve addr setting
		cfg.Connector.ControllerAddr = fmt.Sprintf("0.0.0.0:%s", tngURL.Port())
	} else {
		return nil, err
	}

	cfg.Connector.AgentID = dto.AgentID
	cfg.Connector.AppName = dto.AppName
	cfg.Connector.AppType = dto.AppType
	cfg.Connector.LogLevel = dto.LogLevel
	cfg.Connector.Enabled = dto.Enabled
	cfg.GWConnections = dto.GWConnections
	if len(dto.DSConnection.HostName) != 0 {
		cfg.DSConnection.HostName = dto.DSConnection.HostName
	}

	log.Config(cfg.Connector.LogFile, int(cfg.Connector.LogLevel))

	if output, err := yaml.Marshal(cfg); err != nil {
		log.Warn(err)
	} else {
		configPath := os.Getenv(string(ConfigEnv))
		if configPath == "" {
			wd, err := os.Getwd()
			if err != nil {
				log.Warn(err)
				wd = ""
			}
			configPath = path.Join(wd, ConfigName)
		}
		if err := ioutil.WriteFile(configPath, output, 0644); err != nil {
			log.Warn(err)
		}
	}

	return &dto, nil
}
