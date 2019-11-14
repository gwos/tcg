package config

import (
	"github.com/kelseyhightower/envconfig"
	"gopkg.in/yaml.v3"
	"log"
	"os"
	"path"
	"sync"
)

var once sync.Once
var cfg *Config

// ConfigEnv defines environment variable for config file path
const (
	ConfigEnv       = "TNG_CONFIG"
	ConfigName      = "config.yml"
	EnvConfigPrefix = "TNG"
)

// GWConfig defines Groundwork Connection configuration
type GWConfig struct {
	Host     string
	Account  string
	Password string
	AppName  string `yaml:"appName"`
}

// AgentConfig defines TNG Transit Agent configuration
type AgentConfig struct {
	// ControllerAddr accepts value for `http.Server{Addr}`
	// TCP address to listen on, ":http" if empty
	ControllerAddr     string `yaml:"controllerAddr"`
	ControllerCertFile string `yaml:"controllerCertFile"`
	ControllerKeyFile  string `yaml:"controllerKeyFile"`
	// NatsAckWait accepts number of seconds
	// should be greater then the GWClient request duration
	NatsAckWait      int64  `yaml:"natsAckWait"`
	NatsFilestoreDir string `yaml:"natsFilestoreDir"`
	// NatsStoreType accepts "FILE"|"MEMORY"
	NatsStoreType string `yaml:"natsStoreType"`
	// NatsURL accepts value in form "nats://localhost:4222"
	NatsURL         string `yaml:"natsURL"`
	StartController bool   `yaml:"startController"`
	StartNats       bool   `yaml:"startNats"`
	// StartTransport defines that NATS starts with Transport
	StartTransport bool `yaml:"startTransport"`
}

// Config defines TNG Agent configuration
type Config struct {
	AgentConfig AgentConfig `yaml:"agentConfig"`
	GWConfig    GWConfig    `yaml:"gwConfig"`
}

// GetConfig implements Singleton pattern
func GetConfig() *Config {
	once.Do(func() {
		cfg = &Config{}

		configPath := os.Getenv(ConfigEnv)
		if configPath == "" {
			wd, err := os.Getwd()
			if err != nil {
				log.Println(err)
			}
			configPath = path.Join(wd, ConfigName)
		}

		configFile, err := os.Open(configPath)
		if err == nil {
			err = yaml.NewDecoder(configFile).Decode(cfg)
			if err != nil {
				log.Println(err)
			}
		}

		err = envconfig.Process(EnvConfigPrefix, cfg)
		if err != nil {
			log.Println(err)
		}
	})
	return cfg
}
