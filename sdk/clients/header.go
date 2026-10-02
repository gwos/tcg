package clients

import (
	"context"
	"maps"
	"net/http"
)

// Header key is canonicalized by [textproto.CanonicalMIMEHeaderKey].
// That's important for Get/Set operations on http.Header
const (
	HdrCompressed     = "Compressed"
	HdrPayloadLen     = "Payload-Length"
	HdrPayloadType    = "Payload-Type"
	HdrSpanSpanID     = "Span-Span-Id"
	HdrSpanTraceID    = "Span-Trace-Id"
	HdrSpanTraceFlags = "Span-Trace-Flags"
	HdrTodoTracerCtx  = "Todo-Tracer-Ctx"
)

type ctxKeyType int

const ctxHeader ctxKeyType = iota

// CtxWithHeader returns a context carrying header merged over the header of ctx, if any.
// The header of ctx is not modified, so contexts derived from one parent stay independent.
func CtxWithHeader(ctx context.Context, header http.Header) context.Context {
	if h, ok := ctx.Value(ctxHeader).(http.Header); ok {
		merged := h.Clone()
		maps.Copy(merged, header)
		return context.WithValue(ctx, ctxHeader, merged)
	}
	return context.WithValue(ctx, ctxHeader, header)
}

func HeaderFromCtx(ctx context.Context) (http.Header, bool) {
	h, ok := ctx.Value(ctxHeader).(http.Header)
	return h, ok
}
