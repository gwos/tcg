package tracing

import (
	"context"
	"net/http"

	"go.opentelemetry.io/contrib/instrumentation/net/http/httptrace/otelhttptrace"
)

func HookRequestContext(ctx context.Context, req *http.Request) (context.Context, *http.Request) {
	ctx, req = otelhttptrace.W3C(ctx, req)
	otelhttptrace.Inject(ctx, req)
	return ctx, req
}
