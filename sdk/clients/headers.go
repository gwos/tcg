package clients

type CtxKey interface{}

var CtxHeaders = CtxKey("headers")

const (
	HdrCompressed     = "compressed"
	HdrPayloadType    = "payloadType"
	HdrSpanSpanID     = "spanSpanID"
	HdrSpanTraceID    = "spanTraceID"
	HdrSpanTraceFlags = "spanTraceFlags"
	HdrTodoTracerCtx  = "todoTracerCtx"
)
