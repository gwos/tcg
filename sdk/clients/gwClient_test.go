package clients

import (
	"bytes"
	"compress/gzip"
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"

	tcgerr "github.com/gwos/tcg/sdk/errors"
)

const (
	testAppName = "test-app"
	testToken   = "token-1"
	bodyOK      = "ok"
	bodyBoom    = "boom"
	bodyTrue    = "true"
	eventsURI   = "https://gw.example/api/events"
)

// fakeGW is a GroundWork API stub that answers each path with a configured status and body.
type fakeGW struct {
	*httptest.Server

	mu        sync.Mutex
	responses map[string]fakeResponse
	requests  []fakeRequest
}

type fakeResponse struct {
	status int
	body   string
}

type fakeRequest struct {
	method, path, query string
	header              http.Header
	body                []byte
}

func newFakeGW(t *testing.T) *fakeGW {
	t.Helper()
	f := &fakeGW{responses: map[string]fakeResponse{
		string(GWEntrypointAuthenticate): {http.StatusOK, `{"name":"u","accessToken":"` + testToken + `"}`},
		string(GWEntrypointConnect):      {http.StatusOK, testToken},
	}}
	f.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		f.mu.Lock()
		f.requests = append(f.requests, fakeRequest{r.Method, r.URL.Path, r.URL.RawQuery, r.Header.Clone(), body})
		resp, ok := f.responses[r.URL.Path]
		f.mu.Unlock()
		if !ok {
			resp = fakeResponse{http.StatusOK, bodyOK}
		}
		w.WriteHeader(resp.status)
		_, _ = io.WriteString(w, resp.body)
	}))
	t.Cleanup(f.Close)
	return f
}

func (f *fakeGW) respond(path GWEntrypoint, status int, body string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.responses[string(path)] = fakeResponse{status, body}
}

func (f *fakeGW) lastRequest(t *testing.T) fakeRequest {
	t.Helper()
	f.mu.Lock()
	defer f.mu.Unlock()
	if len(f.requests) == 0 {
		t.Fatal("no requests received")
	}
	return f.requests[len(f.requests)-1]
}

func (f *fakeGW) client() *GWClient {
	return &GWClient{
		AppName: testAppName,
		GWConnection: GWConnection{
			HostName: f.URL,
			UserName: "user",
			Password: "secret",
		},
	}
}

// closedHostName returns a host name that refuses connections.
func closedHostName(t *testing.T) string {
	t.Helper()
	s := httptest.NewServer(http.NotFoundHandler())
	s.Close()
	return s.URL
}

func TestGWClientStatusErrors(t *testing.T) {
	methods := []struct {
		name  string
		path  GWEntrypoint
		login bool
		call  func(*GWClient) error
	}{
		{name: "connectLocal", path: GWEntrypointConnect, login: true, call: func(c *GWClient) error {
			c.LocalConnection = true
			return c.Connect()
		}},
		{name: "AuthenticatePassword", path: GWEntrypointAuthenticate, login: true, call: func(c *GWClient) error {
			_, err := c.AuthenticatePassword("user", "secret")
			return err
		}},
		{name: "Disconnect", path: GWEntrypointDisconnect, call: func(c *GWClient) error {
			return c.Disconnect()
		}},
		{name: "SendRequest", path: GWEntrypointEvents, call: func(c *GWClient) error {
			_, err := c.SendEvents(context.Background(), []byte(`{}`))
			return err
		}},
	}
	statuses := []struct {
		status int
		body   string
		want   error
		// wantLogin overrides want for the login methods
		wantLogin error
	}{
		{http.StatusNotFound, "no such entity", tcgerr.ErrNotFound, nil},
		{http.StatusNotFound, "invalid password", tcgerr.ErrNotFound, tcgerr.ErrUnauthorized},
		{http.StatusBadGateway, "bad gateway", tcgerr.ErrGateway, nil},
		{http.StatusServiceUnavailable, "sync", tcgerr.ErrSynchronizer, nil},
		{http.StatusGatewayTimeout, "slow", tcgerr.ErrGateway, nil},
		{http.StatusInternalServerError, bodyBoom, tcgerr.ErrUndecided, nil},
		{http.StatusCreated, "created", tcgerr.ErrUndecided, nil},
	}
	for _, m := range methods {
		for _, s := range statuses {
			t.Run(m.name+"/"+http.StatusText(s.status)+"/"+s.body, func(t *testing.T) {
				gw := newFakeGW(t)
				gw.respond(m.path, s.status, s.body)
				err := m.call(gw.client())
				want := s.want
				if m.login && s.wantLogin != nil {
					want = s.wantLogin
				}
				if !errors.Is(err, want) {
					t.Fatalf("error = %v; want %v", err, want)
				}
				if got, wantMsg := err.Error(), want.Error()+": "+s.body; got != wantMsg {
					t.Errorf("error text = %q; want %q", got, wantMsg)
				}
			})
		}
	}
}

func TestGWClientUnauthorized(t *testing.T) {
	t.Run("login", func(t *testing.T) {
		gw := newFakeGW(t)
		gw.respond(GWEntrypointAuthenticate, http.StatusUnauthorized, "denied")
		_, err := gw.client().AuthenticatePassword("user", "secret")
		if !errors.Is(err, tcgerr.ErrUnauthorized) || err.Error() != tcgerr.ErrUnauthorized.Error()+": denied" {
			t.Fatalf("error = %v", err)
		}
	})
	t.Run("disconnect", func(t *testing.T) {
		gw := newFakeGW(t)
		gw.respond(GWEntrypointDisconnect, http.StatusUnauthorized, "denied")
		if err := gw.client().Disconnect(); !errors.Is(err, tcgerr.ErrUnauthorized) {
			t.Fatalf("error = %v", err)
		}
	})
	t.Run("send reconnect fails", func(t *testing.T) {
		gw := newFakeGW(t)
		gw.respond(GWEntrypointEvents, http.StatusUnauthorized, "expired")
		gw.respond(GWEntrypointAuthenticate, http.StatusUnauthorized, "denied")
		_, err := gw.client().SendEvents(context.Background(), []byte(`{}`))
		if !errors.Is(err, tcgerr.ErrUnauthorized) || !strings.HasSuffix(err.Error(), ": denied") {
			t.Fatalf("error = %v; want the reconnect error", err)
		}
	})
	t.Run("send still unauthorized after reconnect", func(t *testing.T) {
		gw := newFakeGW(t)
		gw.respond(GWEntrypointEvents, http.StatusUnauthorized, "expired")
		_, err := gw.client().SendEvents(context.Background(), []byte(`{}`))
		if !errors.Is(err, tcgerr.ErrUnauthorized) || !strings.HasSuffix(err.Error(), ": expired") {
			t.Fatalf("error = %v", err)
		}
	})
}

func TestGWClientSendRetriesWithNewToken(t *testing.T) {
	gw := newFakeGW(t)
	var calls int
	gw.Config.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case string(GWEntrypointAuthenticate):
			_, _ = io.WriteString(w, `{"accessToken":"fresh"}`)
		case string(GWEntrypointEvents):
			calls++
			if r.Header.Get("GWOS-API-TOKEN") != "fresh" {
				w.WriteHeader(http.StatusUnauthorized)
				return
			}
			_, _ = io.WriteString(w, "sent")
		}
	})

	resp, err := gw.client().SendEvents(context.Background(), []byte(`{}`))
	if err != nil || string(resp) != "sent" {
		t.Fatalf("SendEvents() = %q, %v", resp, err)
	}
	if calls != 2 {
		t.Errorf("events calls = %d; want 2", calls)
	}
}

func TestGWClientSuccess(t *testing.T) {
	gw := newFakeGW(t)

	c := gw.client()
	c.LocalConnection = true
	if err := c.Connect(); err != nil {
		t.Fatalf("Connect() local = %v", err)
	}
	if c.token() != testToken {
		t.Errorf("token = %q; want %q", c.token(), testToken)
	}
	if req := gw.lastRequest(t); req.method != http.MethodPost ||
		req.header.Get("Content-Type") != "application/x-www-form-urlencoded" {
		t.Errorf("connect request = %+v", req)
	} else if form, _ := url.ParseQuery(string(req.body)); form.Get("gwos-app-name") != testAppName ||
		form.Get("user") != "user" || form.Get("password") != "secret" {
		t.Errorf("connect form = %v", form)
	}

	token, err := gw.client().AuthenticatePassword("user", "secret")
	if err != nil || token != testToken {
		t.Fatalf("AuthenticatePassword() = %q, %v", token, err)
	}
	if req := gw.lastRequest(t); req.method != http.MethodPut || req.header.Get("GWOS-APP-NAME") != testAppName {
		t.Errorf("authenticate request = %+v", req)
	}

	if err := c.Disconnect(); err != nil {
		t.Fatalf("Disconnect() = %v", err)
	}
	if form, _ := url.ParseQuery(string(gw.lastRequest(t).body)); form.Get("gwos-api-token") != testToken {
		t.Errorf("disconnect form = %v", form)
	}

	gw.respond(GWEntrypointAuthenticate, http.StatusOK, `not json`)
	if _, err := gw.client().AuthenticatePassword("user", "secret"); !errors.Is(err, tcgerr.ErrUndecided) {
		t.Errorf("AuthenticatePassword() bad JSON error = %v", err)
	}
}

func TestGWClientTransportErrors(t *testing.T) {
	hostName := closedHostName(t)
	c := &GWClient{AppName: testAppName, GWConnection: GWConnection{HostName: hostName}}
	calls := map[string]func() error{
		"connectLocal": func() error {
			lc := &GWClient{AppName: testAppName, GWConnection: GWConnection{HostName: hostName, LocalConnection: true}}
			return lc.Connect()
		},
		"AuthenticatePassword": func() error { _, err := c.AuthenticatePassword("u", "p"); return err },
		"Disconnect":           func() error { return c.Disconnect() },
		"ValidateToken":        func() error { return c.ValidateToken("a", "t") },
		"SendRequest":          func() error { _, err := c.SendEvents(context.Background(), nil); return err },
	}
	for name, call := range calls {
		t.Run(name, func(t *testing.T) {
			if err := call(); !errors.Is(err, tcgerr.ErrTransient) {
				t.Fatalf("error = %v; want %v", err, tcgerr.ErrTransient)
			}
		})
	}

	t.Run("SendRequest canceled", func(t *testing.T) {
		gw := newFakeGW(t)
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		if _, err := gw.client().SendEvents(ctx, nil); !errors.Is(err, tcgerr.ErrTransient) {
			t.Fatalf("error = %v; want %v", err, tcgerr.ErrTransient)
		}
	})
}

func TestGWClientValidateToken(t *testing.T) {
	cases := []struct {
		status int
		body   string
		want   error
	}{
		{http.StatusOK, bodyTrue, nil},
		{http.StatusOK, "false", tcgerr.ErrUnauthorized},
		{http.StatusOK, "maybe", tcgerr.ErrUnauthorized},
		{http.StatusInternalServerError, bodyBoom, tcgerr.ErrUndecided},
	}
	for _, tc := range cases {
		t.Run(tc.body, func(t *testing.T) {
			gw := newFakeGW(t)
			gw.respond(GWEntrypointValidateToken, tc.status, tc.body)
			err := gw.client().ValidateToken("app", "tok")
			if tc.want == nil && err != nil || tc.want != nil && !errors.Is(err, tc.want) {
				t.Fatalf("error = %v; want %v", err, tc.want)
			}
			if form, _ := url.ParseQuery(string(gw.lastRequest(t).body)); form.Get("gwos-app-name") != "app" ||
				form.Get("gwos-api-token") != "tok" {
				t.Errorf("form = %v", form)
			}
		})
	}
}

func TestGWClientSendMethods(t *testing.T) {
	send := map[GWEntrypoint]func(*GWClient, []byte) ([]byte, error){
		GWEntrypointClearInDowntime: func(c *GWClient, p []byte) ([]byte, error) { return c.ClearInDowntime(context.Background(), p) },
		GWEntrypointSetInDowntime:   func(c *GWClient, p []byte) ([]byte, error) { return c.SetInDowntime(context.Background(), p) },
		GWEntrypointEvents:          func(c *GWClient, p []byte) ([]byte, error) { return c.SendEvents(context.Background(), p) },
		GWEntrypointEventsAck:       func(c *GWClient, p []byte) ([]byte, error) { return c.SendEventsAck(context.Background(), p) },
		GWEntrypointEventsUnack:     func(c *GWClient, p []byte) ([]byte, error) { return c.SendEventsUnack(context.Background(), p) },
		GWEntrypointMonitoring: func(c *GWClient, p []byte) ([]byte, error) {
			return c.SendResourcesWithMetrics(context.Background(), p)
		},
		GWEntrypointSynchronizer: func(c *GWClient, p []byte) ([]byte, error) { return c.SynchronizeInventory(context.Background(), p) },
	}
	payload := []byte(`{"key":"value"}`)
	for path, call := range send {
		for _, prefixed := range []bool{false, true} {
			t.Run(string(path)+map[bool]string{false: "", true: "/prefixed"}[prefixed], func(t *testing.T) {
				gw := newFakeGW(t)
				c := gw.client()
				c.PrefixResourceNames, c.ResourceNamePrefix = prefixed, "pfx-"
				if err := c.Connect(); err != nil {
					t.Fatal(err)
				}

				resp, err := call(c, payload)
				if err != nil || string(resp) != bodyOK {
					t.Fatalf("send = %q, %v", resp, err)
				}
				req := gw.lastRequest(t)
				if req.method != http.MethodPost || req.path != string(path) || !bytes.Equal(req.body, payload) {
					t.Errorf("request = %s %s %q", req.method, req.path, req.body)
				}
				if got := req.header.Get("Hostnameprefix"); prefixed && got != "pfx-" || !prefixed && got != "" {
					t.Errorf("HostNamePrefix header = %q", got)
				}
				if req.header.Get("GWOS-API-TOKEN") != testToken || req.header.Get("GWOS-APP-NAME") != testAppName ||
					req.header.Get("Content-Type") != "application/json" || req.header.Get("Content-Encoding") != "" {
					t.Errorf("headers = %v", req.header)
				}
				if path == GWEntrypointSynchronizer && req.query != "merge=false" {
					t.Errorf("query = %q", req.query)
				}
			})
		}
	}

	t.Run("dynamic inventory and merge hosts", func(t *testing.T) {
		gw := newFakeGW(t)
		c := gw.client()
		c.IsDynamicInventory, c.MergeHosts = true, true
		if _, err := c.SendResourcesWithMetrics(context.Background(), payload); err != nil {
			t.Fatal(err)
		}
		if req := gw.lastRequest(t); req.path != string(GWEntrypointMonitoring) || req.query != "dynamic=true" {
			t.Errorf("request = %s?%s", req.path, req.query)
		}
		if _, err := c.SynchronizeInventory(context.Background(), payload); err != nil {
			t.Fatal(err)
		}
		if req := gw.lastRequest(t); req.query != "merge=true" {
			t.Errorf("query = %q", req.query)
		}
	})
}

func gunzip(t *testing.T, p []byte) []byte {
	t.Helper()
	r, err := gzip.NewReader(bytes.NewReader(p))
	if err != nil {
		t.Fatal(err)
	}
	b, err := io.ReadAll(r)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func TestGWClientEncoding(t *testing.T) {
	small := []byte(`{"key":"value"}`)
	large := bytes.Repeat([]byte("x"), httpEncodeMinSize+1)
	var gzipped bytes.Buffer
	if _, err := GZip(context.Background(), &gzipped, small); err != nil {
		t.Fatal(err)
	}

	cases := []struct {
		name       string
		httpEncode bool
		payload    []byte
		compress   bool
		want       []byte
	}{
		{"plain", false, small, false, small},
		{"http encode", true, small, true, small},
		{"large", false, large, true, large},
		{"already gzipped", false, gzipped.Bytes(), true, small},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			gw := newFakeGW(t)
			c := gw.client()
			c.HTTPEncode = tc.httpEncode
			if _, err := c.SendResourcesWithMetrics(context.Background(), tc.payload); err != nil {
				t.Fatal(err)
			}
			req := gw.lastRequest(t)
			body := req.body
			if tc.compress {
				if req.header.Get("Content-Encoding") != "gzip" {
					t.Fatalf("Content-Encoding = %q", req.header.Get("Content-Encoding"))
				}
				body = gunzip(t, body)
			} else if req.header.Get("Content-Encoding") != "" {
				t.Fatalf("Content-Encoding = %q", req.header.Get("Content-Encoding"))
			}
			if !bytes.Equal(body, tc.want) {
				t.Errorf("body length = %d; want %d", len(body), len(tc.want))
			}
		})
	}
}

func TestGWClientQueries(t *testing.T) {
	gw := newFakeGW(t)
	gw.respond(GWEntrypointServices, http.StatusOK, `{"services":[{"description":"svc","hostName":"h1"}]}`)
	gw.respond(GWEntrypointHostgroups, http.StatusOK, `{"hostGroups":[{"name":"g1","hosts":[{"hostName":"h1"}]}]}`)
	c := gw.client()

	var services GWServices
	if err := c.GetServicesByAgent("agent-1", &services); err != nil {
		t.Fatal(err)
	}
	if len(services.Services) != 1 || services.Services[0].HostName != "h1" {
		t.Errorf("services = %+v", services)
	}
	q, _ := url.ParseQuery(gw.lastRequest(t).query)
	if q.Get("query") != "agentid='agent-1'" || q.Get("depth") != "Shallow" {
		t.Errorf("services query = %v", q)
	}

	var groups GWHostGroups
	if err := c.GetHostGroupsByAppTypeAndHostNames("NAGIOS", []string{"h1", "h2"}, &groups); err != nil {
		t.Fatal(err)
	}
	if len(groups.HostGroups) != 1 || groups.HostGroups[0].Name != "g1" {
		t.Errorf("groups = %+v", groups)
	}
	q, _ = url.ParseQuery(gw.lastRequest(t).query)
	if want := "( hosts.hostName in ('h1','h2') ) and appType = 'NAGIOS'"; q.Get("query") != want {
		t.Errorf("hostgroups query = %q; want %q", q.Get("query"), want)
	}

	if err := c.GetHostGroupsByAppTypeAndHostNames("NAGIOS", nil, &groups); err == nil {
		t.Error("expected an error for no host names")
	}
	gw.respond(GWEntrypointServices, http.StatusOK, `not json`)
	if err := c.GetServicesByAgent("agent-1", &services); err == nil {
		t.Error("expected a parse error")
	}
	gw.respond(GWEntrypointHostgroups, http.StatusInternalServerError, bodyBoom)
	if err := c.GetHostGroupsByAppTypeAndHostNames("NAGIOS", []string{"h1"}, &groups); !errors.Is(err, tcgerr.ErrUndecided) {
		t.Errorf("error = %v", err)
	}
}

func TestBuildURI(t *testing.T) {
	cases := []struct{ host, want string }{
		{"gw.example", eventsURI},
		{"gw.example/", eventsURI},
		{"gw.example/api", eventsURI},
		{"gw.example/api/", eventsURI},
		{"http://gw.example:8080", "http://gw.example:8080/api/events"},
	}
	for _, tc := range cases {
		if got := buildURI(&strings.Builder{}, tc.host, GWEntrypointEvents); got != tc.want {
			t.Errorf("buildURI(%q) = %q; want %q", tc.host, got, tc.want)
		}
	}
	if got := buildURI(&strings.Builder{}, "gw.example", GWEntrypointMonitoringDyn); got != "https://gw.example/api/monitoring?dynamic=true" {
		t.Errorf("buildURI(dyn) = %q", got)
	}
}

func TestBuildQueryParams(t *testing.T) {
	if got := BuildQueryParams(nil); got != "" {
		t.Errorf("BuildQueryParams(nil) = %q", got)
	}
	if got := BuildQueryParams(map[string]string{"q": "a b&c"}); got != "?q=a+b%26c" {
		t.Errorf("BuildQueryParams(one) = %q", got)
	}
	got := BuildQueryParams(map[string]string{"a": "1", "b": "2"})
	if got != "?a=1&b=2" && got != "?b=2&a=1" {
		t.Errorf("BuildQueryParams(two) = %q", got)
	}
}

func TestIsJSON(t *testing.T) {
	for s, want := range map[string]bool{`{}`: true, `[1]`: true, `{`: false, `"x"`: false, ``: false, `{]`: false} {
		if got := IsJSON([]byte(s)); got != want {
			t.Errorf("IsJSON(%q) = %v", s, got)
		}
	}
}

func BenchmarkGZip(b *testing.B) {
	payload := bytes.Repeat([]byte(`{"name":"svc","status":"SERVICE_OK","value":42},`), 2000)
	b.ReportAllocs()
	for b.Loop() {
		var buf bytes.Buffer
		if _, err := GZip(context.Background(), &buf, payload); err != nil {
			b.Fatal(err)
		}
	}
}
