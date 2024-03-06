package tracing

import (
	"context"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

var IsDebugEnabled = func() bool { return false }

// TraceAttrOption defines option to set span attribute
type TraceAttrOption func(span trace.Span)

// StartTraceSpan starts a span
func StartTraceSpan(ctx context.Context, tracerName, spanName string, opts ...trace.SpanStartOption) (context.Context, trace.Span) {
	return otel.GetTracerProvider().
		Tracer(tracerName).Start(ctx, spanName, opts...)
}

// EndTraceSpan ends span, optionally sets attributes
func EndTraceSpan(span trace.Span, opts ...TraceAttrOption) {
	for _, optFn := range opts {
		optFn(span)
	}
	span.End()
}

// TraceAttrFnDbg sets an attribute with Function if Debug is enabled
func TraceAttrFnDbg(k string, fn func() string) TraceAttrOption {
	return func(span trace.Span) {
		if IsDebugEnabled() {
			span.SetAttributes(attribute.String(k, fn()))
		}
	}
}

// TraceAttrInt sets an int attribute
func TraceAttrInt(k string, v int) TraceAttrOption {
	return func(span trace.Span) { span.SetAttributes(attribute.Int(k, v)) }
}

// TraceAttrStr sets a string attribute
func TraceAttrStr(k, v string) TraceAttrOption {
	return func(span trace.Span) { span.SetAttributes(attribute.String(k, v)) }
}

// TraceAttrStrs sets an string slice attribute
func TraceAttrStrs(k string, v []string) TraceAttrOption {
	return func(span trace.Span) { span.SetAttributes(attribute.StringSlice(k, v)) }
}

// TraceAttrEntrypoint sets an entrypoint attribute
func TraceAttrEntrypoint(v string) TraceAttrOption {
	return func(span trace.Span) { span.SetAttributes(attribute.String("entrypoint", v)) }
}

// TraceAttrError sets an error attribute
func TraceAttrError(v error) TraceAttrOption {
	return func(span trace.Span) {
		if v == nil {
			span.SetAttributes(attribute.Bool("err", false))
			span.SetAttributes(attribute.String("error", ""))
			return
		}
		span.SetAttributes(attribute.Bool("err", true))
		span.SetAttributes(attribute.String("error", v.Error()))
	}
}

// TraceAttrPayload sets a payload attribute
func TraceAttrPayload(v []byte) TraceAttrOption {
	return func(span trace.Span) { span.SetAttributes(attribute.String("payload", string(v))) }
}

// TraceAttrPayloadDbg sets a payload attribute if Debug is enabled
func TraceAttrPayloadDbg(v []byte) TraceAttrOption {
	return func(span trace.Span) {
		if IsDebugEnabled() {
			span.SetAttributes(attribute.String("payload", string(v)))
		}
	}
}

// TraceAttrPayloadLen sets a payloadLen attribute
func TraceAttrPayloadLen(v []byte) TraceAttrOption {
	return func(span trace.Span) { span.SetAttributes(attribute.Int("payloadLen", len(v))) }
}
