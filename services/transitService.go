package services

import (
	"github.com/gwos/tng/config"
	"github.com/gwos/tng/milliseconds"
	"github.com/gwos/tng/nats"
	"github.com/gwos/tng/transit"
	"sync"
	"time"
)

// TransitService implements AgentServices, TransitServices interfaces
type TransitService struct {
	transit.Transit
	agentStats  *AgentStats
	agentStatus *AgentStatus
}

var onceTransitService sync.Once
var transitService *TransitService

// GetTransitService implements Singleton pattern
func GetTransitService() *TransitService {
	onceTransitService.Do(func() {
		transitService = &TransitService{
			transit.Transit{Config: config.GetConfig()},
			&AgentStats{},
			&AgentStatus{
				Controller: Pending,
				NATS:       Pending,
				Transport:  Pending,
			},
		}
	})
	return transitService
}

// SendResourceWithMetrics implements TransitServices.SendResourceWithMetrics interface
func (service *TransitService) SendResourceWithMetrics(request []byte) error {
	return nats.Publish(SubjSendResourceWithMetrics, request)
}

// SynchronizeInventory implements TransitServices.SynchronizeInventory interface
func (service *TransitService) SynchronizeInventory(request []byte) error {
	return nats.Publish(SubjSynchronizeInventory, request)
}

// StartController implements AgentServices.StartController interface
func (service *TransitService) StartController() error {
	// NOTE: the service.agentStatus.Controller will be updated by controller itself
	return GetController().StartServer(
		service.AgentConfig.ControllerAddr,
		service.AgentConfig.ControllerCertFile,
		service.AgentConfig.ControllerKeyFile,
	)
}

// StopController implements AgentServices.StopController interface
func (service *TransitService) StopController() error {
	// NOTE: the service.agentStatus.Controller will be updated by controller itself
	return GetController().StopServer()
}

// StartNATS implements AgentServices.StartNATS interface
func (service *TransitService) StartNATS() error {
	err := nats.StartServer(service.AgentConfig.NATSStoreType, service.AgentConfig.NATSFilestoreDir)
	if err == nil {
		service.agentStatus.Lock()
		service.agentStatus.NATS = Running
		service.agentStatus.Unlock()
	}
	return err
}

// StopNATS implements AgentServices.StopNATS interface
func (service *TransitService) StopNATS() error {
	nats.StopServer()
	service.agentStatus.Lock()
	service.agentStatus.NATS = Stopped
	service.agentStatus.Unlock()
	return nil
}

// StartTransport implements AgentServices.StartTransport interface
func (service *TransitService) StartTransport() error {
	dispatcherMap := nats.DispatcherMap{
		SubjSendResourceWithMetrics: func(b []byte) error {
			_, err := service.Transit.SendResourcesWithMetrics(b)
			service.agentStats.Lock()
			if err == nil {
				service.agentStats.LastMetricsRun = milliseconds.MillisecondTimestamp{Time: time.Now()}
				service.agentStats.BytesSent += len(b)
				service.agentStats.MessagesSent++
			} else {
				service.agentStats.LastError = err.Error()
			}
			service.agentStats.Unlock()
			return err
		},
		SubjSynchronizeInventory: func(b []byte) error {
			_, err := service.Transit.SynchronizeInventory(b)
			service.agentStats.Lock()
			if err == nil {
				service.agentStats.LastInventoryRun = milliseconds.MillisecondTimestamp{Time: time.Now()}
				service.agentStats.BytesSent += len(b)
				service.agentStats.MessagesSent++
			} else {
				service.agentStats.LastError = err.Error()
			}
			service.agentStats.Unlock()
			return err
		},
	}

	err := nats.StartDispatcher(&dispatcherMap)
	if err == nil {
		service.agentStatus.Lock()
		service.agentStatus.Transport = Running
		service.agentStatus.Unlock()
	}
	return err
}

// StopTransport implements AgentServices.StopTransport interface
func (service *TransitService) StopTransport() error {
	err := nats.StopDispatcher()
	if err == nil {
		service.agentStatus.Lock()
		service.agentStatus.Transport = Stopped
		service.agentStatus.Unlock()
	}
	return err
}

// Stats implements AgentServices.Stats interface
func (service *TransitService) Stats() *AgentStats {
	return service.agentStats
}

// Status implements AgentServices.Status interface
func (service *TransitService) Status() *AgentStatus {
	return service.agentStatus
}
