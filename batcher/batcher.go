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

// BatchBuilder defines builder interface
type BatchBuilder interface {
	// Build builds the batch payloads
	// it's possible that not all input payloads can be combined into one
	Build(buf *[][]byte, maxBytes int)
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
	xBatchedAt *expvar.Int
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

// Add adds single payload to batch buffer
func (bt *Batcher) Add(p []byte) {
	bt.mu.Lock()

	// trace.SpanFromContext(bt.traceCtx).AddEvent("batcher:Add", trace.WithAttributes(
	// 	attribute.Int("payloadLen", len(p)),
	// 	attribute.String("payload", string(p)), // cannot wrap if debug
	// ))
	_, span := tracing.StartTraceSpan(bt.traceCtx, bt.tracerName, "batcher:Add")
	log.Trace().Str("bt.tracerName", bt.tracerName).
		RawJSON("payload", p).
		Int("payloadLen", len(p)).
		Msg("Batcher.Add")

	bt.buf = append(bt.buf, p)
	bt.bufSize += len(p)

	tracing.EndTraceSpan(span,
		tracing.TraceAttrPayloadDbg(p),
		tracing.TraceAttrPayloadLen(p),
	)

	bt.mu.Unlock()
	if bt.bufSize > bt.maxBytes {
		log.Trace().Str("bt.tracerName", bt.tracerName).
			Msgf("batch buffer size %dKB exceeded the soft limit %dKB",
				bt.bufSize/1024, bt.maxBytes/1024)
		bt.Batch()
	}
}

// Batch processes buffered payloads
func (bt *Batcher) Batch() {
	bt.xBatchedAt.Set(time.Now().UnixMilli())
	bt.mu.Lock()

	buf, bufSize := bt.buf, bt.bufSize
	bt.buf, bt.bufSize = make([][]byte, 0), 0

	bt.mu.Unlock()
	if len(buf) > 0 {
		func() {
			/* wrap into closure for simple defer,
			cannot use services package due to import cycle */
			ctx, span := tracing.StartTraceSpan(bt.traceCtx, bt.tracerName, "batcher:Batch")
			defer func() {
				tracing.EndTraceSpan(span,
					tracing.TraceAttrInt("maxBytes", bt.maxBytes),
					tracing.TraceAttrInt("bufferLen", len(buf)),
					tracing.TraceAttrInt("bufferSize", bufSize),
					tracing.TraceAttrFnDbg("buffer", func() string { return string(bytes.Join(buf, []byte("\n"))) }),
					// tracing.TraceAttrFnDbg("output", func() string { return string(bytes.Join(payloads, []byte("\n"))) }),
					// tracing.TraceAttrInt("outputLen", len(payloads)),
				)
				tracing.EndTraceSpan(bt.traceSpan)
				bt.traceCtx, bt.traceSpan = tracing.StartTraceSpan(context.Background(), bt.tracerName, "batching")
			}()

			bt.builder.Build(&buf, bt.maxBytes)
			log.Trace().Func(func(e *zerolog.Event) { // process only if loglevel enabled
				e.RawJSON("buf", append(append([]byte("["), bytes.Join(buf, []byte(","))...), ']'))
			}).
				Str("bt.tracerName", bt.tracerName).
				Int("bufferLen", len(buf)).
				Int("bufferSize", bufSize).
				Int("maxBytes", bt.maxBytes).
				Msg("Batcher.Batch")
			if len(buf) > 0 {
				for _, p := range buf {
					if len(p) > 0 {
						if err := bt.handler(ctx, p); err != nil {
							log.Err(err).Str("bt.tracerName", bt.tracerName).
								RawJSON("payload", p).
								Int("payloadLen", len(p)).
								Msg("Batcher.Batch handler")
						}
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
	log.Trace().Str("bt.tracerName", bt.tracerName).Msg("Batcher.Reset")
	bt.Batch()
	bt.maxBytes = maxBytes
	if d == 0 {
		d = math.MaxInt64
	}
	bt.ticker.Reset(d)
}
