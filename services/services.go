package services

import (
	"time"

	"github.com/gwos/tng/transit"
)

// Services defines TNG Agent services interface
type Services interface {
	SendResourceWithMetrics(request []byte) error
	SynchronizeInventory(request []byte) error
	StartNATS() error
	StopNATS()
	StartTransport()
	StopTransport() error
}

// AgentStats defines TNG Agent statistics
type AgentStats struct {
	AgentID                string
	AppType                string
	BytesSent              int
	MetricsSent            int
	MessagesSent           int
	LastInventoryRun       transit.MillisecondTimestamp
	LastMetricsRun         transit.MillisecondTimestamp
	ExecutionTimeInventory time.Duration
	ExecutionTimeMetrics   time.Duration
	UpSince                transit.MillisecondTimestamp
	LastError              string
}
