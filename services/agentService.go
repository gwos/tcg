package services

import (
	"github.com/gwos/tng/config"
	"github.com/gwos/tng/milliseconds"
	"github.com/gwos/tng/nats"
	"github.com/gwos/tng/transit"
	"sync"
	"time"
)

// AgentService implements AgentServices interface
type AgentService struct {
	transit.Transit
	agentStats  *AgentStats
	agentStatus *AgentStatus
}

var onceAgentService sync.Once
var agentService *AgentService

// GetAgentService implements Singleton pattern
func GetAgentService() *AgentService {
	onceAgentService.Do(func() {
		agentService = &AgentService{
			transit.Transit{Config: config.GetConfig()},
			&AgentStats{},
			&AgentStatus{
				Controller: Pending,
				NATS:       Pending,
				Transport:  Pending,
			},
		}
	})
	return agentService
}

// StartController implements AgentServices.StartController interface
func (service *AgentService) StartController() error {
	// NOTE: the service.agentStatus.Controller will be updated by controller itself
	return GetController().StartController()
}

// StopController implements AgentServices.StopController interface
func (service *AgentService) StopController() error {
	// NOTE: the service.agentStatus.Controller will be updated by controller itself
	return GetController().StopController()
}

// StartNATS implements AgentServices.StartNATS interface
func (service *AgentService) StartNATS() error {
	err := nats.StartServer(service.AgentConfig.NATSStoreType, service.AgentConfig.NATSFilestoreDir)
	if err == nil {
		service.agentStatus.Lock()
		service.agentStatus.NATS = Running
		service.agentStatus.Unlock()
	}
	return err
}

// StopNATS implements AgentServices.StopNATS interface
func (service *AgentService) StopNATS() error {
	nats.StopServer()
	service.agentStatus.Lock()
	service.agentStatus.NATS = Stopped
	service.agentStatus.Unlock()
	return nil
}

// StartTransport implements AgentServices.StartTransport interface
func (service *AgentService) StartTransport() error {
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
func (service *AgentService) StopTransport() error {
	err := nats.StopDispatcher()
	if err == nil {
		service.agentStatus.Lock()
		service.agentStatus.Transport = Stopped
		service.agentStatus.Unlock()
	}
	return err
}

// Stats implements AgentServices.Stats interface
func (service *AgentService) Stats() *AgentStats {
	return service.agentStats
}

// Status implements AgentServices.Status interface
func (service *AgentService) Status() *AgentStatus {
	return service.agentStatus
}
