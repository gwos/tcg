package services

import (
	"time"

	"github.com/gwos/tng/nats"
	"github.com/gwos/tng/transit"
)

// Define NATS subjects
const (
	SendResourceWithMetricsSubject = "send-resource-with-metrics"
	SynchronizeInventorySubject    = "synchronize-inventory"
)

// Service implements Services interface
type Service struct {
	transit.Transit
	AgentStats AgentStats
}

// SendResourceWithMetrics implements Services.SendResourceWithMetrics interface
func (service Service) SendResourceWithMetrics(request []byte) error {
	return nats.Publish(SendResourceWithMetricsSubject, request)
}

// SynchronizeInventory implements Services.SynchronizeInventory interface
func (service Service) SynchronizeInventory(request []byte) error {
	return nats.Publish(SynchronizeInventorySubject, request)
}

// StartNATS implements Services.StartNATS interface
func (service Service) StartNATS() error {
	return nats.StartServer()
}

// StopNATS implements Services.StopNATS interface
func (service Service) StopNATS() {
	nats.StopServer()
}

// StartTransport implements Services.StartTransport interface
func (service Service) StartTransport() error {
	dispatcherMap := nats.DispatcherMap{
		SendResourceWithMetricsSubject: func(b []byte) error {
			_, err := service.Transit.SendResourcesWithMetrics(b)
			if err == nil {
				service.AgentStats.LastMetricsRun = transit.MillisecondTimestamp{Time: time.Now()}
				service.AgentStats.BytesSent += len(b)
				service.AgentStats.MessagesSent++
			} else {
				service.AgentStats.LastError = err.Error()
			}
			return err
		},
		SynchronizeInventorySubject: func(b []byte) error {
			_, err := service.Transit.SynchronizeInventory(b)
			if err != nil {
				service.AgentStats.LastInventoryRun = transit.MillisecondTimestamp{Time: time.Now()}
				service.AgentStats.BytesSent += len(b)
				service.AgentStats.MessagesSent++
			} else {
				service.AgentStats.LastError = err.Error()
			}
			return err
		},
	}

	return nats.StartDispatcher(&dispatcherMap)
}

// StopTransport implements Services.StopTransport interface
func (service Service) StopTransport() error {
	return nats.StopDispatcher()
}
