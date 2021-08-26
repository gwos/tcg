package services

import (
	"context"
	"fmt"
	"sync"

	"github.com/gwos/tcg/nats"
	"github.com/gwos/tcg/services/batcher"
	"github.com/gwos/tcg/services/batcher/events"
	"github.com/gwos/tcg/services/batcher/metrics"
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
			func() batcher.BatchBuilder { return events.NewEventsBatchBuilder() },
			transitService.sendEvents,
			transitService.Connector.BatchEvents,
			transitService.Connector.BatchMaxBytes,
		)
		transitService.metricsBatcher = batcher.NewBatcher(
			func() batcher.BatchBuilder { return metrics.NewMetricsBatchBuilder() },
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

// ClearInDowntime implements TransitServices.ClearInDowntime interface
func (service *TransitService) ClearInDowntime(ctx context.Context, payload []byte) error {
	var (
		b   []byte
		err error
	)
	_, span := StartTraceSpan(ctx, "services", "ClearInDowntime")
	defer func() {
		EndTraceSpan(span,
			TraceAttrError(err),
			TraceAttrPayloadLen(b),
		)
	}()

	b, err = natsPayload{span.SpanContext(), payload, typeClearInDowntime}.Marshal()
	if err != nil {
		return err
	}
	err = nats.Publish(subjDowntime, b)
	return err
}

// SetInDowntime implements TransitServices.SetInDowntime interface
func (service *TransitService) SetInDowntime(ctx context.Context, payload []byte) error {
	var (
		b   []byte
		err error
	)
	_, span := StartTraceSpan(ctx, "services", "SetInDowntime")
	defer func() {
		EndTraceSpan(span,
			TraceAttrError(err),
			TraceAttrPayloadLen(b),
		)
	}()

	b, err = natsPayload{span.SpanContext(), payload, typeSetInDowntime}.Marshal()
	if err != nil {
		return err
	}
	err = nats.Publish(subjDowntime, b)
	return err
}

// SendEvents implements TransitServices.SendEvents interface
func (service *TransitService) SendEvents(ctx context.Context, payload []byte) error {
	if service.Connector.BatchEvents == 0 {
		return service.sendEvents(ctx, payload)
	}
	return service.eventsBatcher.Add(payload)
}

func (service *TransitService) sendEvents(ctx context.Context, payload []byte) error {
	var (
		b   []byte
		err error
	)
	_, span := StartTraceSpan(ctx, "services", "SendEvents")
	defer func() {
		EndTraceSpan(span,
			TraceAttrError(err),
			TraceAttrPayloadLen(b),
		)
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
	var (
		b   []byte
		err error
	)
	_, span := StartTraceSpan(ctx, "services", "SendEventsAck")
	defer func() {
		EndTraceSpan(span,
			TraceAttrError(err),
			TraceAttrPayloadLen(b),
		)
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
	var (
		b   []byte
		err error
	)
	_, span := StartTraceSpan(ctx, "services", "SendEventsUnack")
	defer func() {
		EndTraceSpan(span,
			TraceAttrError(err),
			TraceAttrPayloadLen(b),
		)
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
	if service.Connector.BatchMetrics == 0 {
		return service.sendMetrics(ctx, payload)
	}
	return service.metricsBatcher.Add(payload)
}

func (service *TransitService) sendMetrics(ctx context.Context, payload []byte) error {
	var (
		b   []byte
		err error
	)
	_, span := StartTraceSpan(ctx, "services", "SendResourceWithMetrics")
	defer func() {
		EndTraceSpan(span,
			TraceAttrError(err),
			TraceAttrPayloadLen(b),
		)
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
	var (
		b   []byte
		err error
	)
	_, span := StartTraceSpan(ctx, "services", "SynchronizeInventory")
	defer func() {
		EndTraceSpan(span,
			TraceAttrError(err),
			TraceAttrPayloadLen(b),
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
