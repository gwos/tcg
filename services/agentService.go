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
		agentService.Reload()
	})
	return agentService
}

// Reload implements AgentServices.Reload interface
func (service *AgentService) Reload() error {
	if res, clErr := service.dsClient.FetchConnector(service.AgentID); clErr == nil {
		if err := config.GetConfig().LoadConnectorDTO(res); err != nil {
			return err
		}
	} else {
		return clErr
	}

	reloadFlags := struct {
		Controller bool
		Transport  bool
		Nats       bool
	}{
		service.Status().Controller == Running,
		service.Status().Transport == Running,
		service.Status().Nats == Running,
	}
	// TODO: Handle errors
	if reloadFlags.Controller {
		service.StopController()
		service.StartController()
	}
	if reloadFlags.Nats {
		service.StopNats()
		service.StartNats()
	}
	if reloadFlags.Transport {
		// service.StopTransport() // stopped with Nats
		service.StartTransport()
	}

	return nil
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
func (service *AgentService) StartTransport() error {
	cons := config.GetConfig().GWConnections
	if len(cons) == 0 {
		return fmt.Errorf("StartTransport: %v", "empty GWConnections")
	}
	/* Process clients */
	gwClients := make([]*clients.GWClient, len(cons))
	for i := range cons {
		gwClients[i] = &clients.GWClient{
			AppName:      service.AppName,
			GWConnection: cons[i],
		}
	}
	service.gwClients = gwClients
	/* Process dispatcher */
	if sdErr := nats.StartDispatcher(service.makeDispatcherOptions()); sdErr == nil {
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

func (service *AgentService) makeDispatcherOptions() []nats.DispatcherOption {
	var dispatcherOptions []nats.DispatcherOption
	for _, gwClient := range service.gwClients {
		// TODO: filter the message by rules per gwClient
		gwClientRef := gwClient
		durableID := fmt.Sprintf("%s", gwClient.HostName)
		dispatcherOptions = append(
			dispatcherOptions,
			service.makeDispatcherOption(
				durableID,
				SubjSendEvent,
				func(b []byte) error {
					_, err := gwClientRef.SendEvent(b)
					return err
				},
			),
			service.makeDispatcherOption(
				durableID,
				SubjSendResourceWithMetrics,
				func(b []byte) error {
					_, err := gwClientRef.SendResourcesWithMetrics(b)
					return err
				},
			),
			service.makeDispatcherOption(
				durableID,
				SubjSynchronizeInventory,
				func(b []byte) error {
					_, err := gwClientRef.SynchronizeInventory(b)
					return err
				},
			),
		)
	}
	return dispatcherOptions
}

func (service *AgentService) makeDispatcherOption(durableID, subj string, subjFn func([]byte) error) nats.DispatcherOption {
	return nats.DispatcherOption{
		DurableID: durableID,
		Subject:   subj,
		Handler: func(b []byte) error {
			// TODO: filter the message by rules per gwClient
			err := subjFn(b)
			if err == nil {
				service.statsChanel <- statsCounter{
					bytesSent: len(b),
					lastError: nil,
					subject:   subj,
				}
			} else {
				service.statsChanel <- statsCounter{
					bytesSent: 0,
					lastError: err,
					subject:   subj,
				}
			}
			return err
		},
	}
}
