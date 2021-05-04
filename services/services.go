package services

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
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

func (t payloadType) all() []string {
	return []string{
		"undefined",
		"events",
		"eventsAck",
		"eventsUnack",
		"inventory",
		"metrics",
		"clearInDowntime",
		"setInDowntime",
	}
}

func (t payloadType) String() string {
	return t.all()[t]
}

func (t *payloadType) FromString(s string) error {
	for i, v := range t.all() {
		if s == v {
			*t = payloadType(i)
			return nil
		}
	}
	return fmt.Errorf("unknown payload type")
}

type natsPayload struct {
	SpanContext trace.SpanContext

	Payload []byte
	Type    payloadType
}

// Marshal implements Marshaler
// internally it applyes the latest format version
func (p natsPayload) Marshal() ([]byte, error) {
	if b, err := p.marshalV2(); err == nil {
		return append([]byte("v2:"), b...), nil
	} else {
		return nil, err
	}
}

// Unmarshal implements Unmarshaler
// internally it applyes implementation based on format version
func (p *natsPayload) Unmarshal(input []byte) error {
	if len(input) == 0 {
		return nil
	}
	switch {
	case input[0] < 8:
		return p.unmarshalV1(input)
	case bytes.HasPrefix(input, []byte("v1:")):
		return p.unmarshalV1(input[3:])
	case bytes.HasPrefix(input, []byte("v2:")):
		return p.unmarshalV2(input[3:])
	default:
		return fmt.Errorf("unknown payload format")
	}
}

func (p natsPayload) marshalV1() ([]byte, error) {
	spanID := p.SpanContext.SpanID()
	traceID := p.SpanContext.TraceID()
	traceFlags := p.SpanContext.TraceFlags()
	var buf bytes.Buffer
	if err := buf.WriteByte(byte(p.Type)); err != nil {
		return nil, err
	}
	if _, err := buf.Write(spanID[:]); err != nil {
		return nil, err
	}
	if _, err := buf.Write(traceID[:]); err != nil {
		return nil, err
	}
	if err := buf.WriteByte(byte(traceFlags)); err != nil {
		return nil, err
	}
	if _, err := buf.Write(p.Payload[:]); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func (p *natsPayload) unmarshalV1(input []byte) error {
	/* process input bytes as:
	byte     payloadType
	[8]byte  SpanContext SpanID
	[16]byte SpanContext TraceID
	byte     SpanContext TraceFlags
	[]byte   Payload */
	var (
		spanID     [8]byte
		traceID    [16]byte
		traceFlags byte
	)
	buf := bytes.NewBuffer(input)
	p.Type = payloadType(buf.Next(1)[0])
	if _, err := buf.Read(spanID[:]); err != nil {
		return err
	}
	if _, err := buf.Read(traceID[:]); err != nil {
		return err
	}
	traceFlags = buf.Next(1)[0]
	p.Payload = buf.Bytes()
	p.SpanContext = trace.NewSpanContext(trace.SpanContextConfig{
		SpanID:     spanID,
		TraceID:    traceID,
		TraceFlags: trace.TraceFlags(traceFlags),
	})
	return nil
}

/* natsPayload2 used for json encoding
takes only simple fields from SpanContext because of
trace.SpanContextConfig doesn't support unmarshaling (otel-v.0.20.0)
trace.SpanIDFromHex and trace.TraceIDFromHex don't support zero values
suitable in case of NoopTracerProvider */
type natsPayload2 struct {
	Type    string          `json:"type"`
	Payload json.RawMessage `json:"payload"`

	SpanID     string `json:"spanID"`
	TraceID    string `json:"traceID"`
	TraceFlags uint8  `json:"traceFlags"`
}

func (p natsPayload) marshalV2() ([]byte, error) {
	spanID := p.SpanContext.SpanID()
	traceID := p.SpanContext.TraceID()
	p2 := natsPayload2{
		Type:       p.Type.String(),
		Payload:    p.Payload,
		SpanID:     hex.EncodeToString(spanID[:]),
		TraceID:    hex.EncodeToString(traceID[:]),
		TraceFlags: uint8(p.SpanContext.TraceFlags()),
	}
	return json.Marshal(p2)
}

func (p *natsPayload) unmarshalV2(input []byte) error {
	var p2 natsPayload2
	if err := json.Unmarshal(input, &p2); err != nil {
		return err
	}
	spanCtxCfg := trace.SpanContextConfig{
		TraceFlags: trace.TraceFlags(p2.TraceFlags),
	}
	if v, err := hex.DecodeString(p2.SpanID); err == nil {
		copy(spanCtxCfg.SpanID[:], v)
	} else {
		return err
	}
	if v, err := hex.DecodeString(p2.TraceID); err == nil {
		copy(spanCtxCfg.TraceID[:], v)
	} else {
		return err
	}
	var pt payloadType
	if err := pt.FromString(p2.Type); err != nil {
		return err
	}
	*p = natsPayload{
		Type:        pt,
		Payload:     p2.Payload,
		SpanContext: trace.NewSpanContext(spanCtxCfg),
	}
	return nil
}
