package services

import (
	"fmt"
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
	statsChanel chan statsCounter
}

type statsCounter struct {
	subject   string
	bytesSent int
	lastError error
}

var onceAgentService sync.Once
var agentService *AgentService

// GetAgentService implements Singleton pattern
func GetAgentService() *AgentService {
	onceAgentService.Do(func() {
		agentService = &AgentService{
			config.GetConfig().AgentConfig,
			&AgentStats{},
			&AgentStatus{
				Controller: Pending,
				Nats:       Pending,
				Transport:  Pending,
			},
			nil,
			make(chan statsCounter),
		}
		go agentService.listenChanel()
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
	if service.agentStatus.Controller == Stopped || service.agentStatus.Controller == Pending {
		return nil
	}
	return GetController().StopController()
}

// StartNats implements AgentServices.StartNats interface
func (service *AgentService) StartNats() error {
	err := nats.StartServer(nats.Config{
		DispatcherAckWait:     time.Second * time.Duration(service.AgentConfig.NatsAckWait),
		DispatcherMaxInflight: service.AgentConfig.NatsMaxInflight,
		MaxPubAcksInflight:    service.AgentConfig.NatsMaxInflight,
		FilestoreDir:          service.AgentConfig.NatsFilestoreDir,
		StoreType:             service.AgentConfig.NatsStoreType,
		NatsHost:              service.AgentConfig.NatsHost,
	})
	if err == nil {
		service.agentStatus.Lock()
		service.agentStatus.Nats = Running
		service.agentStatus.Unlock()
	}
	return err
}

// StopNats implements AgentServices.StopNats interface
func (service *AgentService) StopNats() error {
	if service.agentStatus.Nats == Stopped || service.agentStatus.Nats == Pending {
		return nil
	}

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
	gwConfigs := config.GetConfig().GWConfigs
	if len(gwConfigs) == 0 {
		return fmt.Errorf("StartTransport: %v", "empty GWConfigs")
	}
	gwClients := make([]*clients.GWClient, len(gwConfigs))
	for i := range gwConfigs {
		gwClients[i] = &clients.GWClient{GWConfig: gwConfigs[i]}
	}

	service.gwClients = gwClients

	var dispatcherOptions []nats.DispatcherOption
	for _, gwClient := range service.gwClients {
		gwClientCopy := gwClient
		durableID := fmt.Sprintf("%s", gwClient.Host)
		dispatcherOptions = append(
			dispatcherOptions,
			nats.DispatcherOption{
				DurableID: durableID,
				Subject:   SubjSendResourceWithMetrics,
				Handler: func(b []byte) error {
					// TODO: filter the message by rules per gwClient
					_, err := gwClientCopy.SendResourcesWithMetrics(b)
					if err == nil {
						res := statsCounter{
							subject:   SubjSendResourceWithMetrics,
							bytesSent: len(b),
							lastError: nil,
						}
						service.statsChanel <- res
					} else {
						res := statsCounter{
							subject:   SubjSendResourceWithMetrics,
							bytesSent: 0,
							lastError: err,
						}
						service.statsChanel <- res
					}
					return err
				},
			},
			nats.DispatcherOption{
				DurableID: durableID,
				Subject:   SubjSynchronizeInventory,
				Handler: func(b []byte) error {
					// TODO: filter the message by rules per gwClient
					_, err := gwClientCopy.SynchronizeInventory(b)
					if err == nil {
						res := statsCounter{
							subject:   SubjSynchronizeInventory,
							bytesSent: len(b),
							lastError: nil,
						}
						service.statsChanel <- res
					} else {
						res := statsCounter{
							subject:   SubjSynchronizeInventory,
							bytesSent: 0,
							lastError: err,
						}
						service.statsChanel <- res
					}
					return err
				},
			},
		)
	}

	err := nats.StartDispatcher(dispatcherOptions)
	if err == nil {
		service.agentStatus.Lock()
		service.agentStatus.Transport = Running
		service.agentStatus.Unlock()
	}
	return err
}

// StopTransport implements AgentServices.StopTransport interface
func (service *AgentService) StopTransport() error {
	if service.agentStatus.Transport == Stopped || service.agentStatus.Transport == Pending {
		return nil
	}

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

func (service *AgentService) listenChanel() {
	for {
		res := <-service.statsChanel

		if res.lastError != nil {
			service.agentStats.LastError = res.lastError.Error()
		} else {
			service.agentStats.BytesSent += res.bytesSent
			service.agentStats.MessagesSent++
			switch res.subject {
			case SubjSynchronizeInventory:
				service.agentStats.LastInventoryRun = milliseconds.MillisecondTimestamp{Time: time.Now()}
			case SubjSendResourceWithMetrics:
				service.agentStats.LastMetricsRun = milliseconds.MillisecondTimestamp{Time: time.Now()}
			}
		}
	}
}
