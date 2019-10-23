package services

import (
	"github.com/gwos/tng/milliseconds"
	"sync"
	"time"
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
	LastInventoryRun       milliseconds.MillisecondTimestamp
	LastMetricsRun         milliseconds.MillisecondTimestamp
	ExecutionTimeInventory time.Duration
	ExecutionTimeMetrics   time.Duration
	UpSince                milliseconds.MillisecondTimestamp
	LastError              string
	sync.RWMutex
}
