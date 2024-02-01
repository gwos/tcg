package batcher

import (
	"bytes"
	"context"
	"math"
	"reflect"
	"sync"
	"time"

	"github.com/gwos/tcg/tracing"
	"github.com/rs/zerolog/log"
	"go.opentelemetry.io/otel/trace"
)

// BatchBuilder defines builder interface
type BatchBuilder interface {
	// Build builds the batch payloads
	// it's possible that not all input payloads can be combined into one
	Build(input [][]byte, maxBytes int) [][]byte
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

	traceCtx   context.Context
	traceSpan  trace.Span
	tracerName string
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

		tracerName: "batcher:" + reflect.TypeOf(bb).String(),
	}
	bt.traceCtx, bt.traceSpan = tracing.StartTraceSpan(context.Background(), bt.tracerName, "batching")

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

	// trace.SpanFromContext(bt.traceCtx).AddEvent("batcher:Add", trace.WithAttributes(
	// 	attribute.Int("payloadLen", len(p)),
	// 	attribute.String("payload", string(p)), // cannot wrap if debug
	// ))
	_, span := tracing.StartTraceSpan(bt.traceCtx, bt.tracerName, "batcher:Add")

	bt.buf = append(bt.buf, p)
	bt.bufSize += len(p)

	tracing.EndTraceSpan(span,
		tracing.TraceAttrPayloadDbg(p),
		tracing.TraceAttrPayloadLen(p),
	)

	bt.mu.Unlock()
	if bt.bufSize > bt.maxBytes {
		log.Debug().Msgf("batch buffer size %dKB exceeded the soft limit %dKB",
			bt.bufSize/1024, bt.maxBytes/1024)
		bt.Batch()
	}
}

// Batch processes buffered payloads
func (bt *Batcher) Batch() {
	bt.mu.Lock()

	buf, bufSize := bt.buf, bt.bufSize
	bt.buf, bt.bufSize = make([][]byte, 0), 0

	bt.mu.Unlock()
	if len(buf) > 0 {
		func() {
			var payloads [][]byte
			/* wrap into closure for simple defer,
			cannot use services package due to import cycle */
			ctx, span := tracing.StartTraceSpan(bt.traceCtx, bt.tracerName, "batcher:Batch")
			defer func() {
				tracing.EndTraceSpan(span,
					tracing.TraceAttrInt("maxBytes", bt.maxBytes),
					tracing.TraceAttrInt("bufferLen", len(buf)),
					tracing.TraceAttrInt("bufferSize", bufSize),
					tracing.TraceAttrFnDbg("buffer", func() string { return string(bytes.Join(buf, []byte("\n"))) }),
					tracing.TraceAttrFnDbg("output", func() string { return string(bytes.Join(payloads, []byte("\n"))) }),
					tracing.TraceAttrInt("outputLen", len(payloads)),
				)
				tracing.EndTraceSpan(bt.traceSpan)
				bt.traceCtx, bt.traceSpan = tracing.StartTraceSpan(context.Background(), bt.tracerName, "batching")
			}()

			payloads = bt.builder.Build(buf, bt.maxBytes)
			if len(payloads) > 0 {
				for _, p := range payloads {
					if len(p) > 0 {
						_ = bt.handler(ctx, p)
					}
				}
			}
		}()
	}
}

// Exit stops the internal ticker
func (bt *Batcher) Exit() {
	bt.tickerExit <- true
}

// Reset applies configuration
func (bt *Batcher) Reset(d time.Duration, maxBytes int) {
	bt.Batch()
	bt.maxBytes = maxBytes
	if d == 0 {
		d = math.MaxInt64
	}
	bt.ticker.Reset(d)
}
