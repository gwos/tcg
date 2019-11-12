package services

import (
	"github.com/gwos/tng/milliseconds"
	"sync"
	"time"
)

// Define NATS subjects
const (
	SubjSendResourceWithMetrics = "send-resource-with-metrics"
	SubjSynchronizeInventory    = "synchronize-inventory"
)

// StatusEnum defines status value
type StatusEnum string

// Status
const (
	Pending StatusEnum = "Pending"
	Running            = "Running"
	Stopped            = "Stopped"
	Unknown            = "Unknown"
)

// AgentStats defines TNG Agent statistics
type AgentStats struct {
	AgentID                string
	AppType                string
	BytesSent              int
	MetricsSent            int
	MessagesSent           int
	LastInventoryRun       milliseconds.MillisecondTimestamp
	LastMetricsRun         milliseconds.MillisecondTimestamp
	ExecutionTimeInventory time.Duration
	ExecutionTimeMetrics   time.Duration
	UpSince                milliseconds.MillisecondTimestamp
	LastError              string
	sync.Mutex
}

// AgentStatus defines TNG Agent status
type AgentStatus struct {
	Controller StatusEnum
	NATS       StatusEnum
	Transport  StatusEnum
	sync.Mutex
}

// AgentServices defines TNG Agent services interface
type AgentServices interface {
	StartController() error
	StopController() error
	StartNATS() error
	StopNATS() error
	StartTransport() error
	StopTransport() error
	Stats() *AgentStats
	Status() *AgentStatus
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
}
