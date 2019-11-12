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

// GroundworkConfig defines Groundwork Connection configuration
type GroundworkConfig struct {
	Host     string
	Account  string
	Password string
	AppName  string `yaml:"appName"`
}

// AgentConfig defines TNG Transit Agent configuration
type AgentConfig struct {
	// ControllerAddr accepts value for `http.Server{Addr}`: TCP address to listen on, ":http" if empty
	ControllerAddr     string `yaml:"controllerAddr"`
	ControllerCertFile string `yaml:"controllerCertFile"`
	ControllerKeyFile  string `yaml:"controllerKeyFile"`
	NATSFilestoreDir   string `yaml:"natsFilestoreDir"`
	// NATSStoreType accepts "FILE"|"MEMORY"
	NATSStoreType   string `yaml:"natsStoreType"`
	StartController bool   `yaml:"startController"`
	StartNATS       bool   `yaml:"startNATS"`
	// StartTransport defines that NATS starts with Transport
	StartTransport bool `yaml:"startTransport"`
}

// Config defines TNG Agent configuration
type Config struct {
	AgentConfig       AgentConfig       `yaml:"agentConfig"`
	GroundworkConfig  GroundworkConfig  `yaml:"groundworkConfig"`
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
