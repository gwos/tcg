package services

import (
	"context"
	"fmt"
	"sync"

	"github.com/gwos/tcg/nats"
	"go.opentelemetry.io/otel/label"
)

// TransitService implements AgentServices, TransitServices interfaces
type TransitService struct {
	*AgentService
}

var onceTransitService sync.Once
var transitService *TransitService

// GetTransitService implements Singleton pattern
func GetTransitService() *TransitService {
	onceTransitService.Do(func() {
		transitService = &TransitService{GetAgentService()}
	})
	return transitService
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
			label.Int("payloadLen", len(b)),
			label.String("error", fmt.Sprint(err)),
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
			label.Int("payloadLen", len(b)),
			label.String("error", fmt.Sprint(err)),
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
