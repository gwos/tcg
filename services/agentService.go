package services

import (
	"fmt"
	"github.com/gwos/tng/clients"
	"github.com/gwos/tng/config"
	"github.com/gwos/tng/milliseconds"
	"github.com/gwos/tng/nats"
	"github.com/gwos/tng/transit"
	"github.com/hashicorp/go-uuid"
	"math"
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
	ctrlIdx     uint8
	ctrlChan    chan *CtrlAction
	statsChan   chan statsCounter
}

// CtrlAction defines queued controll action
type CtrlAction struct {
	Idx      uint8
	Subj     ctrlSubj
	SyncChan chan error
}

type statsCounter struct {
	subject   string
	bytesSent int
	lastError error
}

type ctrlSubj string

const (
	ctrlSubjReload          ctrlSubj = "reload"
	ctrlSubjStartController          = "startController"
	ctrlSubjStopController           = "stopController"
	ctrlSubjStartNats                = "startNats"
	ctrlSubjStopNats                 = "stopNats"
	ctrlSubjStartTransport           = "startTransport"
	ctrlSubjStopTransport            = "stopTransport"
)

const ctrlLimit = 9

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
				Controller: Stopped,
				Nats:       Stopped,
				Transport:  Stopped,
			},
			&clients.DSClient{
				AppName:      agentConnector.AppName,
				DSConnection: config.GetConfig().DSConnection,
			},
			nil,
			0,
			make(chan *CtrlAction, ctrlLimit),
			make(chan statsCounter),
		}
		go agentService.listenCtrlChan()
		go agentService.listenStatsChan()
		agentService.Reload()
	})
	return agentService
}

// MakeTracerContext implements AgentServices.MakeTracerContext interface
func (service *AgentService) MakeTracerContext() (transit.TracerContext, error) {
	traceToken, err := uuid.GenerateUUID()
	if err != nil {
		return transit.TracerContext{}, err
	}
	return transit.TracerContext{
		AgentID:    service.Connector.AgentID,
		AppType:    service.Connector.AppType,
		TimeStamp:  milliseconds.MillisecondTimestamp{Time: time.Now()},
		TraceToken: traceToken,
		Version:    transit.TransitModelVersion,
	}, nil
}

// ReloadAsync implements AgentServices.ReloadAsync interface
func (service *AgentService) ReloadAsync(syncChan chan error) (*CtrlAction, error) {
	return service.ctrlPushAsync(ctrlSubjReload, syncChan)
}

// StartControllerAsync implements AgentServices.StartControllerAsync interface
func (service *AgentService) StartControllerAsync(syncChan chan error) (*CtrlAction, error) {
	return service.ctrlPushAsync(ctrlSubjStartController, syncChan)
}

// StopControllerAsync implements AgentServices.StopControllerAsync interface
func (service *AgentService) StopControllerAsync(syncChan chan error) (*CtrlAction, error) {
	return service.ctrlPushAsync(ctrlSubjStopController, syncChan)
}

// StartNatsAsync implements AgentServices.StartNatsAsync interface
func (service *AgentService) StartNatsAsync(syncChan chan error) (*CtrlAction, error) {
	return service.ctrlPushAsync(ctrlSubjStartNats, syncChan)
}

// StopNatsAsync implements AgentServices.StopNatsAsync interface
func (service *AgentService) StopNatsAsync(syncChan chan error) (*CtrlAction, error) {
	return service.ctrlPushAsync(ctrlSubjStopNats, syncChan)
}

// StartTransportAsync implements AgentServices.StartTransportAsync interface.
func (service *AgentService) StartTransportAsync(syncChan chan error) (*CtrlAction, error) {
	return service.ctrlPushAsync(ctrlSubjStartTransport, syncChan)
}

// StopTransportAsync implements AgentServices.StopTransportAsync interface
func (service *AgentService) StopTransportAsync(syncChan chan error) (*CtrlAction, error) {
	return service.ctrlPushAsync(ctrlSubjStopTransport, syncChan)
}

// Reload implements AgentServices.Reload interface
func (service *AgentService) Reload() error {
	return service.ctrlPushSync(ctrlSubjReload)
}

// StartController implements AgentServices.StartController interface
func (service *AgentService) StartController() error {
	return service.ctrlPushSync(ctrlSubjStartController)
}

// StopController implements AgentServices.StopController interface
func (service *AgentService) StopController() error {
	return service.ctrlPushSync(ctrlSubjStopController)
}

// StartNats implements AgentServices.StartNats interface
func (service *AgentService) StartNats() error {
	return service.ctrlPushSync(ctrlSubjStartNats)
}

// StopNats implements AgentServices.StopNats interface
func (service *AgentService) StopNats() error {
	return service.ctrlPushSync(ctrlSubjStopNats)
}

// StartTransport implements AgentServices.StartTransport interface.
func (service *AgentService) StartTransport() error {
	return service.ctrlPushSync(ctrlSubjStartTransport)
}

// StopTransport implements AgentServices.StopTransport interface
func (service *AgentService) StopTransport() error {
	return service.ctrlPushSync(ctrlSubjStopTransport)
}

// Stats implements AgentServices.Stats interface
func (service *AgentService) Stats() AgentStats {
	return *service.agentStats
}

// Status implements AgentServices.Status interface
func (service *AgentService) Status() AgentStatus {
	return *service.agentStatus
}

func (service *AgentService) ctrlPushAsync(subj ctrlSubj, syncChan chan error) (*CtrlAction, error) {
	ctrl := &CtrlAction{service.ctrlIdx + 1, subj, syncChan}
	select {
	case service.ctrlChan <- ctrl:
		service.ctrlIdx = ctrl.Idx
		if service.ctrlIdx > (math.MaxUint8 - 1) {
			service.ctrlIdx = 0
		}
		return ctrl, nil
	default:
		return nil, fmt.Errorf("Ctrl limit reached: %v", ctrlLimit)
	}
}

func (service *AgentService) ctrlPushSync(subj ctrlSubj) error {
	syncChan := make(chan error)
	if _, err := service.ctrlPushAsync(subj, syncChan); err != nil {
		return err
	}
	return <-syncChan
}

func (service *AgentService) listenCtrlChan() {
	for {
		ctrl := <-service.ctrlChan
		service.agentStatus.Ctrl = ctrl
		var err error
		switch ctrl.Subj {
		case ctrlSubjReload:
			err = service.reload()
		case ctrlSubjStartController:
			err = service.startController()
		case ctrlSubjStopController:
			err = service.stopController()
		case ctrlSubjStartNats:
			err = service.startNats()
		case ctrlSubjStopNats:
			err = service.stopNats()
		case ctrlSubjStartTransport:
			err = service.startTransport()
		case ctrlSubjStopTransport:
			err = service.stopTransport()
		}
		// TODO: provide timeout
		if ctrl.SyncChan != nil {
			ctrl.SyncChan <- err
		}
		service.agentStatus.Ctrl = nil
	}
}

func (service *AgentService) listenStatsChan() {
	for {
		res := <-service.statsChan

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
				service.statsChan <- statsCounter{
					bytesSent: len(b),
					lastError: nil,
					subject:   subj,
				}
			} else {
				service.statsChan <- statsCounter{
					bytesSent: 0,
					lastError: err,
					subject:   subj,
				}
			}
			return err
		},
	}
}

func (service *AgentService) reload() error {
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
		service.stopController()
		service.startController()
	}
	if reloadFlags.Nats {
		service.stopNats()
		service.startNats()
	}
	if reloadFlags.Transport {
		// service.StopTransport() // stopped with Nats
		service.startTransport()
	}

	return nil
}

func (service *AgentService) startController() error {
	// NOTE: the service.agentStatus.Controller will be updated by controller itself
	return GetController().startController()
}

func (service *AgentService) stopController() error {
	// NOTE: the service.agentStatus.Controller will be updated by controller itself
	if service.agentStatus.Controller == Stopped {
		return nil
	}
	return GetController().stopController()
}

func (service *AgentService) startNats() error {
	err := nats.StartServer(nats.Config{
		DispatcherAckWait:     time.Second * time.Duration(service.Connector.NatsAckWait),
		DispatcherMaxInflight: service.Connector.NatsMaxInflight,
		MaxPubAcksInflight:    service.Connector.NatsMaxInflight,
		FilestoreDir:          service.Connector.NatsFilestoreDir,
		StoreType:             service.Connector.NatsStoreType,
		NatsHost:              service.Connector.NatsHost,
	})
	if err == nil {
		service.agentStatus.Nats = Running
	}
	return err
}

func (service *AgentService) stopNats() error {
	if service.agentStatus.Nats == Stopped {
		return nil
	}

	// Stop Transport as dependency
	err := service.stopTransport()
	// skip Stop Transport error checking
	nats.StopServer()
	service.agentStatus.Nats = Stopped
	return err
}

func (service *AgentService) startTransport() error {
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
		service.agentStatus.Transport = Running
	} else {
		return sdErr
	}
	return nil
}

func (service *AgentService) stopTransport() error {
	if service.agentStatus.Transport == Stopped {
		return nil
	}

	err := nats.StopDispatcher()
	if err == nil {
		service.agentStatus.Transport = Stopped
	}
	return err
}
