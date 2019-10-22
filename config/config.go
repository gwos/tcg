package config

import (
	"log"
	"os"
	"path"

	"gopkg.in/yaml.v2"
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
	Identity                GroundworkAction `yaml:"identity"`
}

// GroundworkConfig defines Groundwork Connection configuration
type GroundworkConfig struct {
	Host     string `yaml:"host" envconfig:"GW_HOST"`
	Account  string `yaml:"account" envconfig:"GW_ACCOUNT"`
	Password string `yaml:"password" envconfig:"GW_PASSWORD"`
	Token    string
}

// AgentConfig defines TNG Transit Agent configuration
type AgentConfig struct {
	Port            int  `yaml:"port" envconfig:"AGENT_PORT"`
	SSL             bool `yaml:"ssl" envconfig:"AGENT_SSL"`
	StartController bool `yaml:"startController"`
	StartNATS       bool `yaml:"startNATS"`
	StartTransport  bool `yaml:"startTransport"`
}

// Config defines TNG Agent configuration
type Config struct {
	AgentConfig       `yaml:"agentConfig"`
	*GroundworkConfig  `yaml:"groundworkConfig"`
	GroundworkActions `yaml:"groundworkActions"`
}

// GetConfig returns configuration
func GetConfig() *Config {
	var env Config

	workDir, err := os.Getwd()
	if err != nil {
		log.Println(err)
	}

	configFile, err := os.Open(path.Join(workDir, "config.yml"))
	if err != nil {
		log.Println(err)
	}

	decoder := yaml.NewDecoder(configFile)
	err = decoder.Decode(&env)
	if err != nil {
		log.Println(err)
	}

	return &env
}
