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
	HdrPayloadLen     = "Payload-Lenght"
	HdrPayloadType    = "Payload-Type"
	HdrSpanSpanID     = "Span-Span-Id"
	HdrSpanTraceID    = "Span-Trace-Id"
	HdrSpanTraceFlags = "Span-Trace-Flags"
	HdrTodoTracerCtx  = "Todo-Tracer-Ctx"
)

type CtxKey any

var ctxHeader = CtxKey("header")

func CtxWithHeader(ctx context.Context, header http.Header) context.Context {
	if h, ok := ctx.Value(ctxHeader).(http.Header); ok {
		maps.Copy(h, header)
		return context.WithValue(ctx, ctxHeader, h)
	}
	return context.WithValue(ctx, ctxHeader, header)
}

func HeaderFromCtx(ctx context.Context) (http.Header, bool) {
	h, ok := ctx.Value(ctxHeader).(http.Header)
	return h, ok
}
