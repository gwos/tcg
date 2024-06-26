package services

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"time"

	"github.com/gwos/tcg/batcher"
	"github.com/gwos/tcg/batcher/events"
	"github.com/gwos/tcg/batcher/metrics"
	"github.com/gwos/tcg/sdk/clients"
	"github.com/gwos/tcg/tracing"
	"github.com/rs/zerolog/log"
)

// EnvSuppressX provides ability to globally suppress the submission of all data.
// Not for any production use-case, but strictly troubleshooting:
// this would be useful in troubleshooting to isolate synch issues to malformed perf data
// (which is a really common problem)
const (
	EnvSuppressDowntimes = "TCG_SUPPRESS_DOWNTIMES"
	EnvSuppressEvents    = "TCG_SUPPRESS_EVENTS"
	EnvSuppressInventory = "TCG_SUPPRESS_INVENTORY"
	EnvSuppressMetrics   = "TCG_SUPPRESS_METRICS"
)

// TransitService implements AgentServices, TransitServices interfaces
type TransitService struct {
	*AgentService
	listMetricsHandler func() ([]byte, error)

	eventsBatcher  *batcher.Batcher
	metricsBatcher *batcher.Batcher

	inventoryKeeper struct {
		sync.Mutex
		TickerFn
		buf []byte
		hdr []string
	}

	suppressDowntimes bool
	suppressEvents    bool
	suppressInventory bool
	suppressMetrics   bool
}

var onceTransitService sync.Once
var transitService *TransitService

// GetTransitService implements Singleton pattern
func GetTransitService() *TransitService {
	onceTransitService.Do(func() {
		transitService = &TransitService{
			AgentService:       GetAgentService(),
			listMetricsHandler: defaultListMetricsHandler,
		}
		transitService.eventsBatcher = batcher.NewBatcher(
			new(events.EventsBatchBuilder),
			transitService.sendEvents,
			transitService.Connector.BatchEvents,
			transitService.Connector.BatchMaxBytes,
		)
		transitService.metricsBatcher = batcher.NewBatcher(
			new(metrics.MetricsBatchBuilder),
			transitService.sendMetrics,
			transitService.Connector.BatchMetrics,
			transitService.Connector.BatchMaxBytes,
		)

		transitService.inventoryKeeper.TickerFn = *NewTickerFn(time.Second, func() {
			p := &transitService.inventoryKeeper
			p.Lock()
			defer p.Unlock()
			if len(p.buf) == 0 {
				return
			}
			if err := Put2Nats(context.TODO(), subjInventoryMetrics, p.buf, p.hdr...); err == nil {
				p.buf, p.hdr = nil, nil
			}
		})

		applySuppressEnv := func(b *bool, env, str string) {
			v, err := strconv.ParseBool(os.Getenv(env))
			*b = err == nil && v
			if *b {
				log.Error().Msgf("TCG will suppress %v due to %v env var is active", str, env)
			}
		}
		applySuppressEnv(&transitService.suppressDowntimes, EnvSuppressDowntimes, "Downtimes")
		applySuppressEnv(&transitService.suppressEvents, EnvSuppressEvents, "Events")
		applySuppressEnv(&transitService.suppressInventory, EnvSuppressInventory, "Inventory")
		applySuppressEnv(&transitService.suppressMetrics, EnvSuppressMetrics, "Metrics")
	})
	return transitService
}

func defaultListMetricsHandler() ([]byte, error) {
	return nil, fmt.Errorf("listMetricsHandler unavailable")
}

// ListMetrics implements TransitServices.ListMetrics interface
func (service *TransitService) ListMetrics() ([]byte, error) {
	return service.listMetricsHandler()
}

// RegisterListMetricsHandler implements TransitServices.RegisterListMetricsHandler interface
func (service *TransitService) RegisterListMetricsHandler(handler func() ([]byte, error)) {
	service.listMetricsHandler = handler
}

// RemoveListMetricsHandler implements TransitServices.RemoveListMetricsHandler interface
func (service *TransitService) RemoveListMetricsHandler() {
	service.listMetricsHandler = defaultListMetricsHandler
}

// ClearInDowntime implements TransitServices.ClearInDowntime interface
func (service *TransitService) ClearInDowntime(ctx context.Context, payload []byte) error {
	if service.suppressDowntimes {
		return nil
	}

	ctx, span := tracing.StartTraceSpan(ctx, "services", "ClearInDowntime")
	var err error
	defer func() {
		tracing.EndTraceSpan(span,
			tracing.TraceAttrError(err),
			tracing.TraceAttrPayloadDbg(payload),
			tracing.TraceAttrPayloadLen(payload),
		)
		if err != nil {
			log.Err(err).Msg("ClearInDowntime failed")
		}
	}()

	err = Put2Nats(ctx, subjDowntimes, payload,
		clients.HdrPayloadType, typeClearInDowntime.String())
	return err
}

// SetInDowntime implements TransitServices.SetInDowntime interface
func (service *TransitService) SetInDowntime(ctx context.Context, payload []byte) error {
	if service.suppressDowntimes {
		return nil
	}

	ctx, span := tracing.StartTraceSpan(ctx, "services", "SetInDowntime")
	var err error
	defer func() {
		tracing.EndTraceSpan(span,
			tracing.TraceAttrError(err),
			tracing.TraceAttrPayloadDbg(payload),
			tracing.TraceAttrPayloadLen(payload),
		)
		if err != nil {
			log.Err(err).Msg("ClearInDowntime failed")
		}
	}()

	err = Put2Nats(ctx, subjDowntimes, payload,
		clients.HdrPayloadType, typeSetInDowntime.String())
	return err
}

// SendEvents implements TransitServices.SendEvents interface
func (service *TransitService) SendEvents(ctx context.Context, payload []byte) error {
	if service.suppressEvents {
		return nil
	}

	service.stats.LastEventsRun.Set(time.Now().UnixMilli())
	if service.Connector.BatchEvents == 0 {
		return service.sendEvents(ctx, payload)
	}
	service.eventsBatcher.Add(payload)
	return nil
}

func (service *TransitService) sendEvents(ctx context.Context, payload []byte) error {
	service.stats.x.Add("sendEvents", 1)

	ctx, span := tracing.StartTraceSpan(ctx, "services", "SendEvents")
	var err error
	defer func() {
		tracing.EndTraceSpan(span,
			tracing.TraceAttrError(err),
			tracing.TraceAttrPayloadDbg(payload),
			tracing.TraceAttrPayloadLen(payload),
		)
		if err != nil {
			log.Err(err).Msg("sendEvents failed")
		}
	}()

	err = Put2Nats(ctx, subjEvents, payload,
		clients.HdrPayloadType, typeEvents.String())
	return err
}

// SendEventsAck implements TransitServices.SendEventsAck interface
func (service *TransitService) SendEventsAck(ctx context.Context, payload []byte) error {
	if service.suppressEvents {
		return nil
	}

	ctx, span := tracing.StartTraceSpan(ctx, "services", "SendEventsAck")
	var err error
	defer func() {
		tracing.EndTraceSpan(span,
			tracing.TraceAttrError(err),
			tracing.TraceAttrPayloadDbg(payload),
			tracing.TraceAttrPayloadLen(payload),
		)
		if err != nil {
			log.Err(err).Msg("SendEventsAck failed")
		}
	}()

	err = Put2Nats(ctx, subjEvents, payload,
		clients.HdrPayloadType, typeEventsAck.String())
	return err
}

// SendEventsUnack implements TransitServices.SendEventsUnack interface
func (service *TransitService) SendEventsUnack(ctx context.Context, payload []byte) error {
	if service.suppressEvents {
		return nil
	}

	ctx, span := tracing.StartTraceSpan(ctx, "services", "SendEventsUnack")
	var err error
	defer func() {
		tracing.EndTraceSpan(span,
			tracing.TraceAttrError(err),
			tracing.TraceAttrPayloadDbg(payload),
			tracing.TraceAttrPayloadLen(payload),
		)
		if err != nil {
			log.Err(err).Msg("SendEventsUnack failed")
		}
	}()

	err = Put2Nats(ctx, subjEvents, payload,
		clients.HdrPayloadType, typeEventsUnack.String())
	return err
}

// SendResourceWithMetrics implements TransitServices.SendResourceWithMetrics interface
func (service *TransitService) SendResourceWithMetrics(ctx context.Context, payload []byte) error {
	if service.suppressMetrics {
		return nil
	}

	service.stats.LastMetricsRun.Set(time.Now().UnixMilli())
	if service.Connector.BatchMetrics == 0 {
		return service.sendMetrics(ctx, payload)
	}
	service.metricsBatcher.Add(payload)
	return nil
}

func (service *TransitService) sendMetrics(ctx context.Context, payload []byte) error {
	service.stats.x.Add("sendMetrics", 1)

	ctx, span := tracing.StartTraceSpan(ctx, "services", "SendResourceWithMetrics")
	var err error
	defer func() {
		tracing.EndTraceSpan(span,
			tracing.TraceAttrError(err),
			tracing.TraceAttrPayloadDbg(payload),
			tracing.TraceAttrPayloadLen(payload),
		)
		if err != nil {
			log.Err(err).Msg("sendMetrics failed")
		}
	}()

	payload, todoTracerCtx := service.mixTracerContext(payload)
	headers := []string{clients.HdrPayloadType, typeMetrics.String()}
	if todoTracerCtx {
		headers = append(headers, clients.HdrTodoTracerCtx)
	}
	err = Put2Nats(ctx, subjInventoryMetrics, payload,
		headers...)
	return err

	// b, err = natsPayload{span.SpanContext(), payload, typeMetrics}.Marshal()
	// if err != nil {
	// 	return err
	// }
	// err = nats.Publish(subjInventoryMetrics, b)
	// return err
}

// SynchronizeInventory implements TransitServices.SynchronizeInventory interface
func (service *TransitService) SynchronizeInventory(ctx context.Context, payload []byte) error {
	if service.suppressInventory {
		return nil
	}
	service.stats.LastInventoryRun.Set(time.Now().UnixMilli())

	_, span := tracing.StartTraceSpan(ctx, "services", "SynchronizeInventory")
	var err error
	defer func() {
		tracing.EndTraceSpan(span,
			tracing.TraceAttrError(err),
			tracing.TraceAttrPayloadDbg(payload),
			tracing.TraceAttrPayloadLen(payload),
		)
		if err != nil {
			log.Err(err).Msg("SynchronizeInventory failed")
		}
	}()

	payload, todoTracerCtx := service.mixTracerContext(payload)
	headers := []string{clients.HdrPayloadType, typeInventory.String()}
	if todoTracerCtx {
		headers = append(headers, clients.HdrTodoTracerCtx)
	}
	// Note. There is a corner case when Nats is not ready
	// We can buffer inventory and send when ready
	// err = nats.Publish(subjInventoryMetrics, b)
	// return err
	func(payload []byte, headers []string) {
		service.inventoryKeeper.Lock()
		defer service.inventoryKeeper.Unlock()
		service.inventoryKeeper.buf, service.inventoryKeeper.hdr = payload, headers

		f0 := filepath.Join(service.NatsStoreDir, "inventory.json")
		f1 := filepath.Join(service.NatsStoreDir, "inventory1.json")
		_, _ = os.MkdirAll(service.NatsStoreDir, 0777), os.Rename(f0, f1)
		if err := os.WriteFile(f0, payload, 0666); err != nil {
			log.Err(err).Msg("could not store inventory file")
		}
	}(payload, headers)
	return nil
}

// TickerFn is wrapper for time.Ticker
type TickerFn struct {
	time.Ticker
	Stop func()
	done chan bool
}

func NewTickerFn(d time.Duration, fn func()) *TickerFn {
	ticker := new(TickerFn)
	ticker.Ticker = *time.NewTicker(d)
	ticker.done = make(chan bool)
	ticker.Stop = func() {
		ticker.Ticker.Stop()
		ticker.done <- true
	}
	go func() {
		for {
			select {
			case <-ticker.done:
				return
			case <-ticker.C:
				fn()
			}
		}
	}()
	return ticker
}
