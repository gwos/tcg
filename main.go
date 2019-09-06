package main

import (
	"fmt"
	"github.com/gwos/tng/controller"
	"github.com/gwos/tng/transit"
	"github.com/kelseyhightower/envconfig"
	"gopkg.in/yaml.v2"
	"os"
)

func main() {
	var cfg transit.GroundworkConfig

	// Reading configuration from a file
	f, err := os.Open("config.yml")
	if err != nil {
		fmt.Println(err.Error())
	}

	decoder := yaml.NewDecoder(f)
	err = decoder.Decode(&cfg)
	if err != nil {
		fmt.Println(err.Error())
	}

	fmt.Println(cfg.HostName, "\n", cfg.Port, "\n", cfg.Account, "\n", cfg.Token, "\n", cfg.SSL)

	// Reading configuration from environment variables
	err = envconfig.Process("", cfg)
	if err != nil {
		fmt.Println(err.Error())
	}

	fmt.Println(cfg.HostName, "\n", cfg.Port, "\n", cfg.Account, "\n", cfg.Token, "\n", cfg.SSL)

	controller.StartServer()
}


