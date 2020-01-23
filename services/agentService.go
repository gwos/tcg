package services

import (
	"fmt"
	"github.com/gwos/tng/clients"
	"github.com/gwos/tng/config"
	"github.com/gwos/tng/milliseconds"
	"github.com/gwos/tng/nats"
	"gopkg.in/yaml.v3"
	"sync"
	"time"
)

// AgentService implements AgentServices interface
type AgentService struct {
	*config.Connector
	agentStats  *AgentStats
	agentStatus *AgentStatus
	dsClient    *clients.DSClient
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
		agentConnector := config.GetConfig().Connector
		agentService = &AgentService{
			agentConnector,
			&AgentStats{
				AgentID: agentConnector.AgentID,
				AppType: agentConnector.AppType,
			},
			&AgentStatus{
				Controller: Pending,
				Nats:       Pending,
				Transport:  Pending,
			},
			&clients.DSClient{
				AppName:      agentConnector.AppName,
				DSConnection: config.GetConfig().DSConnection,
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
		DispatcherAckWait:     time.Second * time.Duration(service.Connector.NatsAckWait),
		DispatcherMaxInflight: service.Connector.NatsMaxInflight,
		MaxPubAcksInflight:    service.Connector.NatsMaxInflight,
		FilestoreDir:          service.Connector.NatsFilestoreDir,
		StoreType:             service.Connector.NatsStoreType,
		NatsHost:              service.Connector.NatsHost,
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

// StartTransport implements AgentServices.StartTransport interface.
// It's called by Controller with updated config or without args on startup.
// So, if args empty it uses local config and tries update it with DSClient.
func (service *AgentService) StartTransport(cons ...*config.GWConnection) error {
	if len(cons) == 0 {
		cons = config.GetConfig().GWConnections
		// Try fetch GWConnections with DSClient
		var resCons config.GWConnections
		if res, clErr := service.dsClient.FetchGWConnections(service.AgentID); clErr == nil {
			if err := yaml.Unmarshal(res, &resCons); err == nil {
				config.GetConfig().GWConnections = resCons
				cons = resCons
			}
		}
	}
	if len(cons) == 0 {
		return fmt.Errorf("StartTransport: %v", "empty GWConnections")
	}
	gwClients := make([]*clients.GWClient, len(cons))
	for i := range cons {
		gwClients[i] = &clients.GWClient{
			AppName:      service.AppName,
			GWConnection: cons[i],
		}
	}

	service.gwClients = gwClients

	var dispatcherOptions []nats.DispatcherOption
	for _, gwClient := range service.gwClients {
		gwClientCopy := gwClient
		durableID := fmt.Sprintf("%s", gwClient.HostName)
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
			nats.DispatcherOption{
				DurableID: durableID,
				Subject:   SubjSendEvent,
				Handler: func(b []byte) error {
					// TODO: filter the message by rules per gwClient
					_, err := gwClientCopy.SendEvent(b)
					if err == nil {
						res := statsCounter{
							subject:   SubjSendEvent,
							bytesSent: len(b),
							lastError: nil,
						}
						service.statsChanel <- res
					} else {
						res := statsCounter{
							subject:   SubjSendEvent,
							bytesSent: 0,
							lastError: err,
						}
						service.statsChanel <- res
					}
					return nil
				},
			},
		)
	}

	if sdErr := nats.StartDispatcher(dispatcherOptions); sdErr == nil {
		service.agentStatus.Lock()
		service.agentStatus.Transport = Running
		service.agentStatus.Unlock()
	} else {
		return sdErr
	}
	return nil
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
			case SubjSendEvent:
				service.agentStats.LastAlertRun = milliseconds.MillisecondTimestamp{Time: time.Now()}
			}

		}
	}
}
