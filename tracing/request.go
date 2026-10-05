package tracing

import (
	"context"
	"io"
	"net/http"

	"github.com/gwos/tcg/sdk/clients"
	"go.opentelemetry.io/contrib/instrumentation/net/http/httptrace/otelhttptrace"
)

func HookRequestContext(ctx context.Context, req *http.Request) (context.Context, *http.Request) {
	ctx, req = otelhttptrace.W3C(ctx, req)
	otelhttptrace.Inject(ctx, req)
	return ctx, req
}

// countingWriter counts the bytes written through it
type countingWriter struct {
	w io.Writer
	n int
}

func (cw *countingWriter) Write(p []byte) (int, error) {
	n, err := cw.w.Write(p)
	cw.n += n
	return n, err
}

func GZip(ctx context.Context, w io.Writer, p []byte) (context.Context, error) {
	var (
		err error
		cw  = &countingWriter{w: w}
	)
	ctx, span := StartTraceSpan(ctx, "request", "gzip")
	defer func() {
		EndTraceSpan(span,
			TraceAttrError(err),
			TraceAttrInt("inputLen", len(p)),
			TraceAttrInt("outputLen", cw.n),
		)
	}()
	err = clients.GZipTo(cw, p)
	return ctx, err
}
