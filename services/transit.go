package services

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/gwos/tcg/batcher"
	"github.com/gwos/tcg/batcher/events"
	"github.com/gwos/tcg/batcher/metrics"
	"github.com/gwos/tcg/nats"
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

	var (
		b   []byte
		err error
	)
	ctx, span := tracing.StartTraceSpan(ctx, "services", "ClearInDowntime")
	_, span2 := tracing.StartTraceSpan(ctx, "services", "nats:publish")
	defer func() {
		tracing.EndTraceSpan(span2,
			tracing.TraceAttrError(err),
			tracing.TraceAttrPayloadDbg(b),
			tracing.TraceAttrPayloadLen(b),
		)
		tracing.EndTraceSpan(span)
	}()

	b, err = natsPayload{span.SpanContext(), payload, typeClearInDowntime}.Marshal()
	if err != nil {
		return err
	}
	err = nats.Publish(subjDowntimes, b)
	return err
}

// SetInDowntime implements TransitServices.SetInDowntime interface
func (service *TransitService) SetInDowntime(ctx context.Context, payload []byte) error {
	if service.suppressDowntimes {
		return nil
	}

	var (
		b   []byte
		err error
	)
	ctx, span := tracing.StartTraceSpan(ctx, "services", "SetInDowntime")
	_, span2 := tracing.StartTraceSpan(ctx, "services", "nats:publish")
	defer func() {
		tracing.EndTraceSpan(span2,
			tracing.TraceAttrError(err),
			tracing.TraceAttrPayloadDbg(b),
			tracing.TraceAttrPayloadLen(b),
		)
		tracing.EndTraceSpan(span)
	}()

	b, err = natsPayload{span.SpanContext(), payload, typeSetInDowntime}.Marshal()
	if err != nil {
		return err
	}
	err = nats.Publish(subjDowntimes, b)
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
	service.stats.exp.Add("sendEvents", 1)
	var (
		b   []byte
		err error
	)
	ctx, span := tracing.StartTraceSpan(ctx, "services", "SendEvents")
	_, span2 := tracing.StartTraceSpan(ctx, "services", "nats:publish")
	defer func() {
		tracing.EndTraceSpan(span2,
			tracing.TraceAttrError(err),
			tracing.TraceAttrPayloadDbg(b),
			tracing.TraceAttrPayloadLen(b),
		)
		tracing.EndTraceSpan(span)
	}()

	b, err = natsPayload{span.SpanContext(), payload, typeEvents}.Marshal()
	if err != nil {
		return err
	}
	err = nats.Publish(subjEvents, b)
	return err
}

// SendEventsAck implements TransitServices.SendEventsAck interface
func (service *TransitService) SendEventsAck(ctx context.Context, payload []byte) error {
	if service.suppressEvents {
		return nil
	}

	var (
		b   []byte
		err error
	)
	ctx, span := tracing.StartTraceSpan(ctx, "services", "SendEventsAck")
	_, span2 := tracing.StartTraceSpan(ctx, "services", "nats:publish")
	defer func() {
		tracing.EndTraceSpan(span2,
			tracing.TraceAttrError(err),
			tracing.TraceAttrPayloadDbg(b),
			tracing.TraceAttrPayloadLen(b),
		)
		tracing.EndTraceSpan(span)
	}()

	b, err = natsPayload{span.SpanContext(), payload, typeEventsAck}.Marshal()
	if err != nil {
		return err
	}
	err = nats.Publish(subjEvents, b)
	return err
}

// SendEventsUnack implements TransitServices.SendEventsUnack interface
func (service *TransitService) SendEventsUnack(ctx context.Context, payload []byte) error {
	if service.suppressEvents {
		return nil
	}

	var (
		b   []byte
		err error
	)
	ctx, span := tracing.StartTraceSpan(ctx, "services", "SendEventsUnack")
	_, span2 := tracing.StartTraceSpan(ctx, "services", "nats:publish")
	defer func() {
		tracing.EndTraceSpan(span2,
			tracing.TraceAttrError(err),
			tracing.TraceAttrPayloadDbg(b),
			tracing.TraceAttrPayloadLen(b),
		)
		tracing.EndTraceSpan(span)
	}()

	b, err = natsPayload{span.SpanContext(), payload, typeEventsUnack}.Marshal()
	if err != nil {
		return err
	}
	err = nats.Publish(subjEvents, b)
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
	service.stats.exp.Add("sendMetrics", 1)
	var (
		b   []byte
		err error
	)
	ctx, span := tracing.StartTraceSpan(ctx, "services", "SendResourceWithMetrics")
	_, span2 := tracing.StartTraceSpan(ctx, "services", "nats:publish")
	defer func() {
		tracing.EndTraceSpan(span2,
			tracing.TraceAttrError(err),
			tracing.TraceAttrPayloadDbg(b),
			tracing.TraceAttrPayloadLen(b),
		)
		tracing.EndTraceSpan(span)
	}()

	payload, err = service.mixTracerContext(payload)
	if err != nil {
		return err
	}
	b, err = natsPayload{span.SpanContext(), payload, typeMetrics}.Marshal()
	if err != nil {
		return err
	}
	err = nats.Publish(subjInventoryMetrics, b)
	return err
}

// SynchronizeInventory implements TransitServices.SynchronizeInventory interface
func (service *TransitService) SynchronizeInventory(ctx context.Context, payload []byte) error {
	if service.suppressInventory {
		return nil
	}

	service.stats.LastInventoryRun.Set(time.Now().UnixMilli())
	var (
		b   []byte
		err error
	)
	ctx, span1 := tracing.StartTraceSpan(ctx, "services", "SynchronizeInventory")
	_, span := tracing.StartTraceSpan(ctx, "services", "nats:publish")
	defer func() {
		tracing.EndTraceSpan(span,
			tracing.TraceAttrError(err),
			tracing.TraceAttrPayloadDbg(b),
			tracing.TraceAttrPayloadLen(b),
		)
		tracing.EndTraceSpan(span1,
			tracing.TraceAttrPayloadDbg(payload),
			tracing.TraceAttrPayloadLen(payload),
		)
	}()

	payload, err = service.mixTracerContext(payload)
	if err != nil {
		return err
	}
	b, err = natsPayload{span.SpanContext(), payload, typeInventory}.Marshal()
	if err != nil {
		return err
	}
	err = nats.Publish(subjInventoryMetrics, b)
	return err
}
