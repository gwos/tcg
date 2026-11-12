package log

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/url"
	"testing"
)

// captureHandler records the last handled record
type captureHandler struct {
	level slog.Level
	attrs []slog.Attr
	last  *slog.Record
}

func (h *captureHandler) Enabled(_ context.Context, l slog.Level) bool { return l >= h.level }

func (h *captureHandler) Handle(_ context.Context, r slog.Record) error {
	h.last = &r
	return nil
}

func (h *captureHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	h.attrs = append(h.attrs, attrs...)
	return h
}

func (h *captureHandler) WithGroup(string) slog.Handler { return h }

func (h *captureHandler) attr(t *testing.T, key string) slog.Value {
	t.Helper()
	if h.last == nil {
		t.Fatal("no record handled")
	}
	var v slog.Value
	found := false
	h.last.Attrs(func(a slog.Attr) bool {
		if a.Key == key {
			v, found = a.Value, true
			return false
		}
		return true
	})
	if !found {
		t.Fatalf("attr %q not found", key)
	}
	return v
}

func withSink(t *testing.T, h slog.Handler) {
	t.Helper()
	orig := Logger
	Logger = slog.New(h)
	t.Cleanup(func() { Logger = orig })
}

func TestXLoggerMasks(t *testing.T) {
	sink := &captureHandler{level: slog.LevelDebug}
	withSink(t, sink)

	header := http.Header{
		"Gwos-Api-Token": {"tok-123"},
		"Authorization":  {"Bearer abc"},
		"Gwos-App-Name":  {"tcg"},
	}
	form := url.Values{"user": {"admin"}, "password": {"pass-123"}}
	payload := json.RawMessage(`{"token":"tok-456","nested":{"password":"pass-456"},"name":"host"}`)

	XLogger.LogAttrs(context.Background(), slog.LevelDebug, "authenticate password=pass-789",
		slog.Any("header", header),
		slog.Any("form", form),
		slog.Any("payload", payload),
		slog.String("response", `token: tok-789 ok`),
		slog.String("url", "http://host/api?apikey=key-123&x=1"),
		slog.String("secret", "plain"),
		slog.Any("error", errors.New("bad Authorization: Basic dXNlcjpwYXNz")),
		slog.Int("status", 401),
		slog.Group("auth", slog.String("token", "tok-000"), slog.String("user", "admin")),
		slog.Group("password", slog.String("value", "pass-000")),
		slog.Any("formvalues", map[string]string{"password": "pass-111", "user": "admin"}),
		slog.Any("nilerr", (*nilErr)(nil)),
	)

	if got := sink.last.Message; got != "authenticate password=***" {
		t.Errorf("message: %q", got)
	}
	h := sink.attr(t, "header").Any().(http.Header)
	if h.Get("Gwos-Api-Token") != MaskValue || h.Get("Authorization") != MaskValue || h.Get("Gwos-App-Name") != "tcg" {
		t.Errorf("header: %v", h)
	}
	if header.Get("Gwos-Api-Token") != "tok-123" {
		t.Errorf("source header mutated: %v", header)
	}
	f := sink.attr(t, "form").Any().(url.Values)
	if f.Get("password") != MaskValue || f.Get("user") != "admin" {
		t.Errorf("form: %v", f)
	}
	p := string(sink.attr(t, "payload").Any().(json.RawMessage))
	if want := `{"token":"***","nested":{"password":"***"},"name":"host"}`; p != want {
		t.Errorf("payload: %s", p)
	}
	if string(payload) == p {
		t.Error("source payload mutated")
	}
	for key, want := range map[string]string{
		"response": "token: *** ok",
		"url":      "http://host/api?apikey=***",
		"secret":   MaskValue,
		"error":    "bad Authorization: Basic ***",
	} {
		if got := sink.attr(t, key).String(); got != want {
			t.Errorf("%s: %q, want %q", key, got, want)
		}
	}
	if got := sink.attr(t, "status").Int64(); got != 401 {
		t.Errorf("status: %d", got)
	}
	if got := sink.attr(t, "password").String(); got != MaskValue {
		t.Errorf("secret group: %q", got)
	}
	fv := sink.attr(t, "formvalues").Any().(map[string]string)
	if fv["password"] != MaskValue || fv["user"] != "admin" {
		t.Errorf("formvalues: %v", fv)
	}
	if got := sink.attr(t, "nilerr").String(); got != "<nil>" {
		t.Errorf("nilerr: %q", got)
	}
	group := sink.attr(t, "auth").Group()
	if group[0].Value.String() != MaskValue || group[1].Value.String() != "admin" {
		t.Errorf("group: %v", group)
	}
}

func TestXLoggerFollowsLogger(t *testing.T) {
	first := &captureHandler{level: slog.LevelInfo}
	withSink(t, first)
	l := XLogger.With(slog.String("token", "tok"))

	second := &captureHandler{level: slog.LevelInfo}
	Logger = slog.New(second)
	l.Info("hello")

	if first.last != nil {
		t.Error("record went to the replaced logger")
	}
	if second.last == nil {
		t.Fatal("record not handled by the current logger")
	}
	// WithAttrs is replayed onto the current sink, masked
	if len(second.attrs) != 1 || second.attrs[0].Value.String() != MaskValue {
		t.Errorf("sink attrs: %v", second.attrs)
	}
}

func TestXLoggerDisabled(t *testing.T) {
	sink := &captureHandler{level: slog.LevelInfo}
	withSink(t, sink)
	if XLogger.Enabled(context.Background(), slog.LevelDebug) {
		t.Error("debug enabled")
	}
	XLogger.Debug("password=x")
	if sink.last != nil {
		t.Error("disabled record handled")
	}
}

func TestMaskString(t *testing.T) {
	for _, tc := range []struct{ in, want string }{
		{`password=abc`, `password=***`},
		{`token: abc def`, `token: *** def`},
		{`Authorization: Bearer abc`, `Authorization: Bearer ***`},
		{`Authorization Bearer abc`, `Authorization Bearer ***`},
		{`authorization basic abc`, `authorization basic ***`},
		{`{"Gwos-Api-Token":["abc"],"name":"n"}`, `{"Gwos-Api-Token":"***","name":"n"}`},
		{`Get "https://admin:s3cr3t@gw.example/api": EOF`, `Get "https://admin:***@gw.example/api": EOF`},
		{`https://:s3cr3t@gw.example`, `https://:***@gw.example`},
		// prose and plain URLs are kept
		{`authorization failed`, `authorization failed`},
		{`could not set property password on host1`, `could not set property password on host1`},
		{`secret store unavailable`, `secret store unavailable`},
		{`http://gw.example:8080/api?x=a@b`, `http://gw.example:8080/api?x=a@b`},
		{`http://gw:8080?email=a@b.com`, `http://gw:8080?email=a@b.com`},
		{`http://gw:8080#a@b`, `http://gw:8080#a@b`},
		{`token abc123`, `token abc123`},
		{`mailto user@host`, `mailto user@host`},
	} {
		if got := MaskString(tc.in); got != tc.want {
			t.Errorf("MaskString(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

type nilErr struct{ msg string }

func (e *nilErr) Error() string { return e.msg }
