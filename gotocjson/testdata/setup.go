package setup

import (
	"github.com/gwos/tng/log"
	"github.com/kelseyhightower/envconfig"
	"gopkg.in/yaml.v3"
	"os"
	"path"
	"sync"
)

var once sync.Once
var cfg *Config

// We need this type so constants like ConfigEnv are not declared as "string" type,
// which would interfere with how the C code enumeration for the associated values
// gets generated.  That said, this will probably mean that any use of these constants
// will need to be explicitly converted to "string" type in the Go code.
type ConfigStringConstant string

// ConfigEnv defines environment variable for setup file path
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

// AgentConfig defines TNG Transit Agent configuration
type AgentConfig struct {
	// ControllerAddr accepts value for combined "host:port"
	// used as `http.Server{Addr}`
	ControllerAddr     string `yaml:"controllerAddr"`
	ControllerCertFile string `yaml:"controllerCertFile"`
	ControllerKeyFile  string `yaml:"controllerKeyFile"`
	// NatsAckWait accepts number of seconds
	// should be greater then the GWClient request duration
	NatsAckWait   int64 `yaml:"natsAckWait"`
	NatsFilestoreDir   string `yaml:"natsFilestoreDir"`
	// NatsStoreType accepts "FILE"|"MEMORY"
	NatsStoreType   string `yaml:"natsStoreType"`
	// NatsHost accepts value for combined "host:port"
	// used as `strings.Split(natsHost, ":")`
	NatsHost   string `yaml:"natsHost"`
	LogLevel   LogLevel `yaml:"logLevel"`
}

// GroundworkAction defines configurable options for an action
type GroundworkAction struct {
	Entrypoint string `yaml:"entrypoint"`
}

// GroundworkActions configures Groundwork actions
type GroundworkActions struct {
	Connect                 GroundworkAction `yaml:"connect"`
	Disconnect              GroundworkAction `yaml:"disconnect"`
	SynchronizeInventory    GroundworkAction `yaml:"synchronizeInventory"`
	SendResourceWithMetrics GroundworkAction `yaml:"sendResourceWithMetrics"`
}

// GWConfig defines Groundwork Connection configuration
type GWConfig struct {
	// Host accepts value for combined "host:port"
	// used as `url.URL{Host}`
	Host     string
	Account  string
	Password string
	AppName  string `yaml:"appName"`
}

// GWConfigs defines a set of configurations
type GWConfigs []*GWConfig

// Decode implements envconfig.Decoder interface
func (gwConfigs *GWConfigs) Decode(value string) error {
	var overrides GWConfigs
	if err := yaml.Unmarshal([]byte(value), &overrides); err != nil {
		return err
	}
	if len(overrides) > len(*gwConfigs) {
		buf := GWConfigs(make([]*GWConfig, len(overrides)))
		copy(buf, overrides)
		copy(buf, *gwConfigs)
		*gwConfigs = *(&buf)
	}
	for i, v := range overrides {
		if v.Host != "" {
			(*gwConfigs)[i].Host = v.Host
		}
		if v.Account != "" {
			(*gwConfigs)[i].Account = v.Account
		}
		if v.Password != "" {
			(*gwConfigs)[i].Password = v.Password
		}
		if v.AppName != "" {
			(*gwConfigs)[i].AppName = v.AppName
		}
	}
	return nil
}

// Config defines TNG Agent configuration
type Config struct {
	AgentConfig *AgentConfig `yaml:"agentConfig"`
	GWConfigs GWConfigs `yaml:"gwConfigs"`
	GroundworkActions GroundworkActions `yaml:"groundworkActions"`
}

// GetConfig implements Singleton pattern
func GetConfig() *Config {
	once.Do(func() {
		cfg = &Config{}

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

		log.Config(int(cfg.AgentConfig.LogLevel))
	})
	return cfg
}
