package services

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/gwos/tcg/batcher"
	"github.com/gwos/tcg/batcher/events"
	"github.com/gwos/tcg/batcher/metrics"
	"github.com/gwos/tcg/config"
	"github.com/gwos/tcg/sdk/clients"
	"github.com/gwos/tcg/tracing"
	"github.com/rs/zerolog/log"
)

type TransitOperation string

const (
	TOpClearInDowntime TransitOperation = "downtime-clear"
	TOpSetInDowntime   TransitOperation = "downtime-set"
	TOpSendEvents      TransitOperation = "events"
	TOpSendEventsAck   TransitOperation = "events-ack"
	TOpSendEventsUnack TransitOperation = "events-unack"
	TOpSendMetrics     TransitOperation = "metrics"
	TOpSyncInventory   TransitOperation = "inventory"
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

func (service *TransitService) exportTransit(op TransitOperation, payload []byte) error {
	if len(service.Connector.ExportTransitDir) == 0 {
		return nil
	}
	if err := os.MkdirAll(service.Connector.ExportTransitDir, 0777); err != nil {
		log.Err(err).Msg("exportTransit failed")
		return err
	}
	if err := os.WriteFile(
		filepath.Join(service.Connector.ExportTransitDir,
			time.Now().UTC().Format(time.RFC3339Nano)+"-"+string(op)+".json"),
		payload, 0664,
	); err != nil {
		log.Err(err).Msg("exportTransit failed")
		return err
	}
	return nil
}

// ClearInDowntime implements TransitServices.ClearInDowntime interface
func (service *TransitService) ClearInDowntime(ctx context.Context, payload []byte) error {
	ctx, span := tracing.StartTraceSpan(ctx, "services", string(TOpClearInDowntime))
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

	_ = service.exportTransit(TOpClearInDowntime, payload)

	if config.Suppress.Downtimes {
		tracing.TraceAttrStr("suppress", "downtimes")(span)
		return nil
	}

	err = Put2Nats(ctx, subjDowntimes, payload,
		clients.HdrPayloadType, typeClearInDowntime.String())
	return err
}

// SetInDowntime implements TransitServices.SetInDowntime interface
func (service *TransitService) SetInDowntime(ctx context.Context, payload []byte) error {
	ctx, span := tracing.StartTraceSpan(ctx, "services", string(TOpSetInDowntime))
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

	_ = service.exportTransit(TOpSetInDowntime, payload)

	if config.Suppress.Downtimes {
		tracing.TraceAttrStr("suppress", "downtimes")(span)
		return nil
	}

	err = Put2Nats(ctx, subjDowntimes, payload,
		clients.HdrPayloadType, typeSetInDowntime.String())
	return err
}

// SendEvents implements TransitServices.SendEvents interface
func (service *TransitService) SendEvents(ctx context.Context, payload []byte) error {
	_ = service.exportTransit(TOpSendEvents, payload)

	if config.Suppress.Events {
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

	ctx, span := tracing.StartTraceSpan(ctx, "services", string(TOpSendEvents))
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
	ctx, span := tracing.StartTraceSpan(ctx, "services", string(TOpSendEventsAck))
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

	_ = service.exportTransit(TOpSendEventsAck, payload)

	if config.Suppress.Events {
		tracing.TraceAttrStr("suppress", "events")(span)
		return nil
	}

	err = Put2Nats(ctx, subjEvents, payload,
		clients.HdrPayloadType, typeEventsAck.String())
	return err
}

// SendEventsUnack implements TransitServices.SendEventsUnack interface
func (service *TransitService) SendEventsUnack(ctx context.Context, payload []byte) error {
	ctx, span := tracing.StartTraceSpan(ctx, "services", string(TOpSendEventsUnack))
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

	_ = service.exportTransit(TOpSendEventsUnack, payload)

	if config.Suppress.Events {
		tracing.TraceAttrStr("suppress", "events")(span)
		return nil
	}

	err = Put2Nats(ctx, subjEvents, payload,
		clients.HdrPayloadType, typeEventsUnack.String())
	return err
}

// SendResourceWithMetrics implements TransitServices.SendResourceWithMetrics interface
func (service *TransitService) SendResourceWithMetrics(ctx context.Context, payload []byte) error {
	_ = service.exportTransit(TOpSendMetrics, payload)

	if config.Suppress.Metrics {
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

	ctx, span := tracing.StartTraceSpan(ctx, "services", string(TOpSendMetrics))
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
	_, span := tracing.StartTraceSpan(ctx, "services", string(TOpSyncInventory))
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

	_ = service.exportTransit(TOpSyncInventory, payload)

	if config.Suppress.Inventory {
		tracing.TraceAttrStr("suppress", "inventory")(span)
		return nil
	}

	service.stats.LastInventoryRun.Set(time.Now().UnixMilli())

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
