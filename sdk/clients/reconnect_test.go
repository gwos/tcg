package clients

import (
	"context"
	"io"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
)

// TestSendRequestReconnectsOncePerExpiredToken sends two requests with the same expired token.
// The second one gets its 401 only after the first has already logged in again,
// so it must retry with the new token instead of logging in once more.
func TestSendRequestReconnectsOncePerExpiredToken(t *testing.T) {
	var (
		logins    atomic.Int32
		staleSeen atomic.Int32
		firstDone = make(chan struct{})
		closeOnce sync.Once
	)
	gw := newFakeGW(t)
	gw.Config.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case string(GWEntrypointAuthenticate):
			n := logins.Add(1)
			_, _ = io.WriteString(w, `{"accessToken":"fresh-`+strconv.Itoa(int(n))+`"}`)
		case string(GWEntrypointEvents):
			if strings.HasPrefix(r.Header.Get("GWOS-API-TOKEN"), "fresh-") {
				_, _ = io.WriteString(w, bodyOK)
				closeOnce.Do(func() { close(firstDone) })
				return
			}
			if staleSeen.Add(1) == 2 {
				<-firstDone
			}
			w.WriteHeader(http.StatusUnauthorized)
		}
	})

	c := gw.client()
	tokens.Store(c.tokenKey(), "stale")

	var wg sync.WaitGroup
	for range 2 {
		wg.Go(func() {
			if _, err := c.SendEvents(context.Background(), []byte(`{}`)); err != nil {
				t.Error(err)
			}
		})
	}
	wg.Wait()

	if n := logins.Load(); n != 1 {
		t.Errorf("logins = %d; want 1", n)
	}
}
