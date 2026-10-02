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
	if got := gw.lastRequest(t).header.Values("Accept"); len(got) != 2 {
		t.Errorf("sent Accept = %v; want the ctx and request values", got)
	}
}
