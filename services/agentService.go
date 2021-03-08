package services

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/gwos/tcg/cache"
	"github.com/gwos/tcg/clients"
	"github.com/gwos/tcg/config"
	tcgerr "github.com/gwos/tcg/errors"
	"github.com/gwos/tcg/log"
	"github.com/gwos/tcg/milliseconds"
	"github.com/gwos/tcg/nats"
	"github.com/gwos/tcg/transit"
	"github.com/hashicorp/go-uuid"
	apitrace "go.opentelemetry.io/otel/api/trace"
	"go.opentelemetry.io/otel/exporters/trace/jaeger"
	"go.opentelemetry.io/otel/label"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

// AgentService implements AgentServices interface
type AgentService struct {
	*sync.Mutex
	*config.Connector
	agentStats            *AgentStats
	agentStatus           *AgentStatus
	dsClient              *clients.DSClient
	gwClients             []*clients.GWClient
	ctrlIdx               uint8
	ctrlChan              chan *CtrlAction
	statsChan             chan statsCounter
	tracerToken           []byte
	configHandler         func([]byte)
	demandConfigHandler   func() bool
	exitHandler           func()
	telemetryFlushHandler func()
	telemetryProvider     apitrace.TracerProvider
}

// CtrlAction defines queued controll action
type CtrlAction struct {
	Data     interface{}
	Idx      uint8
	Subj     ctrlSubj
	SyncChan chan error
}

type statsCounter struct {
	bytesSent   int
	lastError   error
	payloadType payloadType
	timestamp   time.Time
}

type ctrlSubj string

const (
	ctrlSubjConfig          ctrlSubj = "config"
	ctrlSubjStartController          = "startController"
	ctrlSubjStopController           = "stopController"
	ctrlSubjStartNats                = "startNats"
	ctrlSubjStopNats                 = "stopNats"
	ctrlSubjStartTransport           = "startTransport"
	ctrlSubjStopTransport            = "stopTransport"
)

const ctrlLimit = 9
const ckTraceToken = "ckTraceToken"
const statsLastErrorsLim = 10
const defaultDeadlineTimer = 9 * time.Second
const traceOnDemandAgentID = "#traceOnDemandAgentID#"
const traceOnDemandAppType = "#traceOnDemandAppType#"

var onceAgentService sync.Once
var agentService *AgentService

// GetAgentService implements Singleton pattern
func GetAgentService() *AgentService {
	onceAgentService.Do(func() {
		telemetryProvider, telemetryFlushHandler, _ := initTelemetryProvider()

		/* prepare random tracerToken */
		tracerToken := []byte("aaaabbbbccccdddd")
		if randBuf, err := uuid.GenerateRandomBytes(16); err == nil {
			copy(tracerToken, randBuf)
		} else {
			/* fallback with multiplied timestamp */
			binary.PutVarint(tracerToken, time.Now().UnixNano())
			binary.PutVarint(tracerToken[6:], time.Now().UnixNano())
		}
		cache.TraceTokenCache.Set(ckTraceToken, uint64(1), -1)

		agentConnector := config.GetConfig().Connector
		agentService = &AgentService{
			&sync.Mutex{},
			agentConnector,
			&AgentStats{
				UpSince: &milliseconds.MillisecondTimestamp{Time: time.Now()},
			},
			&AgentStatus{
				Controller: Stopped,
				Nats:       Stopped,
				Transport:  Stopped,
			},
			&clients.DSClient{DSConnection: config.GetConfig().DSConnection},
			nil,
			0,
			make(chan *CtrlAction, ctrlLimit),
			make(chan statsCounter),
			tracerToken,
			defaultConfigHandler,
			defaultDemandConfigHandler,
			defaultExitHandler,
			telemetryFlushHandler,
			telemetryProvider,
		}

		go agentService.listenCtrlChan()
		go agentService.listenStatsChan()
		agentService.handleExit()

		log.SetHook(agentService.hookLogErrors, log.ErrorLevel)
		log.With(log.Fields{
			"AgentID":        agentService.AgentID,
			"AppType":        agentService.AppType,
			"AppName":        agentService.AppName,
			"ControllerAddr": agentService.ControllerAddr,
			"DSClient":       agentService.dsClient.HostName,
		}).Log(log.DebugLevel, "#AgentService Config")
	})

	return agentService
}

// DemandConfig implements AgentServices.DemandConfig interface
func (service *AgentService) DemandConfig() error {
	if err := service.StartController(); err != nil {
		return err
	}
	if config.GetConfig().IsConfiguringPMC() {
		log.Info("[Demand Config]: Configuring PARENT_MANAGED_CHILD")
		// expect the config api call
		return nil
	}
	if len(service.AgentID) == 0 || len(service.dsClient.HostName) == 0 {
			log.Info("[Demand Config]: Config Server is not configured")
		// expect the config api call
		return nil
	}

	go func() {
		for i := 0; ; i++ {
			if err := service.dsClient.Reload(service.AgentID); err != nil {
				log.With(log.Fields{"error": err}).
					Log(log.ErrorLevel, "[Demand Config]: Config Server is not available")
				time.Sleep(time.Duration((i%4+1)*5) * time.Second)
				continue
			}
			break
		}
		log.Info("[Demand Config]: Config Server found and connected")
	}()
	return nil
}

// MakeTracerContext implements AgentServices.MakeTracerContext interface
func (service *AgentService) MakeTracerContext() *transit.TracerContext {
	/* combine TraceToken from fixed and incremental parts */
	tokenBuf := make([]byte, 16)
	copy(tokenBuf, service.tracerToken)
	if tokenInc, err := cache.TraceTokenCache.IncrementUint64(ckTraceToken, 1); err == nil {
		binary.PutUvarint(tokenBuf, tokenInc)
	} else {
		/* fallback with timestamp */
		binary.PutVarint(tokenBuf, time.Now().UnixNano())
	}
	traceToken, _ := uuid.FormatUUID(tokenBuf)

	/* use placeholders on demand config, then replace on fixTracerContext */
	agentID := service.Connector.AgentID
	appType := service.Connector.AppType
	if len(agentID) == 0 {
		agentID = traceOnDemandAgentID
	}
	if len(appType) == 0 {
		appType = traceOnDemandAppType
	}

	return &transit.TracerContext{
		AgentID:    agentID,
		AppType:    appType,
		TimeStamp:  milliseconds.MillisecondTimestamp{Time: time.Now()},
		TraceToken: traceToken,
		Version:    transit.ModelVersion,
	}
}

func defaultConfigHandler([]byte) {}

func defaultDemandConfigHandler() bool { return true }

func defaultExitHandler() {}

// RegisterConfigHandler sets callback
// usefull for process extensions
func (service *AgentService) RegisterConfigHandler(fn func([]byte)) {
	service.configHandler = fn
}

// RemoveConfigHandler removes callback
func (service *AgentService) RemoveConfigHandler() {
	service.configHandler = defaultConfigHandler
}

// RegisterDemandConfigHandler sets callback
func (service *AgentService) RegisterDemandConfigHandler(fn func() bool) {
	service.demandConfigHandler = fn
}

// RemoveDemandConfigHandler removes callback
func (service *AgentService) RemoveDemandConfigHandler() {
	service.demandConfigHandler = defaultDemandConfigHandler
}

// RegisterExitHandler sets callback
func (service *AgentService) RegisterExitHandler(fn func()) {
	service.exitHandler = fn
}

// RemoveExitHandler removes callback
func (service *AgentService) RemoveExitHandler() {
	service.exitHandler = defaultExitHandler
}

// StartControllerAsync implements AgentServices.StartControllerAsync interface
func (service *AgentService) StartControllerAsync(syncChan chan error) (*CtrlAction, error) {
	return service.ctrlPushAsync(nil, ctrlSubjStartController, syncChan)
}

// StopControllerAsync implements AgentServices.StopControllerAsync interface
func (service *AgentService) StopControllerAsync(syncChan chan error) (*CtrlAction, error) {
	return service.ctrlPushAsync(nil, ctrlSubjStopController, syncChan)
}

// StartNatsAsync implements AgentServices.StartNatsAsync interface
func (service *AgentService) StartNatsAsync(syncChan chan error) (*CtrlAction, error) {
	return service.ctrlPushAsync(nil, ctrlSubjStartNats, syncChan)
}

// StopNatsAsync implements AgentServices.StopNatsAsync interface
func (service *AgentService) StopNatsAsync(syncChan chan error) (*CtrlAction, error) {
	return service.ctrlPushAsync(nil, ctrlSubjStopNats, syncChan)
}

// StartTransportAsync implements AgentServices.StartTransportAsync interface.
func (service *AgentService) StartTransportAsync(syncChan chan error) (*CtrlAction, error) {
	return service.ctrlPushAsync(nil, ctrlSubjStartTransport, syncChan)
}

// StopTransportAsync implements AgentServices.StopTransportAsync interface
func (service *AgentService) StopTransportAsync(syncChan chan error) (*CtrlAction, error) {
	return service.ctrlPushAsync(nil, ctrlSubjStopTransport, syncChan)
}

// StartController implements AgentServices.StartController interface
func (service *AgentService) StartController() error {
	return service.ctrlPushSync(nil, ctrlSubjStartController)
}

// StopController implements AgentServices.StopController interface
func (service *AgentService) StopController() error {
	return service.ctrlPushSync(nil, ctrlSubjStopController)
}

// StartNats implements AgentServices.StartNats interface
func (service *AgentService) StartNats() error {
	return service.ctrlPushSync(nil, ctrlSubjStartNats)
}

// StopNats implements AgentServices.StopNats interface
func (service *AgentService) StopNats() error {
	return service.ctrlPushSync(nil, ctrlSubjStopNats)
}

// StartTransport implements AgentServices.StartTransport interface.
func (service *AgentService) StartTransport() error {
	return service.ctrlPushSync(nil, ctrlSubjStartTransport)
}

// StopTransport implements AgentServices.StopTransport interface
func (service *AgentService) StopTransport() error {
	return service.ctrlPushSync(nil, ctrlSubjStopTransport)
}

// Stats implements AgentServices.Stats interface
func (service *AgentService) Stats() AgentIdentityStats {
	return AgentIdentityStats{AgentIdentity{
		AgentID: service.Connector.AgentID,
		AppName: service.Connector.AppName,
		AppType: service.Connector.AppType,
	},
		*service.agentStats,
	}
}

// Status implements AgentServices.Status interface
func (service *AgentService) Status() AgentStatus {
	return *service.agentStatus
}

func (service *AgentService) ctrlPushAsync(data interface{}, subj ctrlSubj, syncChan chan error) (*CtrlAction, error) {
	ctrl := &CtrlAction{data, service.ctrlIdx + 1, subj, syncChan}
	select {
	case service.ctrlChan <- ctrl:
		service.ctrlIdx = ctrl.Idx
		if service.ctrlIdx > (math.MaxUint8 - 1) {
			service.ctrlIdx = 0
		}
		return ctrl, nil
	default:
		return nil, fmt.Errorf("Ctrl limit reached: %v ", ctrlLimit)
	}
}

func (service *AgentService) ctrlPushSync(data interface{}, subj ctrlSubj) error {
	syncChan := make(chan error)
	if _, err := service.ctrlPushAsync(data, subj, syncChan); err != nil {
		return err
	}
	return <-syncChan
}

func (service *AgentService) listenCtrlChan() {
	deadlineTimer := time.NewTimer(defaultDeadlineTimer)
	deadlineTimer.Stop()
	var subject ctrlSubj
	go deadlineTimerHandler(deadlineTimer, &subject)
	for {
		ctrl := <-service.ctrlChan
		logEntry := log.With(log.Fields{
			"Idx":  ctrl.Idx,
			"Subj": ctrl.Subj,
			// "Data": string(ctrl.Data),
		})
		logEntry.Log(log.DebugLevel, "#AgentService.ctrlChan")
		subject = ctrl.Subj

		deadlineTimer.Reset(defaultDeadlineTimer)

		service.agentStatus.Ctrl = ctrl
		var err error
		switch ctrl.Subj {
		case ctrlSubjConfig:
			err = service.config(ctrl.Data.([]byte))
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

		deadlineTimer.Stop()

		if ctrl.SyncChan != nil {
			ctrl.SyncChan <- err
		}
		service.agentStatus.Ctrl = nil
	}
}

func (service *AgentService) listenStatsChan() {
	for {
		res := <-service.statsChan

		ts := milliseconds.MillisecondTimestamp{Time: res.timestamp}
		if res.lastError != nil {
			service.agentStats.LastErrors = append(service.agentStats.LastErrors, LastError{
				res.lastError.Error(),
				&ts,
			})
			statsLastErrorsLen := len(service.agentStats.LastErrors)
			if statsLastErrorsLen > statsLastErrorsLim {
				service.agentStats.LastErrors = service.agentStats.LastErrors[(statsLastErrorsLen - statsLastErrorsLim):]
			}
		} else {
			service.agentStats.BytesSent += res.bytesSent
			service.agentStats.MessagesSent++
			switch res.payloadType {
			case typeInventory:
				service.agentStats.LastInventoryRun = &ts
			case typeMetrics:
				service.agentStats.LastMetricsRun = &ts
				service.agentStats.MetricsSent++
			case typeEvents:
				// TODO: handle events acks, unacks
				service.agentStats.LastAlertRun = &ts
			}

		}
	}
}

func (service *AgentService) updateStats(bytesSent int, lastError error, payloadType payloadType, timestamp time.Time) {
	service.statsChan <- statsCounter{
		bytesSent:   bytesSent,
		lastError:   lastError,
		payloadType: payloadType,
		timestamp:   timestamp,
	}
}

func (service *AgentService) makeDispatcherOptions() []nats.DispatcherOption {
	var dispatcherOptions []nats.DispatcherOption
	for _, gwClient := range service.gwClients {
		// TODO: filter the message by rules per gwClient
		gwClientRef := gwClient
		dispatcherOptions = append(
			dispatcherOptions,
			service.makeDispatcherOption(
				fmt.Sprintf("#%s#%s#", subjEvents, gwClientRef.HostName),
				subjEvents,
				func(ctx context.Context, p natsPayload) error {
					var err error
					switch p.Type {
					case typeEvents:
						_, err = gwClientRef.SendEvents(ctx, p.Payload)
					case typeEventsAck:
						_, err = gwClientRef.SendEventsAck(ctx, p.Payload)
					case typeEventsUnack:
						_, err = gwClientRef.SendEventsUnack(ctx, p.Payload)
					default:
						err = fmt.Errorf("dispatcher error on process payload type %s:%s", p.Type, subjEvents)
					}
					return err
				},
			),
			service.makeDispatcherOption(
				fmt.Sprintf("#%s#%s#", subjInventoryMetrics, gwClientRef.HostName),
				subjInventoryMetrics,
				func(ctx context.Context, p natsPayload) error {
					var err error
					switch p.Type {
					case typeInventory:
						_, err = gwClientRef.SynchronizeInventory(ctx, service.fixTracerContext(p.Payload))
					case typeMetrics:
						_, err = gwClientRef.SendResourcesWithMetrics(ctx, service.fixTracerContext(p.Payload))
					default:
						err = fmt.Errorf("dispatcher error on process payload type %s:%s", p.Type, subjInventoryMetrics)
					}
					if errors.Is(err, tcgerr.ErrPermanent) {
						/* it looks like an issue with credentialed user
						so, wait for configuration update */
						_ = service.StopTransport()
					}
					return err
				},
			),
			service.makeDispatcherOption(
				fmt.Sprintf("#%s#%s#", subjDowntime, gwClientRef.HostName),
				subjDowntime,
				func(ctx context.Context, p natsPayload) error {
					var err error
					switch p.Type {
					case typeClearInDowntime:
						_, err = gwClientRef.ClearInDowntime(ctx, p.Payload)
					case typeSetInDowntime:
						_, err = gwClientRef.SetInDowntime(ctx, p.Payload)
					default:
						err = fmt.Errorf("dispatcher error on process payload type %s:%s", p.Type, subjDowntime)
					}
					return err
				},
			),
		)
	}
	return dispatcherOptions
}

func (service *AgentService) makeDispatcherOption(durableName, subj string, subjFn func(context.Context, natsPayload) error) nats.DispatcherOption {
	return nats.DispatcherOption{
		DurableName: durableName,
		Subject:     subj,
		Handler: func(b []byte) error {
			var err error
			getCtx := func(sc apitrace.SpanContext) context.Context {
				if sc.IsValid() {
					return apitrace.ContextWithRemoteSpanContext(context.Background(), sc)
				}
				return context.Background()
			}

			p := natsPayload{SpanContext: apitrace.EmptySpanContext()}
			if err = p.UnmarshalText(b); err != nil {
				log.Warn("dispatcher error on unmarshal payload: ", err)
			}
			ctx, span := StartTraceSpan(getCtx(p.SpanContext), "services", subj)
			defer func() {
				span.SetAttributes(
					label.Int("payloadLen", len(b)),
					label.String("error", fmt.Sprint(err)),
					label.String("durableName", durableName),
				)
				span.End()
			}()

			if err = subjFn(ctx, p); err == nil {
				service.updateStats(len(p.Payload), err, p.Type, time.Now())
			}
			return err
		},
	}
}

func (service *AgentService) config(data []byte) error {
	// load general config data
	if _, err := config.GetConfig().LoadConnectorDTO(data); err != nil {
		return err
	}
	// custom connector may provide additional handler for extended fields
	service.configHandler(data)
	// notify C-API config change
	if success := service.demandConfigHandler(); !success {
		log.Warn("[Config]: DemandConfigCallback returned 'false'. Continue with previous inventory.")
	}
	// TODO: add logic to avoid processing previous inventory in case of callback fails
	// stop nats processing
	_ = service.stopTransport()
	// flush uploading telemetry and configure provider while processing stopped
	service.Mutex.Lock()
	if service.telemetryFlushHandler != nil {
		service.telemetryFlushHandler()
	}
	telemetryProvider, telemetryFlushHandler, _ := initTelemetryProvider()
	service.telemetryFlushHandler = telemetryFlushHandler
	service.telemetryProvider = telemetryProvider
	service.Mutex.Unlock()
	// start nats processing if enabled
	if service.Connector.Enabled {
		_ = service.startTransport()
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
		AckWait:             service.Connector.NatsAckWait,
		MaxInflight:         service.Connector.NatsMaxInflight,
		MaxPubAcksInflight:  service.Connector.NatsMaxPubAcksInflight,
		MaxPayload:          service.Connector.NatsMaxPayload,
		MaxPendingBytes:     service.Connector.NatsMaxPendingBytes,
		MaxPendingMsgs:      service.Connector.NatsMaxPendingMsgs,
		MonitorPort:         service.Connector.NatsMonitorPort,
		StoreDir:            service.Connector.NatsStoreDir,
		StoreType:           service.Connector.NatsStoreType,
		StoreMaxAge:         service.Connector.NatsStoreMaxAge,
		StoreMaxBytes:       service.Connector.NatsStoreMaxBytes,
		StoreBufferSize:     service.Connector.NatsStoreBufferSize,
		StoreReadBufferSize: service.Connector.NatsStoreReadBufferSize,
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
	cons := make([]*config.GWConnection, 0)
	for _, c := range config.GetConfig().GWConnections {
		if c.Enabled {
			cons = append(cons, c)
		}
	}
	if len(cons) == 0 {
		log.Warn("[StartTransport]: Empty GWConnections")
		return nil
	}
	/* Process clients */
	gwClients := make([]*clients.GWClient, len(cons))
	for i := range cons {
		gwClients[i] = &clients.GWClient{
			AppName:      service.AppName,
			AppType:      service.AppType,
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
	log.Info("[StartTransport]: Started")
	return nil
}

func (service *AgentService) stopTransport() error {
	if service.agentStatus.Transport == Stopped {
		return nil
	}
	if err := nats.StopDispatcher(); err != nil {
		return err
	}
	service.agentStatus.Transport = Stopped
	log.Info("[StopTransport]: Stopped")
	return nil
}

// mixTracerContext adds `context` field if absent
func (service *AgentService) mixTracerContext(payloadJSON []byte) ([]byte, error) {
	if !bytes.Contains(payloadJSON, []byte("\"context\":")) ||
		!bytes.Contains(payloadJSON, []byte("\"traceToken\":")) {
		ctxJSON, err := json.Marshal(service.MakeTracerContext())
		if err != nil {
			return nil, err
		}
		l := bytes.LastIndexByte(payloadJSON, byte('}'))
		var buf bytes.Buffer
		buf.Write(payloadJSON[:l])
		buf.Write([]byte(",\"context\":"))
		buf.Write(ctxJSON)
		buf.Write([]byte("}"))
		return buf.Bytes(), nil
	}
	return payloadJSON, nil
}

// fixTracerContext replaces placeholders
func (service *AgentService) fixTracerContext(payloadJSON []byte) []byte {
	return bytes.ReplaceAll(
		bytes.ReplaceAll(
			payloadJSON,
			[]byte(traceOnDemandAppType),
			[]byte(service.Connector.AppType),
		),
		[]byte(traceOnDemandAgentID),
		[]byte(service.Connector.AgentID),
	)
}

// handleExit gracefully handles syscalls
func (service AgentService) handleExit() {
	c := make(chan os.Signal, 2)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		s := <-c
		fmt.Printf("\n- Signal %s received, exiting\n", s)
		/* wrap exitHandler with recover */
		go func(fn func()) {
			defer func() {
				if err := recover(); err != nil {
					log.Error("[handleExit]", err)
				}
			}()
			fn()
		}(service.exitHandler)

		if err := service.StopController(); err != nil {
			log.Error("[handleExit]", err.Error())
		}
		if err := service.StopTransport(); err != nil {
			log.Error("[handleExit]", err.Error())
		}
		if err := service.StopNats(); err != nil {
			log.Error("[handleExit]", err.Error())
		}
		os.Exit(0)
	}()
}

// hookLogErrors collects error entries for stats
func (service AgentService) hookLogErrors(entry log.Entry) error {
	service.updateStats(0, fmt.Errorf("%s%s", entry.Context.Value(entry.Entry), entry.Message), typeUndefined, entry.Time)
	return nil
}

// initTelemetryProvider creates a new provider instance
func initTelemetryProvider() (apitrace.TracerProvider, func(), error) {
	var jaegerEndpoint jaeger.EndpointOption
	jaegerAgent := config.GetConfig().Jaegertracing.Agent
	jaegerCollector := config.GetConfig().Jaegertracing.Collector
	if (len(jaegerAgent) == 0) && (len(jaegerCollector) == 0) {
		log.Debug("telemetry not configured")
		return apitrace.NoopTracerProvider(), func() {}, nil
	}
	if len(jaegerAgent) == 0 {
		jaegerEndpoint = jaeger.WithCollectorEndpoint(jaegerCollector)
	} else {
		jaegerEndpoint = jaeger.WithAgentEndpoint(jaegerAgent)
	}

	connector := config.GetConfig().Connector
	serviceName := fmt.Sprintf("%s:%s:%s",
		connector.AppType, connector.AppName, connector.AgentID)
	tags := []label.KeyValue{
		label.String("runtime", "golang"),
	}
	for k, v := range config.GetConfig().Jaegertracing.Tags {
		tags = append(tags, label.String(k, v))
	}
	return jaeger.NewExportPipeline(
		jaegerEndpoint,
		jaeger.WithProcess(jaeger.Process{
			ServiceName: serviceName,
			Tags:        tags,
		}),
		jaeger.WithSDK(&sdktrace.Config{DefaultSampler: sdktrace.AlwaysSample()}),
	)
}

func deadlineTimerHandler(deadlineTimer *time.Timer, subject *ctrlSubj) {
	for {
		select {
		case <-deadlineTimer.C:
			log.Error("#AgentService.ctrlChan timed over:", *subject)
		}
	}
}
