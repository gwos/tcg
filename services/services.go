package services

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"expvar"
	"fmt"
	"strconv"
	"time"

	"github.com/gwos/tcg/logzer"
	"github.com/gwos/tcg/sdk/transit"
	"github.com/gwos/tcg/taskqueue"
	"go.opentelemetry.io/otel/trace"
)

var (
	// export debug info AgentStatus
	xAgentStatusController = expvar.NewString("tcgAgentStatusController")
	xAgentStatusTransport  = expvar.NewString("tcgAgentStatusTransport")
	xAgentStatusNats       = expvar.NewString("tcgAgentStatusNats")

	// export debug info Stats
	xStatsBytesSent              = expvar.NewInt("tcgStatsBytesSent")
	xStatsMetricsSent            = expvar.NewInt("tcgStatsMetricsSent")
	xStatsMessagesSent           = expvar.NewInt("tcgStatsMessagesSent")
	xStatsExecutionTimeInventory = expvar.NewInt("tcgStatsExecutionTimeInventory")
	xStatsExecutionTimeMetrics   = expvar.NewInt("tcgStatsExecutionTimeMetrics")
	xStatsLastEventsRun          = expvar.NewInt("tcgStatsLastAlertRun")
	xStatsLastInventoryRun       = expvar.NewInt("tcgStatsLastInventoryRun")
	xStatsLastMetricsRun         = expvar.NewInt("tcgStatsLastMetricsRun")
	xStatsUpSince                = expvar.NewInt("tcgStatsUpSince")
	xStats                       = expvar.NewMap("tcgStats")
)

// Define NATS subjects
// group downtimes, events actions, and inventory with metrics
// as try to keep the processing order.
// It is recommended to keep the maximum number of tokens in your subjects
// to a reasonable value of 16 tokens max. (https://docs.nats.io/nats-concepts/subjects)
const (
	subjDowntimes        = "tcg.downtimes"
	subjEvents           = "tcg.events"
	subjInventoryMetrics = "tcg.metrics"
)

// Status defines status value
type Status string

// Status as untyped string constants to avoid conversions for expvar
const (
	StatusProcessing = "processing"
	StatusRunning    = "running"
	StatusStopped    = "stopped"
	StatusUnknown    = "unknown"
)

// Stats defines TCG statistics
// exports debug info
type Stats struct {
	BytesSent              *expvar.Int
	MetricsSent            *expvar.Int
	MessagesSent           *expvar.Int
	ExecutionTimeInventory *expvar.Int
	ExecutionTimeMetrics   *expvar.Int
	LastEventsRun          *expvar.Int
	LastInventoryRun       *expvar.Int
	LastMetricsRun         *expvar.Int
	UpSince                *expvar.Int
	// x handles different counters for debug
	x *expvar.Map
}

func NewStats() *Stats {
	p := &Stats{
		BytesSent:              xStatsBytesSent,
		MetricsSent:            xStatsMetricsSent,
		MessagesSent:           xStatsMessagesSent,
		ExecutionTimeInventory: xStatsExecutionTimeInventory,
		ExecutionTimeMetrics:   xStatsExecutionTimeMetrics,
		LastEventsRun:          xStatsLastEventsRun,
		LastInventoryRun:       xStatsLastInventoryRun,
		LastMetricsRun:         xStatsLastMetricsRun,
		UpSince:                xStatsUpSince,
		x:                      xStats,
	}
	p.LastEventsRun.Set(-1)
	p.LastInventoryRun.Set(-1)
	p.LastMetricsRun.Set(-1)
	p.UpSince.Set(time.Now().UnixMilli())
	p.x.Set("uptime", expvar.Func(func() interface{} {
		return time.Since(time.UnixMilli(p.UpSince.Value())).Round(time.Second).String()
	}))
	return p
}

func (p Stats) MarshalJSON() ([]byte, error) {
	// UpSince and Last*Run fields handle timestamps
	// in output should be presented as string of millis
	// so use Int->String conversion instead of Int->Timestamp->String
	type ExportStat struct {
		BytesSent              int64         `json:"bytesSent"`
		MetricsSent            int64         `json:"metricsSent"`
		MessagesSent           int64         `json:"messagesSent"`
		ExecutionTimeInventory time.Duration `json:"executionTimeInventory"`
		ExecutionTimeMetrics   time.Duration `json:"executionTimeMetrics"`
		LastAlertRun           string        `json:"lastAlertRun,omitempty"`
		LastInventoryRun       string        `json:"lastInventoryRun,omitempty"`
		LastMetricsRun         string        `json:"lastMetricsRun,omitempty"`
		UpSince                string        `json:"upSince"`
	}
	exp := ExportStat{
		BytesSent:              p.BytesSent.Value(),
		MetricsSent:            p.MetricsSent.Value(),
		MessagesSent:           p.MessagesSent.Value(),
		ExecutionTimeInventory: time.Duration(p.ExecutionTimeInventory.Value()),
		ExecutionTimeMetrics:   time.Duration(p.ExecutionTimeMetrics.Value()),
		UpSince:                p.UpSince.String(),
	}
	if v := p.LastEventsRun.Value(); v != -1 {
		exp.LastAlertRun = p.LastEventsRun.String()
	}
	if v := p.LastInventoryRun.Value(); v != -1 {
		exp.LastInventoryRun = p.LastInventoryRun.String()
	}
	if v := p.LastMetricsRun.Value(); v != -1 {
		exp.LastMetricsRun = p.LastMetricsRun.String()
	}
	return json.Marshal(exp)
}

// AgentStatsExt defines complex type
type AgentStatsExt struct {
	transit.AgentIdentity
	Stats
	LastErrors []logzer.LogRecord `json:"lastErrors"`
}

// MarshalJSON implements json.Marshaler interface
// handles nested structures
func (p AgentStatsExt) MarshalJSON() ([]byte, error) {
	var (
		err error
		buf []byte
		bb  []byte
	)
	if bb, err = json.Marshal(p.AgentIdentity); err != nil {
		return nil, err
	}
	buf = append(buf, bb[:len(bb)-1]...)
	buf = append(buf, ',')
	if bb, err = json.Marshal(p.Stats); err != nil {
		return nil, err
	}
	buf = append(buf, bb[1:len(bb)-1]...)
	buf = append(buf, `,"lastErrors":`...)
	if bb, err = json.Marshal(p.LastErrors); err != nil {
		return nil, err
	}
	buf = append(buf, bb...)
	buf = append(buf, '}')
	return buf, nil
}

// AgentStatus defines TCG Agent status
// exports debug info
type AgentStatus struct {
	task *taskqueue.Task

	Controller *expvar.String
	Transport  *expvar.String
	Nats       *expvar.String
}

func NewAgentStatus() *AgentStatus {
	p := &AgentStatus{
		Controller: xAgentStatusController,
		Transport:  xAgentStatusTransport,
		Nats:       xAgentStatusNats,
	}
	p.Controller.Set(StatusStopped)
	p.Transport.Set(StatusStopped)
	p.Nats.Set(StatusStopped)
	return p
}

func (p AgentStatus) String() string {
	return fmt.Sprintf("[Nats:%v Transport:%v Controller:%v]",
		p.Nats.Value(), p.Transport.Value(), p.Controller.Value())
}

// ConnectorStatusDTO describes status
type ConnectorStatusDTO struct {
	Status Status `json:"connectorStatus"`
	JobID  uint8  `json:"jobId,omitempty"`
}

// AgentServices defines TCG Agent services interface
type AgentServices interface {
	DemandConfig() error
	Quit() <-chan struct{}
	MakeTracerContext() *transit.TracerContext
	RegisterConfigHandler(func([]byte))
	RemoveConfigHandler()
	RegisterExitHandler(func())
	RemoveExitHandler()
	Stats() Stats
	Status() AgentStatus

	ExitAsync() (*taskqueue.Task, error)
	ResetNatsAsync() (*taskqueue.Task, error)
	StartControllerAsync() (*taskqueue.Task, error)
	StopControllerAsync() (*taskqueue.Task, error)
	StartNatsAsync() (*taskqueue.Task, error)
	StopNatsAsync() (*taskqueue.Task, error)
	StartTransportAsync() (*taskqueue.Task, error)
	StopTransportAsync() (*taskqueue.Task, error)

	Exit() error
	ResetNats() error
	StartController() error
	StopController() error
	StartNats() error
	StopNats() error
	StartTransport() error
	StopTransport() error
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

type payloadType byte

const (
	_ payloadType = iota
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
	return p.marshalV2()
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
	case bytes.HasPrefix(input, []byte(`{"v2":`)):
		return p.unmarshalV2(input)
	default:
		return fmt.Errorf("unknown payload format")
	}
}

func (p natsPayload) marshalV1() ([]byte, error) {
	spanID := p.SpanContext.SpanID()
	traceID := p.SpanContext.TraceID()
	traceFlags := p.SpanContext.TraceFlags()
	buf := make([]byte, 0, len(p.Payload)+26)
	buf = append(buf, byte(p.Type))
	buf = append(buf, spanID[:]...)
	buf = append(buf, traceID[:]...)
	buf = append(buf, byte(traceFlags))
	buf = append(buf, p.Payload...)
	return buf, nil
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

// natsPayload2 used for json encoding
// takes only simple fields from SpanContext because of
// trace.SpanContextConfig doesn't support unmarshaling (otel-v.0.20.0)
// trace.SpanIDFromHex and trace.TraceIDFromHex doesn't support zero values
// suitable in case of NoopTracerProvider
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
	traceFlags := p.SpanContext.TraceFlags()
	buf := make([]byte, 0, len(p.Payload)+132)
	buf = append(buf, `{"v2":{"type":"`...)
	buf = append(buf, p.Type.String()...)
	buf = append(buf, `","payload":`...)
	buf = append(buf, p.Payload...)
	buf = append(buf, `,"spanID":"`...)
	buf = append(buf, hex.EncodeToString(spanID[:])...)
	buf = append(buf, `","traceID":"`...)
	buf = append(buf, hex.EncodeToString(traceID[:])...)
	buf = append(buf, `","traceFlags":`...)
	buf = strconv.AppendUint(buf, uint64(traceFlags), 10)
	buf = append(buf, `}}`...)
	return buf, nil
}

func (p *natsPayload) unmarshalV2(input []byte) error {
	var p2 natsPayload2
	if err := json.Unmarshal(input[6:len(input)-1], &p2); err != nil {
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
