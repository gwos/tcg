package clients

type CtxKey interface{}

var CtxHeaders = CtxKey("headers")

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
