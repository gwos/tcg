package clients

import (
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	tcgerr "github.com/gwos/tcg/sdk/errors"
)

// newFakeDS starts a TLS DalekServices stub, since DSClient uses https for such host names.
func newFakeDS(t *testing.T, status int, body string) (*DSClient, *http.Request, *[]byte) {
	t.Helper()
	var (
		gotReq  = new(http.Request)
		gotBody = new([]byte)
	)
	s := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		*gotBody, _ = io.ReadAll(r.Body)
		*gotReq = *r.Clone(r.Context())
		w.WriteHeader(status)
		_, _ = io.WriteString(w, body)
	}))
	t.Cleanup(s.Close)

	tlsConfig := HttpClientTransport.TLSClientConfig
	insecure := tlsConfig.InsecureSkipVerify
	tlsConfig.InsecureSkipVerify = true
	t.Cleanup(func() { tlsConfig.InsecureSkipVerify = insecure })

	return &DSClient{DSConnection{HostName: strings.TrimPrefix(s.URL, "https://")}}, gotReq, gotBody
}

func TestDSClientValidateToken(t *testing.T) {
	cases := []struct {
		status int
		body   string
		want   error
	}{
		{http.StatusCreated, bodyTrue, nil},
		{http.StatusCreated, "false", tcgerr.ErrUnauthorized},
		{http.StatusOK, bodyTrue, tcgerr.ErrUndecided},
		{http.StatusInternalServerError, bodyBoom, tcgerr.ErrUndecided},
	}
	for _, tc := range cases {
		t.Run(http.StatusText(tc.status)+"/"+tc.body, func(t *testing.T) {
			c, req, body := newFakeDS(t, tc.status, tc.body)
			err := c.ValidateToken("app", "tok")
			if tc.want == nil && err != nil || tc.want != nil && !errors.Is(err, tc.want) {
				t.Fatalf("error = %v; want %v", err, tc.want)
			}
			if req.URL.Path != DSEntrypointValidateToken || req.Method != http.MethodPost {
				t.Errorf("request = %s %s", req.Method, req.URL.Path)
			}
			if form, _ := url.ParseQuery(string(*body)); form.Get("gwos-app-name") != "app" || form.Get("gwos-api-token") != "tok" {
				t.Errorf("form = %v", form)
			}
		})
	}
}

func TestDSClientReload(t *testing.T) {
	cases := []struct {
		status int
		want   error
	}{
		{http.StatusCreated, nil},
		{http.StatusNotFound, tcgerr.ErrNotFound},
		{http.StatusOK, tcgerr.ErrUndecided},
	}
	for _, tc := range cases {
		t.Run(http.StatusText(tc.status), func(t *testing.T) {
			c, req, _ := newFakeDS(t, tc.status, "")
			err := c.Reload("agent-1")
			if tc.want == nil && err != nil || tc.want != nil && !errors.Is(err, tc.want) {
				t.Fatalf("error = %v; want %v", err, tc.want)
			}
			if req.URL.Path != "/dalekservices/connectors/reload/agent-1" || req.Header.Get("Content-Type") != contentJSON {
				t.Errorf("request = %s %v", req.URL.Path, req.Header)
			}
		})
	}
}

func TestDSClientNotConfigured(t *testing.T) {
	c := &DSClient{}
	if err := c.ValidateToken("app", "tok"); err != nil {
		t.Errorf("ValidateToken() = %v", err)
	}
	if err := c.Reload("agent-1"); err != nil {
		t.Errorf("Reload() = %v", err)
	}
}

func TestDSClientTransportError(t *testing.T) {
	c := &DSClient{DSConnection{HostName: strings.TrimPrefix(closedHostName(t), "http://")}}
	if err := c.ValidateToken("app", "tok"); err == nil || errors.Is(err, tcgerr.ErrTransient) {
		t.Errorf("ValidateToken() = %v; want an unwrapped transport error", err)
	}
	if err := c.Reload("agent-1"); err == nil || errors.Is(err, tcgerr.ErrTransient) {
		t.Errorf("Reload() = %v; want an unwrapped transport error", err)
	}
}

func TestMakeDalekServicesScheme(t *testing.T) {
	if got := makeDalekServicesScheme("dalekservices:3001"); got != "http" {
		t.Errorf("scheme = %q", got)
	}
	if got := makeDalekServicesScheme("gw.example"); got != "https" {
		t.Errorf("scheme = %q", got)
	}
}
