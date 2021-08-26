package batcher

import (
	"context"
	"math"
	"sync"
	"time"

	"github.com/rs/zerolog/log"
)

// BatchBuilder defines builder interface
type BatchBuilder interface {
	// Build builds the batch payloads
	// it's possible that not all input payloads can be combined into one
	Build([][]byte) [][]byte
}

// BatchHandler defines handler
type BatchHandler func(context.Context, []byte) error

// Batcher implements buffered batcher
type Batcher struct {
	mu sync.Mutex

	buf        [][]byte
	bufSize    int
	maxBytes   int
	ticker     *time.Ticker
	tickerExit chan bool

	builder BatchBuilder
	handler BatchHandler
}

// NewBatcher returns new instance
func NewBatcher(
	bb BatchBuilder,
	bh BatchHandler,
	d time.Duration,
	maxBytes int) *Batcher {
	if d == 0 {
		d = math.MaxInt64
	}
	bt := Batcher{
		buf:        make([][]byte, 0),
		bufSize:    0,
		maxBytes:   maxBytes,
		ticker:     time.NewTicker(d),
		tickerExit: make(chan bool, 1),

		builder: bb,
		handler: bh,
	}

	/* handle ticker */
	go func() {
		for {
			select {
			case <-bt.ticker.C:
				bt.Batch()
			case <-bt.tickerExit:
				bt.Batch()
				return
			}
		}
	}()

	return &bt
}

// Add adds single payload to batch buffer
func (bt *Batcher) Add(p []byte) {
	bt.mu.Lock()

	bt.buf = append(bt.buf, p)
	bt.bufSize += len(p)

	bt.mu.Unlock()

	bt.bufSize += len(p)
	if bt.bufSize > bt.maxBytes {
		log.Debug().Msgf("batch buffer size %dKB exceeded the soft limit %dKB",
			bt.bufSize/1024, bt.maxBytes/1024)
		bt.Batch()
	}
}

// Batch processes buffered payloads
func (bt *Batcher) Batch() {
	bt.mu.Lock()

	buf := bt.buf
	bt.buf = make([][]byte, 0)
	bt.bufSize = 0

	bt.mu.Unlock()

	payloads := bt.builder.Build(buf)
	if len(payloads) > 0 {
		for _, p := range payloads {
			if len(p) > 0 {
				bt.handler(context.Background(), p)
			}
		}
	}
}

// Exit stops the internal ticker
func (bt *Batcher) Exit() {
	bt.tickerExit <- true
}

// Reset applyes configuration
func (bt *Batcher) Reset(d time.Duration, maxBytes int) {
	bt.Batch()
	bt.maxBytes = maxBytes
	if d == 0 {
		d = math.MaxInt64
	}
	bt.ticker.Reset(d)
}
