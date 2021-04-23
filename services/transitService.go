package services

import (
	"context"
	"fmt"
	"sync"

	"github.com/gwos/tcg/nats"
	"go.opentelemetry.io/otel/attribute"
)

// TransitService implements AgentServices, TransitServices interfaces
type TransitService struct {
	*AgentService
	listMetricsHandler func() ([]byte, error)
}

var onceTransitService sync.Once
var transitService *TransitService

// GetTransitService implements Singleton pattern
func GetTransitService() *TransitService {
	onceTransitService.Do(func() {
		transitService = &TransitService{
			GetAgentService(),
			defaultListMetricsHandler,
		}
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
		span.SetAttributes(
			attribute.Int("payloadLen", len(payload)),
			attribute.String("error", fmt.Sprint(err)),
		)
		span.End()
	}()

	b, err = natsPayload{payload, span.SpanContext(), typeClearInDowntime}.MarshalText()
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
		span.SetAttributes(
			attribute.Int("payloadLen", len(payload)),
			attribute.String("error", fmt.Sprint(err)),
		)
		span.End()
	}()

	b, err = natsPayload{payload, span.SpanContext(), typeSetInDowntime}.MarshalText()
	err = nats.Publish(subjDowntime, b)
	return err
}

// SendEvents implements TransitServices.SendEvents interface
func (service *TransitService) SendEvents(ctx context.Context, payload []byte) error {
	var (
		b   []byte
		err error
	)
	_, span := StartTraceSpan(ctx, "services", "SendEvents")
	defer func() {
		span.SetAttributes(
			attribute.Int("payloadLen", len(payload)),
			attribute.String("error", fmt.Sprint(err)),
		)
		span.End()
	}()

	b, err = natsPayload{payload, span.SpanContext(), typeEvents}.MarshalText()
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
		span.SetAttributes(
			attribute.Int("payloadLen", len(payload)),
			attribute.String("error", fmt.Sprint(err)),
		)
		span.End()
	}()

	b, err = natsPayload{payload, span.SpanContext(), typeEventsAck}.MarshalText()
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
		span.SetAttributes(
			attribute.Int("payloadLen", len(payload)),
			attribute.String("error", fmt.Sprint(err)),
		)
		span.End()
	}()

	b, err = natsPayload{payload, span.SpanContext(), typeEventsUnack}.MarshalText()
	err = nats.Publish(subjEvents, b)
	return err
}

// SendResourceWithMetrics implements TransitServices.SendResourceWithMetrics interface
func (service *TransitService) SendResourceWithMetrics(ctx context.Context, payload []byte) error {
	var (
		b   []byte
		err error
	)
	_, span := StartTraceSpan(ctx, "services", "SendResourceWithMetrics")
	defer func() {
		span.SetAttributes(
			attribute.Int("payloadLen", len(b)),
			attribute.String("error", fmt.Sprint(err)),
		)
		span.End()
	}()

	payload, err = service.mixTracerContext(payload)
	if err != nil {
		return err
	}
	b, err = natsPayload{payload, span.SpanContext(), typeMetrics}.MarshalText()
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
		span.SetAttributes(
			attribute.Int("payloadLen", len(b)),
			attribute.String("error", fmt.Sprint(err)),
		)
		span.End()
	}()

	payload, err = service.mixTracerContext(payload)
	if err != nil {
		return err
	}
	b, err = natsPayload{payload, span.SpanContext(), typeInventory}.MarshalText()
	err = nats.Publish(subjInventoryMetrics, b)
	return err
}
