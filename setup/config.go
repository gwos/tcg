package setup

import (
	"github.com/gwos/tng/log"
	"github.com/kelseyhightower/envconfig"
	"gopkg.in/yaml.v3"
	"math"
	"os"
	"path"
	"sync"
)

var once sync.Once
var cfg *Config

// ConfigStringConstant defines string constant type
type ConfigStringConstant string

// ConfigEnv defines environment variable for setup file path, overrides the ConfigName
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
	NatsHost string   `yaml:"natsHost"`
	LogLevel LogLevel `yaml:"logLevel"`
}

// GWConnection defines Groundwork Connection configuration
type GWConnection struct {
	// HostName accepts value for combined "host:port"
	// used as `url.URL{HostName}`
	HostName string `yaml:"hostName"`
	UserName string `yaml:"userName"`
	Password string `yaml:"password"`
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
type DSConnection GWConnection

// Decode implements envconfig.Decoder interface
// merges incoming value with existed structure
func (con *DSConnection) Decode(value string) error {
	return (*GWConnection)(con).Decode(value)
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
				ControllerAddr:   ":8081",
				LogLevel:         1,
				NatsAckWait:      30,
				NatsMaxInflight:  math.MaxInt32,
				NatsFilestoreDir: "natsstore",
				NatsStoreType:    "FILE",
				NatsHost:         ":4222",
			},
			DSConnection: &DSConnection{},
		}

		configPath := os.Getenv(string(ConfigEnv))
		if configPath == "" {
			wd, err := os.Getwd()
			if err != nil {
				log.Warn(err)
			}
			configPath = path.Join(wd, ConfigName)
		}

		configFile, err := os.Open(configPath)
		if err == nil {
			err = yaml.NewDecoder(configFile).Decode(cfg)
			if err != nil {
				log.Warn(err)
			}
		}

		err = envconfig.Process(EnvConfigPrefix, cfg)
		if err != nil {
			log.Warn(err)
		}

		log.Config(int(cfg.Connector.LogLevel))
	})
	return cfg
}
