package tracing

import (
	"compress/gzip"
	"context"
	"io"
	"net/http"

	"go.opentelemetry.io/contrib/instrumentation/net/http/httptrace/otelhttptrace"
)

func HookRequestContext(ctx context.Context, req *http.Request) (context.Context, *http.Request) {
	ctx, req = otelhttptrace.W3C(ctx, req)
	otelhttptrace.Inject(ctx, req)
	return ctx, req
}

func GZIP(ctx context.Context, w io.Writer, p []byte) (context.Context, error) {
	var (
		err error
		n   int
	)
	ctx, span := StartTraceSpan(ctx, "request", "gzip")
	defer func() {
		EndTraceSpan(span,
			TraceAttrError(err),
			TraceAttrInt("inputLen", len(p)),
			TraceAttrInt("outputLen", n),
		)
	}()
	gw := gzip.NewWriter(w)
	n, err = gw.Write(p)
	_ = gw.Close()
	return ctx, err
}
