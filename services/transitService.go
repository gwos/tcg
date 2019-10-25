package services

import (
	"github.com/gwos/tng/milliseconds"
	"sync"
	"time"

	"github.com/gwos/tng/config"
	"github.com/gwos/tng/nats"
	"github.com/gwos/tng/transit"
)

// Define NATS subjects
const (
	SendResourceWithMetricsSubject = "send-resource-with-metrics"
	SynchronizeInventorySubject    = "synchronize-inventory"
)

var once sync.Once
var service *TransitService

// GetTransitService implements Singleton pattern
func GetTransitService() *TransitService {
	once.Do(func() {
		service = &TransitService{
			transit.Transit{Config: config.GetConfig()},
			&AgentStats{},
		}
	})
	return service
}

// TransitService implements Services interface
type TransitService struct {
	transit.Transit
	AgentStats *AgentStats
}

// SendResourceWithMetrics implements Services.SendResourceWithMetrics interface
func (service *TransitService) SendResourceWithMetrics(request []byte) error {
	return nats.Publish(SendResourceWithMetricsSubject, request)
}

// SynchronizeInventory implements Services.SynchronizeInventory interface
func (service *TransitService) SynchronizeInventory(request []byte) error {
	return nats.Publish(SynchronizeInventorySubject, request)
}

// StartNATS implements Services.StartNATS interface
func (service *TransitService) StartNATS() error {
	return nats.StartServer(service.AgentConfig.NATSStoreType, service.AgentConfig.NATSFilestoreDir)
}

// StopNATS implements Services.StopNATS interface
func (service *TransitService) StopNATS() {
	nats.StopServer()
}

// StartTransport implements Services.StartTransport interface
func (service *TransitService) StartTransport() error {
	dispatcherMap := nats.DispatcherMap{
		SendResourceWithMetricsSubject: func(b []byte) error {
			_, err := service.Transit.SendResourcesWithMetrics(b)
			if err == nil {
				service.AgentStats.Lock()
				service.AgentStats.LastMetricsRun = milliseconds.MillisecondTimestamp{Time: time.Now()}
				service.AgentStats.BytesSent += len(b)
				service.AgentStats.MessagesSent++
				service.AgentStats.Unlock()
			} else {
				service.AgentStats.Lock()
				service.AgentStats.LastError = err.Error()
				service.AgentStats.Unlock()
			}
			return err
		},
		SynchronizeInventorySubject: func(b []byte) error {
			_, err := service.Transit.SynchronizeInventory(b)
			if err == nil {
				service.AgentStats.Lock()
				service.AgentStats.LastInventoryRun = milliseconds.MillisecondTimestamp{Time: time.Now()}
				service.AgentStats.BytesSent += len(b)
				service.AgentStats.MessagesSent++
				service.AgentStats.Unlock()
			} else {
				service.AgentStats.Lock()
				service.AgentStats.LastError = err.Error()
				service.AgentStats.Unlock()
			}
			return err
		},
	}

	return nats.StartDispatcher(&dispatcherMap)
}

// StopTransport implements Services.StopTransport interface
func (service *TransitService) StopTransport() error {
	return nats.StopDispatcher()
}
