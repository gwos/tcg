package services

import (
	"bytes"
	"context"
	"time"

	"github.com/gwos/tcg/milliseconds"
	"github.com/gwos/tcg/taskQueue"
	"github.com/gwos/tcg/transit"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"
)

// Define NATS subjects
// group downtime, events actions, and inventory with metrics
// as try to keep the processing order
const (
	subjDowntime         = "downtime"
	subjEvents           = "events"
	subjInventoryMetrics = "inventory-metrics"
)

// Status defines status value
type Status string

// Status
const (
	StatusProcessing Status = "processing"
	StatusRunning    Status = "running"
	StatusStopped    Status = "stopped"
	StatusUnknown    Status = "unknown"
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
	task       *taskQueue.Task
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
	StartControllerAsync() (*taskQueue.Task, error)
	StopControllerAsync() (*taskQueue.Task, error)
	StartNatsAsync() (*taskQueue.Task, error)
	StopNatsAsync() (*taskQueue.Task, error)
	StartTransportAsync() (*taskQueue.Task, error)
	StopTransportAsync() (*taskQueue.Task, error)
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
	ListMetrics() ([]byte, error)
	RegisterListMetricsHandler(func() ([]byte, error))
	RemoveListMetricsHandler()
	ClearInDowntime(context.Context, []byte) error
	SetInDowntime(context.Context, []byte) error
	SendEvents(context.Context, []byte) error
	SendEventsAck(context.Context, []byte) error
	SendEventsUnack(context.Context, []byte) error
	SendResourceWithMetrics(context.Context, []byte) error
	SynchronizeInventory(context.Context, []byte) error
}

// Controllers defines TCG Agent controllers interface
type Controllers interface {
	RegisterEntrypoints([]Entrypoint)
	RemoveEntrypoints()
}

// TraceSpan aliases trace.Span interface
type TraceSpan trace.Span

// StartTraceSpan starts a span
func StartTraceSpan(ctx context.Context, tracerName, spanName string, opts ...trace.SpanOption) (context.Context, TraceSpan) {
	return otel.GetTracerProvider().
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
	typeClearInDowntime
	typeSetInDowntime
)

func (s payloadType) String() string {
	return [...]string{
		"undefined",
		"events",
		"eventsAck",
		"eventsUnack",
		"inventory",
		"metrics",
		"clearInDowntime",
		"setInDowntime",
	}[s]
}

type natsPayload struct {
	Payload     []byte
	SpanContext trace.SpanContext
	Type        payloadType
}

// MarshalText implements json.Marshaler.
// TODO: provide format versioning
// TODO: support modern SpanContext structure
func (t natsPayload) MarshalText() ([]byte, error) {
	spanID := t.SpanContext.SpanID()
	traceID := t.SpanContext.TraceID()
	traceFlags := t.SpanContext.TraceFlags()
	var b bytes.Buffer
	if err := b.WriteByte(byte(t.Type)); err != nil {
		return nil, err
	}
	if _, err := b.Write(spanID[:]); err != nil {
		return nil, err
	}
	if _, err := b.Write(traceID[:]); err != nil {
		return nil, err
	}
	if err := b.WriteByte(byte(traceFlags)); err != nil {
		return nil, err
	}
	if _, err := b.Write(t.Payload[:]); err != nil {
		return nil, err
	}
	return b.Bytes(), nil
}

// UnmarshalText implements json.Unmarshaler.
// TODO: provide format versioning
// TODO: support modern SpanContext structure
func (t *natsPayload) UnmarshalText(input []byte) error {
	var (
		spanID     [8]byte
		traceID    [16]byte
		traceFlags byte
	)
	b := bytes.NewBuffer(input)
	t.Type = payloadType(b.Next(1)[0])
	if _, err := b.Read(spanID[:]); err != nil {
		return err
	}
	if _, err := b.Read(traceID[:]); err != nil {
		return err
	}
	traceFlags = b.Next(1)[0]
	t.Payload = b.Bytes()
	t.SpanContext = trace.NewSpanContext(trace.SpanContextConfig{
		SpanID:     spanID,
		TraceID:    traceID,
		TraceFlags: trace.TraceFlags(traceFlags),
	})
	return nil
}
