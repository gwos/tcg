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
	"path/filepath"
	"sync"
	"syscall"
	"time"

	"github.com/gwos/tcg/clients"
	"github.com/gwos/tcg/config"
	tcgerr "github.com/gwos/tcg/errors"
	"github.com/gwos/tcg/logger"
	"github.com/gwos/tcg/milliseconds"
	"github.com/gwos/tcg/nats"
	"github.com/gwos/tcg/taskQueue"
	"github.com/gwos/tcg/transit"
	"github.com/hashicorp/go-uuid"
	"github.com/patrickmn/go-cache"
	"github.com/rs/zerolog/log"
	"go.opentelemetry.io/otel"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
)

// AgentService implements AgentServices interface
type AgentService struct {
	*config.Connector

	agentStats  *AgentStats
	agentStatus *AgentStatus
	dsClient    *clients.DSClient
	gwClients   []*clients.GWClient
	quitChan    chan struct{}
	statsChan   chan statsCounter
	taskQueue   *taskQueue.TaskQueue

	tracerCache    *cache.Cache
	tracerToken    []byte                   // gw tracing
	tracerProvider *sdktrace.TracerProvider // otel tracing

	configHandler       func([]byte)
	demandConfigHandler func() bool
	exitHandler         func()
}

type statsCounter struct {
	bytesSent   int
	payloadType payloadType
	timestamp   time.Time
}

type taskSubject string

const (
	taskConfig          taskSubject = "config"
	taskExit            taskSubject = "exit"
	taskResetNats       taskSubject = "resetNats"
	taskStartController taskSubject = "startController"
	taskStopController  taskSubject = "stopController"
	taskStartNats       taskSubject = "startNats"
	taskStopNats        taskSubject = "stopNats"
	taskStartTransport  taskSubject = "startTransport"
	taskStopTransport   taskSubject = "stopTransport"
)

const (
	ckTracerToken        = "ckTraceToken"
	statsLastErrorsLim   = 10
	taskQueueAlarm       = time.Second * 9
	taskQueueCapacity    = 8
	traceOnDemandAgentID = "#traceOnDemandAgentID#"
	traceOnDemandAppType = "#traceOnDemandAppType#"
)

// AllowSignalHandlers defines setting the signal handlers
// true by default, handle signals: os.Interrupt, syscall.SIGTERM
// false on init of C-shared library libtransit
var AllowSignalHandlers = true

var onceAgentService sync.Once
var agentService *AgentService

// GetAgentService implements Singleton pattern
func GetAgentService() *AgentService {
	onceAgentService.Do(func() {
		agentConnector := config.GetConfig().Connector
		agentService = &AgentService{
			Connector: agentConnector,
			agentStats: &AgentStats{
				UpSince: &milliseconds.MillisecondTimestamp{Time: time.Now()},
			},
			agentStatus: &AgentStatus{
				Controller: StatusStopped,
				Nats:       StatusStopped,
				Transport:  StatusStopped,
			},
			dsClient:    &clients.DSClient{DSConnection: config.GetConfig().DSConnection},
			quitChan:    make(chan struct{}, 1),
			statsChan:   make(chan statsCounter),
			tracerCache: cache.New(-1, -1),

			configHandler:       defaultConfigHandler,
			demandConfigHandler: defaultDemandConfigHandler,
			exitHandler:         defaultExitHandler,
		}

		go agentService.listenStatsChan()
		agentService.initTracerToken()
		agentService.initTracerProvider()
		agentService.handleTasks()
		if AllowSignalHandlers {
			agentService.hookInterrupt()
		}

		log.Debug().
			Str("AgentID", agentService.AgentID).
			Str("AppType", agentService.AppType).
			Str("AppName", agentService.AppName).
			Str("BatchMetrics", agentService.BatchMetrics.String()).
			Str("ControllerAddr", agentService.ControllerAddr).
			Str("DSClient", agentService.dsClient.HostName).
			Msg("starting with config")
	})

	return agentService
}

// DemandConfig implements AgentServices.DemandConfig interface
func (service *AgentService) DemandConfig() error {
	if err := service.StartController(); err != nil {
		return err
	}
	if config.GetConfig().IsConfiguringPMC() {
		log.Info().Msg("configuring PARENT_MANAGED_CHILD")
		/* expect the config api call */
		return nil
	}
	if len(service.AgentID) == 0 || len(service.dsClient.HostName) == 0 {
		log.Info().Msg("config server is not configured")
		/* expect the config api call */
		return nil
	}

	go func() {
		for i := 0; ; i++ {
			if err := service.dsClient.Reload(service.AgentID); err != nil {
				log.Err(err).Msg("config server is not available")
				time.Sleep(time.Duration((i%4+1)*5) * time.Second)
				continue
			}
			break
		}
		log.Info().Msg("config server found and connected")
	}()
	return nil
}

// MakeTracerContext implements AgentServices.MakeTracerContext interface
func (service *AgentService) MakeTracerContext() *transit.TracerContext {
	/* combine TraceToken from fixed and incremental parts */
	tokenBuf := make([]byte, 16)
	copy(tokenBuf, service.tracerToken)
	if tokenInc, err := service.tracerCache.IncrementUint64(ckTracerToken, 1); err == nil {
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

// Quit returns channel
// usefull for main loop
func (service *AgentService) Quit() <-chan struct{} {
	return service.quitChan
}

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

// ExitAsync implements AgentServices.ExitAsync interface
func (service *AgentService) ExitAsync() (*taskQueue.Task, error) {
	return service.taskQueue.PushAsync(taskExit)
}

// ResetNatsAsync implements AgentServices.ResetNatsAsync interface
func (service *AgentService) ResetNatsAsync() (*taskQueue.Task, error) {
	return service.taskQueue.PushAsync(taskResetNats)
}

// StartControllerAsync implements AgentServices.StartControllerAsync interface
func (service *AgentService) StartControllerAsync() (*taskQueue.Task, error) {
	return service.taskQueue.PushAsync(taskStartController)
}

// StopControllerAsync implements AgentServices.StopControllerAsync interface
func (service *AgentService) StopControllerAsync() (*taskQueue.Task, error) {
	return service.taskQueue.PushAsync(taskStopController)
}

// StartNatsAsync implements AgentServices.StartNatsAsync interface
func (service *AgentService) StartNatsAsync() (*taskQueue.Task, error) {
	return service.taskQueue.PushAsync(taskStartNats)
}

// StopNatsAsync implements AgentServices.StopNatsAsync interface
func (service *AgentService) StopNatsAsync() (*taskQueue.Task, error) {
	return service.taskQueue.PushAsync(taskStopNats)
}

// StartTransportAsync implements AgentServices.StartTransportAsync interface.
func (service *AgentService) StartTransportAsync() (*taskQueue.Task, error) {
	return service.taskQueue.PushAsync(taskStartTransport)
}

// StopTransportAsync implements AgentServices.StopTransportAsync interface
func (service *AgentService) StopTransportAsync() (*taskQueue.Task, error) {
	return service.taskQueue.PushAsync(taskStopTransport)
}

// Exit implements AgentServices.Exit interface
func (service *AgentService) Exit() error {
	return service.taskQueue.PushSync(taskExit)
}

// ResetNats implements AgentServices.ResetNats interface
func (service *AgentService) ResetNats() error {
	return service.taskQueue.PushSync(taskResetNats)
}

// StartController implements AgentServices.StartController interface
func (service *AgentService) StartController() error {
	return service.taskQueue.PushSync(taskStartController)
}

// StopController implements AgentServices.StopController interface
func (service *AgentService) StopController() error {
	return service.taskQueue.PushSync(taskStopController)
}

// StartNats implements AgentServices.StartNats interface
func (service *AgentService) StartNats() error {
	return service.taskQueue.PushSync(taskStartNats)
}

// StopNats implements AgentServices.StopNats interface
func (service *AgentService) StopNats() error {
	return service.taskQueue.PushSync(taskStopNats)
}

// StartTransport implements AgentServices.StartTransport interface.
func (service *AgentService) StartTransport() error {
	return service.taskQueue.PushSync(taskStartTransport)
}

// StopTransport implements AgentServices.StopTransport interface
func (service *AgentService) StopTransport() error {
	return service.taskQueue.PushSync(taskStopTransport)
}

// Stats implements AgentServices.Stats interface
func (service *AgentService) Stats() AgentStatsExt {
	return AgentStatsExt{
		AgentIdentity: service.Connector.AgentIdentity,
		AgentStats:    *service.agentStats,
		LastErrors:    logger.LastErrors(),
	}
}

// Status implements AgentServices.Status interface
func (service *AgentService) Status() AgentStatus {
	return *service.agentStatus
}

// handleTasks handles task queue
func (service *AgentService) handleTasks() {
	hAlarm := func(task *taskQueue.Task) error {
		log.Error().Msgf("taskQueue timed over: %s", task.Subject)
		return nil
	}
	hTask := func(task *taskQueue.Task) error {
		log.Debug().
			Interface("Subject", task.Subject).
			Uint8("Idx", task.Idx).
			Msg("taskQueue")
		service.agentStatus.task = task
		var err error
		switch task.Subject {
		case taskConfig:
			err = service.config(task.Args[0].([]byte))
		case taskExit:
			err = service.exit()
		case taskResetNats:
			err = service.resetNats()
		case taskStartController:
			err = service.startController()
		case taskStopController:
			err = service.stopController()
		case taskStartNats:
			err = service.startNats()
		case taskStopNats:
			err = service.stopNats()
		case taskStartTransport:
			err = service.startTransport()
		case taskStopTransport:
			err = service.stopTransport()
		}
		service.agentStatus.task = nil
		return err
	}

	service.taskQueue = taskQueue.NewTaskQueue(
		taskQueue.WithAlarm(taskQueueAlarm, hAlarm),
		taskQueue.WithCapacity(taskQueueCapacity),
		taskQueue.WithHandlers(map[taskQueue.Subject]taskQueue.Handler{
			taskConfig:          hTask,
			taskExit:            hTask,
			taskResetNats:       hTask,
			taskStartController: hTask,
			taskStopController:  hTask,
			taskStartNats:       hTask,
			taskStopNats:        hTask,
			taskStartTransport:  hTask,
			taskStopTransport:   hTask,
		}),
	)
}

func (service *AgentService) listenStatsChan() {
	for {
		res := <-service.statsChan
		ts := milliseconds.MillisecondTimestamp{Time: res.timestamp}
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

func (service *AgentService) updateStats(c statsCounter) {
	service.statsChan <- c
}

func (service *AgentService) makeDispatcherOptions() []nats.DispatcherOption {
	var dispatcherOptions []nats.DispatcherOption
	for _, gwClient := range service.gwClients {
		// TODO: filter the message by rules per gwClient
		gwClient := gwClient /* hold loop var copy */
		dispatcherOptions = append(
			dispatcherOptions,
			service.makeDispatcherOption(
				fmt.Sprintf("#%s#%s#", subjDowntime, gwClient.HostName),
				subjDowntime,
				func(ctx context.Context, p natsPayload) error {
					var err error
					switch p.Type {
					case typeClearInDowntime:
						_, err = gwClient.ClearInDowntime(ctx, p.Payload)
					case typeSetInDowntime:
						_, err = gwClient.SetInDowntime(ctx, p.Payload)
					default:
						err = fmt.Errorf("%v: failed to process payload type %s:%s", nats.ErrDispatcher, p.Type, subjDowntime)
					}
					return err
				},
			),
			service.makeDispatcherOption(
				fmt.Sprintf("#%s#%s#", subjEvents, gwClient.HostName),
				subjEvents,
				func(ctx context.Context, p natsPayload) error {
					var err error
					switch p.Type {
					case typeEvents:
						_, err = gwClient.SendEvents(ctx, p.Payload)
					case typeEventsAck:
						_, err = gwClient.SendEventsAck(ctx, p.Payload)
					case typeEventsUnack:
						_, err = gwClient.SendEventsUnack(ctx, p.Payload)
					default:
						err = fmt.Errorf("%v: failed to process payload type %s:%s", nats.ErrDispatcher, p.Type, subjEvents)
					}
					return err
				},
			),
			service.makeDispatcherOption(
				fmt.Sprintf("#%s#%s#", subjInventoryMetrics, gwClient.HostName),
				subjInventoryMetrics,
				func(ctx context.Context, p natsPayload) error {
					var err error
					switch p.Type {
					case typeInventory:
						_, err = gwClient.SynchronizeInventory(ctx, service.fixTracerContext(p.Payload))
					case typeMetrics:
						_, err = gwClient.SendResourcesWithMetrics(ctx, service.fixTracerContext(p.Payload))
					default:
						err = fmt.Errorf("%v: failed to process payload type %s:%s", nats.ErrDispatcher, p.Type, subjInventoryMetrics)
					}
					return err
				},
			),
		)
	}
	return dispatcherOptions
}

func (service *AgentService) makeDispatcherOption(durableName, subj string, handler func(context.Context, natsPayload) error) nats.DispatcherOption {
	return nats.DispatcherOption{
		DurableName: durableName,
		Subject:     subj,
		Handler: func(b []byte) error {
			var err error
			getCtx := func(sc trace.SpanContext) context.Context {
				if sc.IsValid() {
					return trace.ContextWithRemoteSpanContext(context.Background(), sc)
				}
				return context.Background()
			}

			p := natsPayload{}
			if err = p.Unmarshal(b); err != nil {
				log.Warn().Err(err).Msg("could not unmarshal payload")
			}
			ctx, span := StartTraceSpan(getCtx(p.SpanContext), "services", subj)
			defer func() {
				EndTraceSpan(span,
					TraceAttrError(err),
					TraceAttrPayloadLen(b),
					TraceAttrString("durableName", durableName),
				)
			}()

			if err = handler(ctx, p); err == nil {
				service.updateStats(
					statsCounter{bytesSent: len(p.Payload), payloadType: p.Type, timestamp: time.Now()})
			}
			if errors.Is(err, tcgerr.ErrUnauthorized) {
				/* it looks like an issue with credentialed user
				so, wait for configuration update */
				log.Err(err).Msg("dispatcher got an issue with credentialed user, wait for configuration update")
				_ = service.StopTransport()
			} else if errors.Is(err, tcgerr.ErrUndecided) {
				/* it looks like an issue with data */
				log.Err(err).Msg("dispatcher got an issue with data")
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
	log.Debug().
		Str("AgentID", service.AgentID).
		Str("AppType", service.AppType).
		Str("AppName", service.AppName).
		Str("BatchMetrics", service.BatchMetrics.String()).
		Str("ControllerAddr", service.ControllerAddr).
		Str("DSClient", service.dsClient.HostName).
		Msg("loaded config")

	// ensure nested services properly initialized
	GetTransitService().batchCachedMetrics()
	batchMetrics := transitService.Connector.BatchMetrics
	if batchMetrics == 0 {
		batchMetrics = math.MaxInt64
	}
	GetTransitService().batcher.ticker.Reset(batchMetrics)
	GetController().authCache.Flush()
	// custom connector may provide additional handler for extended fields
	service.configHandler(data)
	// notify C-API config change
	if success := service.demandConfigHandler(); !success {
		log.Warn().Msg("demandConfigHandler returned 'false'. Continue with previous inventory.")
	}
	// TODO: add logic to avoid processing previous inventory in case of callback fails
	// stop nats processing
	_ = service.stopTransport()
	// flush uploading telemetry and configure provider while processing stopped
	if service.tracerProvider != nil {
		service.tracerProvider.ForceFlush(context.Background())
	}
	service.initTracerProvider()
	// start nats processing if enabled
	if service.Connector.Enabled {
		_ = service.startTransport()
	}
	return nil
}

func (service *AgentService) exit() error {
	GetTransitService().batchCachedMetrics()
	GetTransitService().batcher.done <- true

	if service.tracerProvider != nil {
		service.tracerProvider.ForceFlush(context.Background())
	}

	/* wrap exitHandler with recover */
	c := make(chan struct{}, 1)
	go func(fn func()) {
		defer func() {
			if err := recover(); err != nil {
				log.Err(err.(error)).Msg("handleExit")
			}
		}()
		fn()
		c <- struct{}{}
	}(service.exitHandler)
	/* wait for exitHandler done */
	<-c
	if err := service.stopController(); err != nil {
		log.Err(err).Msg("handleExit")
	}
	if err := service.stopTransport(); err != nil {
		log.Err(err).Msg("handleExit")
	}
	if err := service.stopNats(); err != nil {
		log.Err(err).Msg("handleExit")
	}
	/* send quit signal */
	service.quitChan <- struct{}{}
	return nil
}

func (service *AgentService) resetNats() error {
	st0 := *(service.agentStatus)
	if err := service.stopNats(); err != nil {
		log.Warn().Err(err).Msg("could not stop nats")
	}
	globs := [...]string{
		"*/msgs.*.dat",
		"*/msgs.*.idx",
		"*/subs.dat",
		"clients.dat",
		"server.dat",
	}
	for _, glob := range globs {
		files, _ := filepath.Glob(filepath.Join(service.Connector.NatsStoreDir, glob))
		for _, f := range files {
			log.Debug().Msgf("removing: %s", f)
			if err := os.Remove(f); err != nil {
				log.Warn().Msgf("could not remove: %s", f)
			}
		}
	}
	if st0.Nats == StatusRunning {
		if err := service.startNats(); err != nil {
			log.Warn().Err(err).Msg("could not start nats")
		}
	}
	if st0.Transport == StatusRunning {
		if err := service.startTransport(); err != nil {
			log.Warn().Err(err).Msg("could not start nats dispatcher")
		}
	}
	return nil
}

func (service *AgentService) startController() error {
	// NOTE: the service.agentStatus.Controller will be updated by controller itself
	return GetController().startController()
}

func (service *AgentService) stopController() error {
	// NOTE: the service.agentStatus.Controller will be updated by controller itself
	if service.agentStatus.Controller == StatusStopped {
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
		service.agentStatus.Nats = StatusRunning
	}
	return err
}

func (service *AgentService) stopNats() error {
	if service.agentStatus.Nats == StatusStopped {
		return nil
	}

	// Stop Transport as dependency
	err := service.stopTransport()
	// skip Stop Transport error checking
	nats.StopServer()
	service.agentStatus.Nats = StatusStopped
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
		log.Warn().Msg("empty GWConnections")
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
		service.agentStatus.Transport = StatusRunning
	} else {
		return sdErr
	}
	log.Info().Msg("dispatcher started")
	return nil
}

func (service *AgentService) stopTransport() error {
	if service.agentStatus.Transport == StatusStopped {
		return nil
	}
	if err := nats.StopDispatcher(); err != nil {
		return err
	}
	service.agentStatus.Transport = StatusStopped
	log.Info().Msg("dispatcher stopped")
	return nil
}

// mixTracerContext adds `context` field if absent
func (service *AgentService) mixTracerContext(payloadJSON []byte) ([]byte, error) {
	if !bytes.Contains(payloadJSON, []byte(`"context":`)) ||
		!bytes.Contains(payloadJSON, []byte(`"traceToken":`)) {
		ctxJSON, err := json.Marshal(service.MakeTracerContext())
		if err != nil {
			return nil, err
		}
		l := bytes.LastIndexByte(payloadJSON, byte('}'))
		return bytes.Join([][]byte{
			payloadJSON[:l], []byte(`,"context":`), ctxJSON, []byte(`}`),
		}, []byte(``)), nil
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

// hookInterrupt gracefully handles syscalls
func (service AgentService) hookInterrupt() {
	c := make(chan os.Signal, 2)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		s := <-c
		log.Info().Msgf("signal %s received, exiting", s)
		/* process exit with taskQueue to prevent feature tasks */
		_, _ = service.ExitAsync()
	}()
}

func (service *AgentService) initTracerToken() {
	/* prepare random tracerToken */
	tracerToken := []byte("aaaabbbbccccdddd")
	if randBuf, err := uuid.GenerateRandomBytes(16); err == nil {
		copy(tracerToken, randBuf)
	} else {
		/* fallback with multiplied timestamp */
		binary.PutVarint(tracerToken, time.Now().UnixNano())
		binary.PutVarint(tracerToken[6:], time.Now().UnixNano())
	}
	service.tracerCache.Set(ckTracerToken, uint64(1), -1)
	service.tracerToken = tracerToken
}

// initTracerProvider inits provider
func (service *AgentService) initTracerProvider() {
	if tp, err := config.GetConfig().InitTracerProvider(); err == nil {
		service.tracerProvider = tp
		otel.SetTracerProvider(tp)
	}
}
