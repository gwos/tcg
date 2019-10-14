package services

import (
	"github.com/gwos/tng/controller"
	"github.com/gwos/tng/nats"
	"github.com/gwos/tng/transit"
	"gopkg.in/yaml.v2"
	"log"
	"os"
	"path"
	"time"
)

const (
	SendResourceWithMetricsSubject = "send-resource-with-metrics"
	SynchronizeInventorySubject    = "synchronize-inventory"
)

func init() {
	workDir, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}

	configFile, err := os.Open(path.Join(workDir, "Config.yml"))
	if err != nil {
		log.Fatal(err)
	}

	combinedConfig := struct {
		transit.Actions          `yaml:"actions"`
		transit.GroundworkConfig `yaml:"Config"`
	}{}

	decoder := yaml.NewDecoder(configFile)
	err = decoder.Decode(&combinedConfig)
	if err != nil {
		log.Fatal(err)
	}

	transit.ServiceActions = combinedConfig.Actions
	transit.Config.Config = combinedConfig.GroundworkConfig

	err = transit.Config.Connect()
	if err != nil {
		log.Fatal(err)
	}

	dispatcherMap := nats.DispatcherMap{
		SendResourceWithMetricsSubject: func(b []byte) error {
			_, err := transit.Config.SendResourcesWithMetrics(b)
			if err == nil {
				controller.AgentStatistics.LastMetricsRun = transit.MillisecondTimestamp{Time: time.Now()}
			}
			return err
		},
		SynchronizeInventorySubject: func(b []byte) error {
			_, err := transit.Config.SynchronizeInventory(b)
			if err != nil {
				controller.AgentStatistics.LastInventoryRun = transit.MillisecondTimestamp{Time: time.Now()}
			}
			return err
		},
	}

	_, err = nats.StartServer()
	if err != nil {
		log.Fatal(err)
	}

	err = nats.StartDispatcher(&dispatcherMap)
	if err != nil {
		log.Fatal(err)
	}

	err = controller.StartServer(transit.Config.Config.SSL, transit.Config.Config.Port)
	if err != nil {
		log.Fatal(err)
	}
}

type Service struct{}

func (transitService Service) SendResourceWithMetrics(resourcesWithMetricsJson []byte) error {
	return nats.Publish(SendResourceWithMetricsSubject, resourcesWithMetricsJson)
}

func (transitService Service) SynchronizeInventory(inventoryJson []byte) error {
	return nats.Publish(SynchronizeInventorySubject, inventoryJson)
}

func (transitService Service) ListMetrics() (*[]transit.MetricDescriptor, error) {
	return transit.Config.ListMetrics()
}