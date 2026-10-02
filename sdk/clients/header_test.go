package clients

import (
	"context"
	"net/http"
	"testing"
)

func TestCtxWithHeaderKeepsParentHeader(t *testing.T) {
	parent := CtxWithHeader(context.Background(), http.Header{"A": {"1"}})
	child := CtxWithHeader(parent, http.Header{"B": {"2"}})

	ph, _ := HeaderFromCtx(parent)
	if _, ok := ph["B"]; ok {
		t.Errorf("parent header = %v; must not get the child's values", ph)
	}
	ch, _ := HeaderFromCtx(child)
	if ch.Get("A") != "1" || ch.Get("B") != "2" {
		t.Errorf("child header = %v; want both values", ch)
	}
}

func TestCtxWithHeaderConcurrentChildren(t *testing.T) {
	parent := CtxWithHeader(context.Background(), http.Header{"A": {"1"}})
	done := make(chan struct{})
	for i := 0; i < 8; i++ {
		go func() {
			defer func() { done <- struct{}{} }()
			for j := 0; j < 100; j++ {
				_ = CtxWithHeader(parent, http.Header{"B": {"2"}})
			}
		}()
	}
	for i := 0; i < 8; i++ {
		<-done
	}
}

func TestSendWithContextKeepsCtxHeader(t *testing.T) {
	gw := newFakeGW(t)
	// A value slice with spare capacity, as built by repeated Header.Add calls.
	accept := make([]string, 1, 4)
	accept[0] = "text/plain"
	ctxHeader := http.Header{"Accept": accept}
	ctx := CtxWithHeader(context.Background(), ctxHeader)

	req := Req{URL: gw.URL + "/x", Method: http.MethodGet, Headers: map[string]string{"Accept": contentJSON}}
	if err := req.SendWithContext(ctx); err != nil {
		t.Fatal(err)
	}
	if got := accept[:cap(accept)][1]; got != "" {
		t.Errorf("ctx header backing array was written: %q", got)
	}
	if got := gw.lastRequest(t).header.Values("Accept"); len(got) != 1 || got[0] != "text/plain" {
		t.Errorf("sent Accept = %v; want only the ctx value", got)
	}
}

func TestSendWithContextSkipsInternalHeaders(t *testing.T) {
	const kept = "kept"
	gw := newFakeGW(t)
	ctx := CtxWithHeader(context.Background(), http.Header{
		HdrCompressed:     {"gzip"},
		HdrPayloadLen:     {"10", "20"},
		HdrPayloadType:    {"metrics"},
		HdrSpanSpanID:     {"s"},
		HdrSpanTraceID:    {"t"},
		HdrSpanTraceFlags: {"01"},
		HdrTodoTracerCtx:  {"x"},
		"X-Custom":        {kept},
	})
	req := Req{URL: gw.URL + "/x", Method: http.MethodGet}
	if err := req.SendWithContext(ctx); err != nil {
		t.Fatal(err)
	}
	sent := gw.lastRequest(t).header
	for k := range internalHeaders {
		if v, ok := sent[k]; ok {
			t.Errorf("internal header %s sent: %v", k, v)
		}
	}
	if sent.Get("X-Custom") != kept {
		t.Errorf("X-Custom = %q; want it passed through", sent.Get("X-Custom"))
	}

	// internal headers stay visible in logs
	var logged http.Header
	for _, a := range req.Details() {
		if a.Key == "header" {
			logged, _ = a.Value.Any().(http.Header)
		}
	}
	if got := logged.Values(HdrPayloadLen); len(got) != 2 || logged.Get(HdrPayloadType) != "metrics" ||
		logged.Get(HdrSpanTraceID) != "t" || logged.Get("X-Custom") != kept {
		t.Errorf("logged header = %v; want the sent and internal headers", logged)
	}
}

func TestSendRequestHostNamePrefixOnce(t *testing.T) {
	gw := newFakeGW(t)
	c := gw.client()
	c.PrefixResourceNames, c.ResourceNamePrefix = true, "client-"
	// a non-canonical key, as the azure connector builds it
	ctx := CtxWithHeader(context.Background(), map[string][]string{"HostNamePrefix": {"ctx-"}})
	if _, err := c.SendEvents(ctx, []byte(`{}`)); err != nil {
		t.Fatal(err)
	}
	if got := gw.lastRequest(t).header.Values("HostNamePrefix"); len(got) != 1 || got[0] != "ctx-" {
		t.Errorf("sent HostNamePrefix = %v; want only the ctx value", got)
	}
}
