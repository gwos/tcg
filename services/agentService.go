package services

import (
	"github.com/gwos/tng/clients"
	"github.com/gwos/tng/config"
	"github.com/gwos/tng/milliseconds"
	"github.com/gwos/tng/nats"
	"sync"
	"time"
)

// AgentService implements AgentServices interface
type AgentService struct {
	*config.AgentConfig
	agentStats  *AgentStats
	agentStatus *AgentStatus
	gwClients   []*clients.GWClient
}

var onceAgentService sync.Once
var agentService *AgentService

// GetAgentService implements Singleton pattern
func GetAgentService() *AgentService {
	onceAgentService.Do(func() {
		gwConfigs := config.GetConfig().GWConfigs
		gwClients := make([]*clients.GWClient, len(gwConfigs))
		for i := range gwConfigs {
			gwClients[i] = &clients.GWClient{GWConfig: gwConfigs[i]}
		}

		agentService = &AgentService{
			config.GetConfig().AgentConfig,
			&AgentStats{},
			&AgentStatus{
				Controller: Pending,
				Nats:       Pending,
				Transport:  Pending,
			},
			gwClients,
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

// StartNats implements AgentServices.StartNats interface
func (service *AgentService) StartNats() error {
	err := nats.StartServer(nats.Config{
		DispatcherAckWait: time.Second * time.Duration(service.AgentConfig.NatsAckWait),
		FilestoreDir:      service.AgentConfig.NatsFilestoreDir,
		StoreType:         service.AgentConfig.NatsStoreType,
		NatsHost:          service.AgentConfig.NatsHost,
	})
	if err == nil {
		service.agentStatus.Lock()
		service.agentStatus.Nats = Running
		service.agentStatus.Unlock()
		// StartTransport as dependency
		if service.AgentConfig.StartTransport {
			err = service.StartTransport()
		}
	}
	return err
}

// StopNats implements AgentServices.StopNats interface
func (service *AgentService) StopNats() error {
	// StopTransport as dependency
	err := service.StopTransport()
	// skip StopTransport error checking
	nats.StopServer()
	service.agentStatus.Lock()
	service.agentStatus.Nats = Stopped
	service.agentStatus.Unlock()
	return err
}

// StartTransport implements AgentServices.StartTransport interface
func (service *AgentService) StartTransport() error {
	dispatcherMap := nats.DispatcherMap{
		SubjSendResourceWithMetrics: func(b []byte) error {
			_, err := service.gwClients[0].SendResourcesWithMetrics(b)
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
			_, err := service.gwClients[0].SynchronizeInventory(b)
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
