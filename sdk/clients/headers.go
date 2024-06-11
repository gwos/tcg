package clients

type CtxKey interface{}

var CtxHeaders = CtxKey("headers")

const (
	HdrCompressed     = "compressed"
	HdrPayloadLen     = "payloadLen"
	HdrPayloadType    = "payloadType"
	HdrSpanSpanID     = "spanSpanID"
	HdrSpanTraceID    = "spanTraceID"
	HdrSpanTraceFlags = "spanTraceFlags"
	HdrTodoTracerCtx  = "todoTracerCtx"
)
