package services

import (
	"context"
	"fmt"
	"net/http"
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
func (service *TransitService) ClearInDowntime(ctx context.Context, payload []byte) error { // nolint:dupl
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

	if err := service.exportTransit(TOpClearInDowntime, payload); err != nil {
		log.Err(err).Msgf("could not exportTransit: %v", TOpClearInDowntime)
	}

	if config.Suppress.Downtimes {
		tracing.TraceAttrStr("suppress", "downtimes")(span)
		return nil
	}

	header := make(http.Header)
	header.Set(clients.HdrPayloadType, typeClearInDowntime.String())
	ctx = clients.CtxWithHeader(ctx, header)
	err = Put2Nats(ctx, subjDowntimes, payload)
	return err
}

// SetInDowntime implements TransitServices.SetInDowntime interface
func (service *TransitService) SetInDowntime(ctx context.Context, payload []byte) error { // nolint:dupl
	ctx, span := tracing.StartTraceSpan(ctx, "services", string(TOpSetInDowntime))
	var err error
	defer func() {
		tracing.EndTraceSpan(span,
			tracing.TraceAttrError(err),
			tracing.TraceAttrPayloadDbg(payload),
			tracing.TraceAttrPayloadLen(payload),
		)
		if err != nil {
			log.Err(err).Msg("SetInDowntime failed")
		}
	}()

	if err := service.exportTransit(TOpSetInDowntime, payload); err != nil {
		log.Err(err).Msgf("could not exportTransit: %v", TOpSetInDowntime)
	}

	if config.Suppress.Downtimes {
		tracing.TraceAttrStr("suppress", "downtimes")(span)
		return nil
	}

	header := make(http.Header)
	header.Set(clients.HdrPayloadType, typeSetInDowntime.String())
	ctx = clients.CtxWithHeader(ctx, header)
	err = Put2Nats(ctx, subjDowntimes, payload)
	return err
}

// SendEvents implements TransitServices.SendEvents interface
func (service *TransitService) SendEvents(ctx context.Context, payload []byte) error {
	if err := service.exportTransit(TOpSendEvents, payload); err != nil {
		log.Err(err).Msgf("could not exportTransit: %v", TOpSendEvents)
	}

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

	header := make(http.Header)
	header.Set(clients.HdrPayloadType, typeEvents.String())
	ctx = clients.CtxWithHeader(ctx, header)
	err = Put2Nats(ctx, subjEvents, payload)
	return err
}

// SendEventsAck implements TransitServices.SendEventsAck interface
func (service *TransitService) SendEventsAck(ctx context.Context, payload []byte) error { // nolint:dupl
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

	if err := service.exportTransit(TOpSendEventsAck, payload); err != nil {
		log.Err(err).Msgf("could not exportTransit: %v", TOpSendEventsAck)
	}

	if config.Suppress.Events {
		tracing.TraceAttrStr("suppress", "events")(span)
		return nil
	}

	header := make(http.Header)
	header.Set(clients.HdrPayloadType, typeEventsAck.String())
	ctx = clients.CtxWithHeader(ctx, header)
	err = Put2Nats(ctx, subjEvents, payload)
	return err
}

// SendEventsUnack implements TransitServices.SendEventsUnack interface
func (service *TransitService) SendEventsUnack(ctx context.Context, payload []byte) error { // nolint:dupl
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

	if err := service.exportTransit(TOpSendEventsUnack, payload); err != nil {
		log.Err(err).Msgf("could not exportTransit: %v", TOpSendEventsUnack)
	}

	if config.Suppress.Events {
		tracing.TraceAttrStr("suppress", "events")(span)
		return nil
	}

	header := make(http.Header)
	header.Set(clients.HdrPayloadType, typeEventsUnack.String())
	ctx = clients.CtxWithHeader(ctx, header)
	err = Put2Nats(ctx, subjEvents, payload)
	return err
}

// SendResourceWithMetrics implements TransitServices.SendResourceWithMetrics interface
func (service *TransitService) SendResourceWithMetrics(ctx context.Context, payload []byte) error {
	if err := service.exportTransit(TOpSendMetrics, payload); err != nil {
		log.Err(err).Msgf("could not exportTransit: %v", TOpSendMetrics)
	}

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
	header := make(http.Header)
	header.Set(clients.HdrPayloadType, typeMetrics.String())
	if todoTracerCtx {
		header.Set(clients.HdrTodoTracerCtx, "-")
	}
	ctx = clients.CtxWithHeader(ctx, header)
	err = Put2Nats(ctx, subjInventoryMetrics, payload)
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

	// Store as file 2 latest inventoryes for debug
	func(payload []byte) {
		f0 := filepath.Join(service.NatsStoreDir, "inventory.json")
		f1 := filepath.Join(service.NatsStoreDir, "inventory1.json")
		_, _ = os.MkdirAll(service.NatsStoreDir, 0777), os.Rename(f0, f1)
		if err := os.WriteFile(f0, payload, 0666); err != nil {
			log.Err(err).Msg("could not store inventory file")
		}
	}(payload)

	if err := service.exportTransit(TOpSyncInventory, payload); err != nil {
		log.Err(err).Msgf("could not exportTransit: %v", TOpSyncInventory)
	}

	if config.Suppress.Inventory {
		tracing.TraceAttrStr("suppress", "inventory")(span)
		return nil
	}

	service.stats.LastInventoryRun.Set(time.Now().UnixMilli())

	payload, todoTracerCtx := service.mixTracerContext(payload)
	header := make(http.Header)
	header.Set(clients.HdrPayloadType, typeInventory.String())
	if todoTracerCtx {
		header.Set(clients.HdrTodoTracerCtx, "-")
	}
	ctx = clients.CtxWithHeader(ctx, header)
	err = Put2Nats(ctx, subjInventoryMetrics, payload)
	return err
}
