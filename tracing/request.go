package tracing

import (
	"bytes"
	"compress/gzip"
	"context"
	"net/http"

	"go.opentelemetry.io/contrib/instrumentation/net/http/httptrace/otelhttptrace"
)

func HookRequestContext(ctx context.Context, req *http.Request) (context.Context, *http.Request) {
	ctx, req = otelhttptrace.W3C(ctx, req)
	otelhttptrace.Inject(ctx, req)
	return ctx, req
}

func GZIP(ctx context.Context, p []byte) (context.Context, []byte, error) {
	var (
		err    error
		output []byte
	)
	ctx, span := StartTraceSpan(ctx, "request", "gzip")
	defer func() {
		EndTraceSpan(span,
			TraceAttrError(err),
			TraceAttrInt("inputLen", len(p)),
			TraceAttrInt("outputLen", len(output)),
		)
	}()
	buf := &bytes.Buffer{}
	gw := gzip.NewWriter(buf)
	_, err = gw.Write(p)
	_ = gw.Close()
	output = buf.Bytes()
	return ctx, output, err
}
