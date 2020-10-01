package services

import (
	"bytes"
	"context"
	"time"

	"github.com/gwos/tcg/milliseconds"
	"github.com/gwos/tcg/transit"
	"go.opentelemetry.io/otel/api/trace"
)

// Define NATS subjects
// group events actions and inventory with metrics
// as try to keep the processing order
const (
	subjEvents           = "events"
	subjInventoryMetrics = "inventory-metrics"
)

// Status defines status value
type Status string

// Status
const (
	Processing Status = "processing"
	Running           = "running"
	Stopped           = "stopped"
	Unknown           = "unknown"
)

// AgentStats defines TCG Agent statistics
type AgentStats struct {
	BytesSent              int                                `json:"bytesSent"`
	MetricsSent            int                                `json:"metricsSent"`
	MessagesSent           int                                `json:"messagesSent"`
	LastInventoryRun       *milliseconds.MillisecondTimestamp `json:"lastInventoryRun,omitempty"`
	LastMetricsRun         *milliseconds.MillisecondTimestamp `json:"lastMetricsRun,omitempty"`
	LastAlertRun           *milliseconds.MillisecondTimestamp `json:"lastAlertRun,omitempty"`
	ExecutionTimeInventory time.Duration                      `json:"executionTimeInventory"`
	ExecutionTimeMetrics   time.Duration                      `json:"executionTimeMetrics"`
	UpSince                *milliseconds.MillisecondTimestamp `json:"upSince"`
	LastErrors             []LastError                        `json:"lastErrors"`
}

// LastError defines
type LastError struct {
	Message string                             `json:"message"`
	Time    *milliseconds.MillisecondTimestamp `json:"time"`
}

// AgentIdentity defines TCG Agent Identity
type AgentIdentity struct {
	AgentID string `json:"agentID"`
	AppName string `json:"appName"`
	AppType string `json:"appType"`
}

// AgentIdentityStats defines complex type
type AgentIdentityStats struct {
	AgentIdentity
	AgentStats
}

// AgentStatus defines TCG Agent status
type AgentStatus struct {
	Ctrl       *CtrlAction
	Controller Status
	Nats       Status
	Transport  Status
}

// ConnectorStatusDTO describes status
type ConnectorStatusDTO struct {
	Status Status `json:"connectorStatus"`
	JobID  uint8  `json:"jobId,omitempty"`
}

// AgentServices defines TCG Agent services interface
type AgentServices interface {
	DemandConfig() error
	MakeTracerContext() *transit.TracerContext
	RegisterConfigHandler(func([]byte))
	RemoveConfigHandler()
	RegisterDemandConfigHandler(func() bool)
	RemoveDemandConfigHandler()
	RegisterExitHandler(func())
	RemoveExitHandler()
	StartControllerAsync(chan error) (*CtrlAction, error)
	StopControllerAsync(chan error) (*CtrlAction, error)
	StartNatsAsync(chan error) (*CtrlAction, error)
	StopNatsAsync(chan error) (*CtrlAction, error)
	StartTransportAsync(chan error) (*CtrlAction, error)
	StopTransportAsync(chan error) (*CtrlAction, error)
	StartController() error
	StopController() error
	StartNats() error
	StopNats() error
	StartTransport() error
	StopTransport() error
	Stats() AgentStats
	Status() AgentStatus
}

// TransitServices defines TCG Agent services interface
type TransitServices interface {
	SendResourceWithMetrics(context.Context, []byte) error
	SynchronizeInventory(context.Context, []byte) error
}

// GetBytesHandlerType defines handler type
type GetBytesHandlerType func() ([]byte, error)

// Controllers defines TCG Agent controllers interface
type Controllers interface {
	ListMetrics() ([]byte, error)
	RegisterEntrypoints([]Entrypoint)
	RemoveEntrypoints()
	RegisterListMetricsHandler(GetBytesHandlerType)
	RemoveListMetricsHandler()
	SendEvents(context.Context, []byte) error
	SendEventsAck(context.Context, []byte) error
	SendEventsUnack(context.Context, []byte) error
}

// TraceSpan aliases trace.Span interface
type TraceSpan trace.Span

// StartTraceSpan starts a span
func StartTraceSpan(ctx context.Context, tracerName, spanName string, opts ...trace.StartOption) (context.Context, TraceSpan) {
	if ctx == nil {
		ctx = context.Background()
	}
	return GetAgentService().TelemetryProvider.
		Tracer(tracerName).Start(ctx, spanName, opts...)
}

type payloadType byte

const (
	typeUndefined payloadType = iota
	typeEvents
	typeEventsAck
	typeEventsUnack
	typeInventory
	typeMetrics
)

func (s payloadType) String() string {
	return [...]string{
		"undefined",
		"events",
		"eventsAck",
		"eventsUnack",
		"inventory",
		"metrics",
	}[s]
}

type natsPayload struct {
	Payload     []byte
	SpanContext trace.SpanContext
	Type        payloadType
}

// MarshalText implements json.Marshaler.
func (t natsPayload) MarshalText() ([]byte, error) {
	var b bytes.Buffer
	if err := b.WriteByte(byte(t.Type)); err != nil {
		return nil, err
	}
	if _, err := b.Write(t.SpanContext.SpanID[:]); err != nil {
		return nil, err
	}
	if _, err := b.Write(t.SpanContext.TraceID[:]); err != nil {
		return nil, err
	}
	if err := b.WriteByte(byte(t.SpanContext.TraceFlags)); err != nil {
		return nil, err
	}
	if _, err := b.Write(t.Payload[:]); err != nil {
		return nil, err
	}
	return b.Bytes(), nil
}

// UnmarshalText implements json.Unmarshaler.
func (t *natsPayload) UnmarshalText(input []byte) error {
	b := bytes.NewBuffer(input)
	t.Type = payloadType(b.Next(1)[0])
	if _, err := b.Read(t.SpanContext.SpanID[:]); err != nil {
		return err
	}
	if _, err := b.Read(t.SpanContext.TraceID[:]); err != nil {
		return err
	}
	t.SpanContext.TraceFlags = b.Next(1)[0]
	t.Payload = b.Bytes()
	return nil
}
