package clients

import (
	"bytes"
	"log/slog"
	"net/http"
	"strings"
	"testing"

	sdklog "github.com/gwos/tcg/sdk/log"
)

// TestGWClientLogsNoSecrets checks that request logs going to a plain slog sink,
// as in telegraf, carry neither the login password nor the returned token.
func TestGWClientLogsNoSecrets(t *testing.T) {
	const password, token = "pass-xyz", "tok-xyz"
	var buf bytes.Buffer
	orig := sdklog.Logger
	sdklog.Logger = slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug}))
	t.Cleanup(func() { sdklog.Logger = orig })

	gw := newFakeGW(t)
	gw.respond(GWEntrypointConnect, http.StatusOK, token)
	c := gw.client()
	c.GWConnection.Password = password
	c.GWConnection.LocalConnection = true
	if err := c.Connect(); err != nil {
		t.Fatalf("Connect() error = %v", err)
	}

	// a token with an unexpected success status must not leak through the error either
	gw.respond(GWEntrypointConnect, http.StatusCreated, token)
	if _, err := c.connectLocal(); err == nil || strings.Contains(err.Error(), token) {
		t.Errorf("connectLocal() 201 error = %v", err)
	}

	// a token in an unexpected login response is masked in the returned error
	gw.respond(GWEntrypointAuthenticate, http.StatusCreated, `{"name":"u","accessToken":"`+token+`"}`)
	if _, err := c.AuthenticatePassword("user", password); err == nil || strings.Contains(err.Error(), token) {
		t.Errorf("AuthenticatePassword() 201 error = %v", err)
	}

	gw.respond(GWEntrypointConnect, http.StatusUnauthorized, "denied")
	if _, err := c.connectLocal(); err == nil {
		t.Fatal("connectLocal() 401 error = nil")
	}

	out := buf.String()
	if !strings.Contains(out, "connect local groundwork") || !strings.Contains(out, "could not connect local groundwork") {
		t.Fatalf("expected records missing:\n%s", out)
	}
	for _, secret := range []string{password, token} {
		if strings.Contains(out, secret) {
			t.Errorf("log contains %q:\n%s", secret, out)
		}
	}
}
