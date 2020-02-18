package services

import (
	"github.com/gwos/tng/milliseconds"
	"github.com/gwos/tng/transit"
	"time"
)

// Define NATS subjects
const (
	SubjSendResourceWithMetrics = "send-resource-with-metrics"
	SubjSynchronizeInventory    = "synchronize-inventory"
	SubjSendEvents              = "send-events"
)

// use one queue for events, events acks and unacks
// as try to keep the processing order
const eventsAckSuffix = "#ack"
const eventsUnackSuffix = "#unack"

// Status defines status value
type Status string

// Status
const (
	Processing Status = "processing"
	Running           = "running"
	Stopped           = "stopped"
	Unknown           = "unknown"
)

// AgentStats defines TNG Agent statistics
type AgentStats struct {
	AgentID                string                             `json:"agentID"`
	AppType                string                             `json:"appType"`
	BytesSent              int                                `json:"bytesSent"`
	MetricsSent            int                                `json:"metricsSent"`
	MessagesSent           int                                `json:"messagesSent"`
	LastInventoryRun       *milliseconds.MillisecondTimestamp `json:"lastInventoryRun,omitempty"`
	LastMetricsRun         *milliseconds.MillisecondTimestamp `json:"lastMetricsRun,omitempty"`
	LastAlertRun           *milliseconds.MillisecondTimestamp `json:"lastAlertRun,omitempty"`
	ExecutionTimeInventory time.Duration                      `json:"executionTimeInventory"`
	ExecutionTimeMetrics   time.Duration                      `json:"executionTimeMetrics"`
	UpSince                *milliseconds.MillisecondTimestamp `json:"upSince"`
	LastErrors             []string                           `json:"lastErrors"`
}

// AgentStatus defines TNG Agent status
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

// AgentServices defines TNG Agent services interface
type AgentServices interface {
	MakeTracerContext() *transit.TracerContext
	ReloadAsync(chan error) (*CtrlAction, error)
	StartControllerAsync(chan error) (*CtrlAction, error)
	StopControllerAsync(chan error) (*CtrlAction, error)
	StartNatsAsync(chan error) (*CtrlAction, error)
	StopNatsAsync(chan error) (*CtrlAction, error)
	StartTransportAsync(chan error) (*CtrlAction, error)
	StopTransportAsync(chan error) (*CtrlAction, error)
	Reload() error
	StartController() error
	StopController() error
	StartNats() error
	StopNats() error
	StartTransport() error
	StopTransport() error
	Stats() AgentStats
	Status() AgentStatus
}

// TransitServices defines TNG Agent services interface
type TransitServices interface {
	SendResourceWithMetrics([]byte) error
	SynchronizeInventory([]byte) error
}

// GetBytesHandlerType defines handler type
type GetBytesHandlerType func() ([]byte, error)

// Controllers defines TNG Agent controllers interface
type Controllers interface {
	ListMetrics() ([]byte, error)
	RegisterListMetricsHandler(GetBytesHandlerType)
	RemoveListMetricsHandler()
	SendEvents([]byte) error
	SendEventsAck([]byte) error
	SendEventsUnack([]byte) error
}
