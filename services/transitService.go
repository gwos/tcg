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

	configFile, err := os.Open(path.Join(workDir, "config.yml"))
	if err != nil {
		log.Fatal(err)
	}

	decoder := yaml.NewDecoder(configFile)
	err = decoder.Decode(&transit.Config)
	if err != nil {
		log.Fatal(err)
	}

	err = transit.Config.Connect()
	if err != nil {
		log.Fatal(err)
	}

	dispatcherMap := nats.DispatcherMap{
		SendResourceWithMetricsSubject: func(b []byte) error {
			_, err := transit.Config.SendResourcesWithMetrics(b)
			if err == nil {
				transit.AgentStatistics.LastMetricsRun = transit.MillisecondTimestamp{Time: time.Now()}
			}
			return err
		},
		SynchronizeInventorySubject: func(b []byte) error {
			_, err := transit.Config.SynchronizeInventory(b)
			if err != nil {
				transit.AgentStatistics.LastInventoryRun = transit.MillisecondTimestamp{Time: time.Now()}
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

	err = controller.StartServer(transit.Config.AgentConfig.SSL, transit.Config.AgentConfig.Port)
	if err != nil {
		log.Fatal(err)
	}
}

// Service implements transit.Services
type Service struct{}

// SendResourceWithMetrics implements transit.Services.SendResourceWithMetrics
func (transitService Service) SendResourceWithMetrics(request []byte) error {
	return nats.Publish(SendResourceWithMetricsSubject, request)
}

// SynchronizeInventory implements transit.Services.SynchronizeInventory
func (transitService Service) SynchronizeInventory(request []byte) error {
	return nats.Publish(SynchronizeInventorySubject, request)
}

// ListMetrics implements transit.Services.ListMetrics
func (transitService Service) ListMetrics() (*[]transit.MetricDescriptor, error) {
	return transit.Config.ListMetrics()
}
