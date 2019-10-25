package config

import (
	"log"
	"os"
	"path"

	"github.com/kelseyhightower/envconfig"
	"gopkg.in/yaml.v3"
)

// ConfigEnv defines environment variable for config file path
const (
	ConfigEnv       = "TNG_CONFIG"
	ConfigName      = "config.yml"
	EnvConfigPrefix = "TNG"
)

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
	ValidateToken           GroundworkAction `yaml:"validateToken"`
}

// GroundworkConfig defines Groundwork Connection configuration
type GroundworkConfig struct {
	Host     string
	Account  string
	Password string
	Token    string
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
	StartTransport  bool   `yaml:"startTransport"`
}

// Config defines TNG Agent configuration
type Config struct {
	AgentConfig       AgentConfig       `yaml:"agentConfig"`
	GroundworkConfig  GroundworkConfig  `yaml:"groundworkConfig"`
	GroundworkActions GroundworkActions `yaml:"groundworkActions"`
}

// GetConfig returns configuration
func GetConfig() *Config {
	var cfg Config

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
		err = yaml.NewDecoder(configFile).Decode(&cfg)
		if err != nil {
			log.Println(err)
		}
	}

	err = envconfig.Process(EnvConfigPrefix, &cfg)
	if err != nil {
		log.Println(err)
	}
	return &cfg
}
