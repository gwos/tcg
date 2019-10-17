package services

import (
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

var service Service

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

	if transit.Config.StartNATS {
		err = service.StartNATS()
		if err != nil {
			log.Fatal(err)
		}
	}

	if transit.Config.StartTransport {
		err = service.StartTransport()
		if err != nil {
			log.Fatal(err)
		}
	}
}

// Service implements TNG Agent operations
type Service struct{}

// SendResourceWithMetrics provides fire and forget method
func (transitService Service) SendResourceWithMetrics(request []byte) error {
	return nats.Publish(SendResourceWithMetricsSubject, request)
}

// SynchronizeInventory provides fire and forget method
func (transitService Service) SynchronizeInventory(request []byte) error {
	return nats.Publish(SynchronizeInventorySubject, request)
}

// StartNATS starts internal NATS
func (transitService Service) StartNATS() error {
	return nats.StartServer()
}

// StopNATS stops internal NATS
func (transitService Service) StopNATS() {
	nats.StopServer()
}

// StartTransport starts dispatching NATS messages to Groundwork
func (transitService Service) StartTransport() error {
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

	return nats.StartDispatcher(&dispatcherMap)
}

// StopTransport stops dispatching NATS messages to Groundwork
func (transitService Service) StopTransport() error {
	return nats.StopDispatcher()
}
