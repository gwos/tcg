package batcher

import (
	"bytes"
	"context"
	"expvar"
	"math"
	"reflect"
	"sync"
	"time"

	"github.com/gwos/tcg/tracing"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"go.opentelemetry.io/otel/trace"
)

var xStats = expvar.NewMap("tcgStatsBatcher")

// Sized pairs a buffered value with its size in bytes.
// The size is exact for serialized input and estimated for structures.
type Sized[T any] struct {
	Value T
	Size  int
}

// BatchBuilder defines builder interface
type BatchBuilder[T any] interface {
	// Build builds the batch payloads
	// it's possible that not all input values can be combined into one
	Build(buf []Sized[T], maxBytes int) [][]byte
}

// BatchHandler defines handler
type BatchHandler func(context.Context, []byte) error

// Batcher implements buffered batcher
type Batcher[T any] struct {
	mu sync.Mutex
	// batchMu serializes batches, so payloads are handled in the order they were added
	batchMu sync.Mutex

	buf        []Sized[T]
	bufSize    int
	maxBytes   int
	ticker     *time.Ticker
	tickerExit chan bool

	builder BatchBuilder[T]
	handler BatchHandler

	traceCtx   context.Context
	traceSpan  trace.Span
	tracerName string
	xBatchedAt *expvar.Int
}

// NewBatcher returns new instance
func NewBatcher[T any](
	bb BatchBuilder[T],
	bh BatchHandler,
	d time.Duration,
	maxBytes int) *Batcher[T] {
	if d == 0 {
		d = math.MaxInt64
	}
	bt := Batcher[T]{
		buf:        make([]Sized[T], 0),
		bufSize:    0,
		maxBytes:   maxBytes,
		ticker:     time.NewTicker(d),
		tickerExit: make(chan bool, 1),

		builder: bb,
		handler: bh,

		tracerName: "batcher:" + reflect.TypeOf(bb).String(),
		xBatchedAt: new(expvar.Int),
	}
	bt.traceCtx, bt.traceSpan = tracing.StartTraceSpan(context.Background(), bt.tracerName, "batching")
	bt.xBatchedAt.Set(-1)
	xStats.Set(bt.tracerName+":batchedAt", bt.xBatchedAt)

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

// Add adds single value to batch buffer
// the size is used for limiting the buffer and splitting batches
func (bt *Batcher[T]) Add(v T, size int) {
	bt.mu.Lock()

	_, span := tracing.StartTraceSpan(bt.traceCtx, bt.tracerName, "batcher:Add")
	log.Trace().Str("bt.tracerName", bt.tracerName).
		Int("payloadLen", size).
		Msg("Batcher.Add")

	bt.buf = append(bt.buf, Sized[T]{Value: v, Size: size})
	bt.bufSize += size
	bufSize := bt.bufSize

	tracing.EndTraceSpan(span,
		tracing.TraceAttrInt("payloadLen", size),
	)

	bt.mu.Unlock()
	if bufSize > bt.maxBytes {
		log.Trace().Str("bt.tracerName", bt.tracerName).
			Msgf("batch buffer size %dKB exceeded the soft limit %dKB",
				bufSize/1024, bt.maxBytes/1024)
		bt.Batch()
	}
}

// Batch processes buffered payloads
func (bt *Batcher[T]) Batch() {
	bt.batchMu.Lock()
	defer bt.batchMu.Unlock()

	bt.xBatchedAt.Set(time.Now().UnixMilli())
	bt.mu.Lock()

	buf, bufSize := bt.buf, bt.bufSize
	bt.buf, bt.bufSize = make([]Sized[T], 0), 0

	bt.mu.Unlock()
	if len(buf) > 0 {
		func() {
			/* wrap into closure for simple defer,
			cannot use services package due to import cycle */
			ctx, span := tracing.StartTraceSpan(bt.traceCtx, bt.tracerName, "batcher:Batch")
			var payloads [][]byte
			defer func() {
				tracing.EndTraceSpan(span,
					tracing.TraceAttrInt("maxBytes", bt.maxBytes),
					tracing.TraceAttrInt("bufferLen", len(buf)),
					tracing.TraceAttrInt("bufferSize", bufSize),
					tracing.TraceAttrInt("outputLen", len(payloads)),
					tracing.TraceAttrFnDbg("output", func() string { return string(bytes.Join(payloads, []byte("\n"))) }),
				)
				tracing.EndTraceSpan(bt.traceSpan)
				bt.traceCtx, bt.traceSpan = tracing.StartTraceSpan(context.Background(), bt.tracerName, "batching")
			}()

			payloads = bt.builder.Build(buf, bt.maxBytes)
			log.Trace().Func(func(e *zerolog.Event) { // process only if loglevel enabled
				e.RawJSON("payloads", append(append([]byte("["), bytes.Join(payloads, []byte(","))...), ']'))
			}).
				Str("bt.tracerName", bt.tracerName).
				Int("bufferLen", len(buf)).
				Int("bufferSize", bufSize).
				Int("maxBytes", bt.maxBytes).
				Msg("Batcher.Batch")
			for _, p := range payloads {
				if len(p) > 0 {
					if err := bt.handler(ctx, p); err != nil {
						log.Err(err).Str("bt.tracerName", bt.tracerName).
							RawJSON("payload", p).
							Int("payloadLen", len(p)).
							Msg("Batcher.Batch handler")
					}
				}
			}
		}()
	}
}

// Exit stops the internal ticker
func (bt *Batcher[T]) Exit() {
	bt.tickerExit <- true
}

// Clear drops all buffered payloads without sending them.
func (bt *Batcher[T]) Clear() {
	bt.mu.Lock()
	defer bt.mu.Unlock()

	bt.buf = make([]Sized[T], 0)
	bt.bufSize = 0
}

// Reset applies configuration
func (bt *Batcher[T]) Reset(d time.Duration, maxBytes int) {
	log.Trace().Str("bt.tracerName", bt.tracerName).Msg("Batcher.Reset")
	bt.Batch()
	bt.maxBytes = maxBytes
	if d == 0 {
		d = math.MaxInt64
	}
	bt.ticker.Reset(d)
}
