package services

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/json"
	"expvar"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/gwos/tcg/config"
	"github.com/gwos/tcg/logzer"
	"github.com/gwos/tcg/nats"
	"github.com/gwos/tcg/sdk/clients"
	tcgerr "github.com/gwos/tcg/sdk/errors"
	"github.com/gwos/tcg/sdk/transit"
	"github.com/gwos/tcg/taskqueue"
	"github.com/gwos/tcg/tracing"
	"github.com/hashicorp/go-uuid"
	"github.com/patrickmn/go-cache"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	tracesdk "go.opentelemetry.io/otel/sdk/trace"
)

// AgentService implements AgentServices interface
type AgentService struct {
	*config.Connector

	agentStatus *AgentStatus
	dsClient    clients.DSClient
	gwClients   []clients.GWClient
	quitChan    chan struct{}
	taskQueue   *taskqueue.TaskQueue

	tracerCache    *cache.Cache
	tracerToken    []byte                   // gw tracing
	tracerProvider *tracesdk.TracerProvider // otel tracing

	configHandler func([]byte)
	exitHandler   func()

	stats *Stats
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
		agentService = &AgentService{
			Connector: &config.GetConfig().Connector,

			agentStatus: &AgentStatus{
				Controller: xAgentStatusController,
				Transport:  xAgentStatusTransport,
				Nats:       xAgentStatusNats,
			},
			dsClient:    clients.DSClient{DSConnection: config.GetConfig().DSConnection.AsClient()},
			quitChan:    make(chan struct{}, 1),
			tracerCache: cache.New(-1, -1),

			configHandler: defaultConfigHandler,
			exitHandler:   defaultExitHandler,

			stats: &Stats{
				BytesSent:              xStatsBytesSent,
				MetricsSent:            xStatsMetricsSent,
				MessagesSent:           xStatsMessagesSent,
				ExecutionTimeInventory: xStatsExecutionTimeInventory,
				ExecutionTimeMetrics:   xStatsExecutionTimeMetrics,
				LastEventsRun:          xStatsLastEventsRun,
				LastInventoryRun:       xStatsLastInventoryRun,
				LastMetricsRun:         xStatsLastMetricsRun,
				UpSince:                xStatsUpSince,
				x:                      xStats,
			},
		}

		agentService.initTracerToken()
		agentService.initOTEL()
		agentService.initProM()
		agentService.handleTasks()
		if AllowSignalHandlers {
			agentService.hookInterrupt()
		}

		log.Debug().Func(func(e *zerolog.Event) {
			e.Strs("env", os.Environ()).
				Str("connector", fmt.Sprintf("%+v", *agentService.Connector)).
				Str("suppress", fmt.Sprintf("%+v", config.Suppress))
		}).
			Stringer("httpClientTimeout", clients.HttpClient.Timeout).
			Stringer("httpClientTimeoutGW", clients.HttpClientGW.Timeout).
			Bool("tlsClientInsecure", clients.HttpClientTransport.TLSClientConfig.InsecureSkipVerify).
			Msg("starting with config")
	})

	return agentService
}

// DemandConfig implements AgentServices.DemandConfig interface
func (service *AgentService) DemandConfig() error {
	if err := service.StartController(); err != nil {
		return err
	}
	// if config.GetConfig().IsPMC() {
	// 	log.Info().Msg("configuring PARENT_MANAGED_CHILD")
	// 	/* expect the config api call */
	// 	return nil
	// }
	if len(service.AgentID) == 0 || len(service.dsClient.HostName) == 0 {
		log.Info().Msg("config server is not configured")
		/* expect the config api call */
		return nil
	}

	go func() {
		for i := 0; ; i++ {
			if err := service.dsClient.Reload(service.AgentID); err != nil {
				log.Warn().Err(err).Msg("config server is not available, will retry")
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
func (service *AgentService) MakeTracerContext() transit.TracerContext {
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

	return transit.TracerContext{
		AgentID:    agentID,
		AppType:    appType,
		TimeStamp:  transit.NewTimestamp(),
		TraceToken: traceToken,
		Version:    transit.ModelVersion,
	}
}

func defaultConfigHandler([]byte) {}

func defaultExitHandler() {}

// Quit returns channel
// useful for main loop
func (service *AgentService) Quit() <-chan struct{} {
	return service.quitChan
}

// RegisterConfigHandler sets callback
// useful for process extensions
func (service *AgentService) RegisterConfigHandler(fn func([]byte)) {
	service.configHandler = fn
}

// RemoveConfigHandler removes callback
func (service *AgentService) RemoveConfigHandler() {
	service.configHandler = defaultConfigHandler
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
func (service *AgentService) ExitAsync() (*taskqueue.Task, error) {
	return service.taskQueue.PushAsync(taskExit)
}

// ResetNatsAsync implements AgentServices.ResetNatsAsync interface
func (service *AgentService) ResetNatsAsync() (*taskqueue.Task, error) {
	return service.taskQueue.PushAsync(taskResetNats)
}

// StartControllerAsync implements AgentServices.StartControllerAsync interface
func (service *AgentService) StartControllerAsync() (*taskqueue.Task, error) {
	return service.taskQueue.PushAsync(taskStartController)
}

// StopControllerAsync implements AgentServices.StopControllerAsync interface
func (service *AgentService) StopControllerAsync() (*taskqueue.Task, error) {
	return service.taskQueue.PushAsync(taskStopController)
}

// StartNatsAsync implements AgentServices.StartNatsAsync interface
func (service *AgentService) StartNatsAsync() (*taskqueue.Task, error) {
	return service.taskQueue.PushAsync(taskStartNats)
}

// StopNatsAsync implements AgentServices.StopNatsAsync interface
func (service *AgentService) StopNatsAsync() (*taskqueue.Task, error) {
	return service.taskQueue.PushAsync(taskStopNats)
}

// StartTransportAsync implements AgentServices.StartTransportAsync interface.
func (service *AgentService) StartTransportAsync() (*taskqueue.Task, error) {
	return service.taskQueue.PushAsync(taskStartTransport)
}

// StopTransportAsync implements AgentServices.StopTransportAsync interface
func (service *AgentService) StopTransportAsync() (*taskqueue.Task, error) {
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
		Stats:         *service.stats,
		LastErrors:    logzer.LastErrors(),
	}
}

// Status implements AgentServices.Status interface
func (service *AgentService) Status() AgentStatus {
	return *service.agentStatus
}

// handleTasks handles task queue
func (service *AgentService) handleTasks() {
	hDebug := func(tt []taskqueue.Task) {
		log.Error().
			Str("lastTasks", fmt.Sprintf("%+v", tt)).
			Msg("task queue")
	}
	hAlarm := func(task *taskqueue.Task) error {
		log.Error().Msgf("task queue timed over: %s", task.Subject)
		return nil
	}
	hTask := func(task *taskqueue.Task) error {
		service.agentStatus.task = task
		var err error

		defer func() {
			log.Debug().Err(err).Stringer("agentStatus", service.agentStatus).Msgf("task queue: done: (%v)%v", task.Idx, task.Subject)
		}()
		log.Debug().Stringer("agentStatus", service.agentStatus).Msgf("task queue: task: (%v)%v", task.Idx, task.Subject)

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

	service.taskQueue = taskqueue.NewTaskQueue(
		taskqueue.WithAlarm(taskQueueAlarm, hAlarm),
		taskqueue.WithCapacity(taskQueueCapacity),
		taskqueue.WithHandlers(map[taskqueue.Subject]taskqueue.Handler{
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
		taskqueue.WithDebugger(hDebug),
	)
}

func (service *AgentService) config(data []byte) error {
	natsChk0, err := service.Connector.Nats.Hashsum()
	if err != nil {
		log.Err(err).Msg("error getting nats config checksum")
	}

	// stop nats processing, allow nats reconfiguring
	isTransportRunning := service.agentStatus.Transport.Value() == StatusRunning
	if err := service.stopTransport(); err != nil {
		log.Err(err).Msg("error stopping transport on processing config")
	}

	// TODO: add logic to avoid processing previous inventory in case of callback fails

	// load general config data
	if _, err := config.GetConfig().LoadConnectorDTO(data); err != nil {
		log.Err(err).Msg("error on processing connector config")
		return err
	}
	service.dsClient = clients.DSClient{DSConnection: config.GetConfig().DSConnection.AsClient()}
	log.Debug().
		Str("AgentID", service.AgentID).
		Str("AppType", service.AppType).
		Str("AppName", service.AppName).
		Stringer("BatchEvents", service.BatchEvents).
		Stringer("BatchMetrics", service.BatchMetrics).
		Int("BatchMaxBytes", service.BatchMaxBytes).
		Str("ControllerAddr", service.ControllerAddr).
		Str("DSClient", service.dsClient.HostName).
		Msg("loaded config")

	if !service.Connector.Enabled {
		log.Error().Msg("could not start nats dispatcher as connector is disabled")
	}

	// ensure nested services properly initialized
	GetTransitService().eventsBatcher.Reset(service.Connector.BatchEvents, service.Connector.BatchMaxBytes)
	GetTransitService().metricsBatcher.Reset(service.Connector.BatchMetrics, service.Connector.BatchMaxBytes)
	GetController().authCache.Flush()
	// flush uploading telemetry and configure provider while processing stopped
	if service.tracerProvider != nil {
		service.tracerProvider.ForceFlush(context.Background())
	}
	service.initOTEL()

	// check nats configuration
	natsChk, err := service.Connector.Nats.Hashsum()
	if err != nil {
		log.Err(err).Msg("error getting nats config checksum")
	}
	if !bytes.Equal(natsChk0, natsChk) {
		// configure nats service and start nats processing if enabled
		if err := service.stopNats(); err != nil {
			log.Err(err).Msg("error stopping nats on processing config")
		}
		if err := service.startNats(); err != nil {
			log.Err(err).Msg("error starting nats on processing config")
		}
		// config changed, so starting
		if service.Connector.Enabled {
			if err := service.startTransport(); err != nil {
				log.Err(err).Msg("error starting nats dispatcher on processing config")
			}
		}
	} else if service.Connector.Enabled && isTransportRunning {
		if err := service.startTransport(); err != nil {
			log.Err(err).Msg("error starting nats dispatcher on processing config")
		}
	}
	// custom connector may provide additional handler for extended fields
	service.configHandler(data)
	return nil
}

func (service *AgentService) exit() error {
	GetTransitService().eventsBatcher.Exit()
	GetTransitService().metricsBatcher.Exit()

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
	isNatsRunning, isTransportRunning :=
		service.agentStatus.Nats.Value() == StatusRunning,
		service.agentStatus.Transport.Value() == StatusRunning

	if err := service.stopNats(); err != nil {
		log.Warn().Err(err).Msg("could not stop nats")
	}
	if err := os.RemoveAll(filepath.Join(service.Connector.NatsStoreDir, "jetstream")); err != nil {
		log.Warn().Err(err).Msgf("could not remove nats jetstream dir")
	}
	if isNatsRunning {
		if err := service.startNats(); err != nil {
			log.Warn().Err(err).Msg("could not start nats")
		}
	}
	if isTransportRunning {
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
	if service.agentStatus.Controller.Value() == StatusStopped {
		return nil
	}
	return GetController().stopController()
}

func (service *AgentService) startNats() error {
	return nats.StartServer(nats.Config{
		AckWait:            service.Connector.NatsAckWait,
		LogColors:          service.Connector.LogColors,
		MaxInflight:        service.Connector.NatsMaxInflight,
		MaxPubAcksInflight: service.Connector.NatsMaxPubAcksInflight,
		MaxPayload:         service.Connector.NatsMaxPayload,
		MonitorPort:        service.Connector.NatsMonitorPort,
		StoreDir:           service.Connector.NatsStoreDir,
		StoreType:          service.Connector.NatsStoreType,
		StoreMaxAge:        service.Connector.NatsStoreMaxAge,
		StoreMaxBytes:      service.Connector.NatsStoreMaxBytes,
		StoreMaxMsgs:       service.Connector.NatsStoreMaxMsgs,

		ConfigFile: service.Connector.NatsServerConfigFile,
	})
}

func (service *AgentService) stopNats() error {
	if service.agentStatus.Nats.Value() == StatusStopped {
		return nil
	}

	// Stop Transport as dependency
	err := service.stopTransport()
	// skip Stop Transport error checking
	nats.StopServer()
	return err
}

func (service *AgentService) startTransport() error {
	if service.Connector.AgentID == traceOnDemandAgentID ||
		service.Connector.AppType == traceOnDemandAppType ||
		len(service.Connector.AgentID) == 0 ||
		len(service.Connector.AppType) == 0 {
		err := fmt.Errorf("%w: AppType/AgentID: %v/%v", tcgerr.ErrNotConfigured,
			service.Connector.AppType, service.Connector.AgentID)
		log.Err(err).Msg("could not start")
		if strings.EqualFold(service.Connector.AppType, "NAGIOS") {
			// prevent DataGeyser fail on restart with empty config
			return nil
		}
		return err
	}
	/* Process clients */
	gwClients := make([]clients.GWClient, 0, len(config.GetConfig().GWConnections))
	for i := range config.GetConfig().GWConnections {
		c := config.GetConfig().GWConnections[i].AsClient()
		if c.Enabled {
			gwClients = append(gwClients, clients.GWClient{
				AppName:      service.AppName,
				AppType:      service.AppType,
				GWConnection: c,
			})
		}
	}
	if len(gwClients) == 0 {
		log.Warn().Msg("empty GWConnections")
		return nil
	}
	service.gwClients = gwClients
	/* Process dispatcher */
	return nats.StartDispatcher(makeSubscriptions(service.gwClients))
}

func (service *AgentService) stopTransport() error {
	if service.agentStatus.Transport.Value() == StatusStopped {
		return nil
	}
	return nats.StopDispatcher()
}

// mixTracerContext adds `context` field if absent
func (service *AgentService) mixTracerContext(payloadJSON []byte) ([]byte, bool) {
	if bytes.Contains(payloadJSON, []byte(`"context":`)) &&
		bytes.Contains(payloadJSON, []byte(`"traceToken":`)) {
		return payloadJSON, false
	}

	tc, todoTracerCtx := service.MakeTracerContext(), false
	ctxJSON, err := json.Marshal(tc)
	if err != nil {
		log.Err(err).Msg("could not mixTracerContext")
		return payloadJSON, false
	}
	if tc.AgentID == traceOnDemandAgentID ||
		tc.AppType == traceOnDemandAppType {
		todoTracerCtx = true
	}

	l := bytes.LastIndexByte(payloadJSON, byte('}'))
	return bytes.Join([][]byte{
		payloadJSON[:l], []byte(`,"context":`), ctxJSON, []byte(`}`),
	}, []byte(``)), todoTracerCtx
}

// fixTracerContext replaces placeholders
func (service *AgentService) fixTracerContext(payloadJSON []byte) []byte {
	if service.Connector.AgentID == traceOnDemandAgentID ||
		service.Connector.AppType == traceOnDemandAppType ||
		len(service.Connector.AgentID) == 0 ||
		len(service.Connector.AppType) == 0 {
		err := fmt.Errorf("%w: AppType/AgentID: %v/%v", tcgerr.ErrNotConfigured,
			service.Connector.AppType, service.Connector.AgentID)
		log.Err(err).Msg("could not fixTracerContext")
		return payloadJSON
	}

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

// initOTEL inits open telemetry
func (service *AgentService) initOTEL() {
	if tp, err := config.GetConfig().InitTracerProvider(); err == nil {
		service.tracerProvider = tp
		otel.SetTracerProvider(tp)
		tracing.IsDebugEnabled = logzer.IsDebugEnabled
	}

	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{}, propagation.Baggage{}))

	nestedHook := clients.HookRequestContext
	clients.HookRequestContext = func(ctx context.Context, req *http.Request) (context.Context, *http.Request) {
		ctx, req = nestedHook(ctx, req)
		return tracing.HookRequestContext(ctx, req)
	}
	clients.GZip = tracing.GZip
}

// initProM inits Prometheus metrics
func (service *AgentService) initProM() {
	if !service.Connector.ExportProm {
		log.Debug().Msg("export Prometheus metrics is not configured")
		return
	}
	exports := make(map[string]*prometheus.Desc)
	expvar.Do(func(kv expvar.KeyValue) {
		exports[kv.Key] = prometheus.NewDesc("expvar_"+kv.Key, kv.Key, nil, nil)
	})
	expvarCollector := collectors.NewExpvarCollector(exports)
	if err := prometheus.Register(expvarCollector); err != nil {
		log.Warn().Err(err).Msg("could not register expvar collector")
	}
}
