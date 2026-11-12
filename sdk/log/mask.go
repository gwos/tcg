package log

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/url"
	"regexp"
	"slices"
)

// MaskValue replaces masked secrets
const MaskValue = "***"

const (
	secretKeys  = `password|token|secret|authorization|api[-_]?key|cookie`
	authSchemes = `basic|bearer|digest|negotiate`
)

var (
	// secretKeyRe matches an attr, header or form key ending in a secret keyword
	secretKeyRe = regexp.MustCompile(`(?i)(?:` + secretKeys + `)$`)

	// maskRules are applied to raw text in order
	maskRules = []struct {
		re   *regexp.Regexp
		repl []byte
	}{
		// (1) Structured JSON: a key ending in a secret keyword,
		// then its value (string | array | number | bool | null) -> "***".
		// The key is fully bracketed so a match can't span into the next field.
		// Note: a nested array value is cut at its first ']'.
		{regexp.MustCompile(`("[^"]*(?i:` + secretKeys + `)"\s*:\s*)` +
			`("(?:[^"\\]|\\.)*"|\[[^\]]*\]|true|false|null|-?\d[\d.eE+-]*)`), []byte(`${1}"` + MaskValue + `"`)},
		// (2) URL userinfo: scheme://user:password@ -> scheme://user:***@
		// Path, query and fragment chars end the match, so a later '@' is kept.
		{regexp.MustCompile(`(://[^/?#\s:@"']*:)[^/?#\s@"']+@`), []byte(`${1}` + MaskValue + `@`)},
		// (3) Free text: a secret keyword, a ':' or '=' separator, then the value run.
		// A bare space separates only before an auth scheme (Authorization Bearer x),
		// so prose like "authorization failed" is kept.
		// An auth scheme (Basic, Bearer, Digest, Negotiate) is kept and the credential after it is masked.
		// The value run excludes whitespace, JSON structural chars (" ' , : { } [ ]),
		// '\' and '&', so masking can't break a JSON escape or drop the next query param.
		// Note: an object-valued secret ("password":{...}) is not handled.
		{regexp.MustCompile(`(?i)(` + secretKeys + `)(\s*[:=]\s*(?:(?:` + authSchemes + `)\s+)?|\s+(?:` + authSchemes + `)\s+)([^\s"',:{}\[\]&\\]+)`), []byte(`${1}${2}` + MaskValue)},
	}
)

// MaskBytes returns p with secrets masked
func MaskBytes(p []byte) []byte {
	for _, rule := range maskRules {
		p = rule.re.ReplaceAll(p, rule.repl)
	}
	return p
}

// MaskString returns s with secrets masked
func MaskString(s string) string {
	return string(MaskBytes([]byte(s)))
}

// maskHandler masks secrets in records and forwards them to Logger.Handler().
// The sink is looked up on each call because consumers reassign Logger after init.
type maskHandler struct {
	// wraps replays WithAttrs/WithGroup calls onto the sink, in order
	wraps []func(slog.Handler) slog.Handler
}

func (h *maskHandler) sink() slog.Handler {
	s := Logger.Handler()
	for _, wrap := range h.wraps {
		s = wrap(s)
	}
	return s
}

func (h *maskHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return Logger.Handler().Enabled(ctx, level)
}

func (h *maskHandler) Handle(ctx context.Context, r slog.Record) error {
	m := slog.NewRecord(r.Time, r.Level, MaskString(r.Message), r.PC)
	r.Attrs(func(a slog.Attr) bool {
		m.AddAttrs(maskAttr(a))
		return true
	})
	return h.sink().Handle(ctx, m)
}

func (h *maskHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	masked := make([]slog.Attr, len(attrs))
	for i, a := range attrs {
		masked[i] = maskAttr(a)
	}
	return h.with(func(s slog.Handler) slog.Handler { return s.WithAttrs(masked) })
}

func (h *maskHandler) WithGroup(name string) slog.Handler {
	return h.with(func(s slog.Handler) slog.Handler { return s.WithGroup(name) })
}

func (h *maskHandler) with(wrap func(slog.Handler) slog.Handler) *maskHandler {
	return &maskHandler{wraps: append(slices.Clip(h.wraps), wrap)}
}

func maskAttr(a slog.Attr) slog.Attr {
	a.Value = a.Value.Resolve()
	if secretKeyRe.MatchString(a.Key) {
		return slog.String(a.Key, MaskValue)
	}
	switch a.Value.Kind() {
	case slog.KindString:
		return slog.String(a.Key, MaskString(a.Value.String()))
	case slog.KindGroup:
		group := a.Value.Group()
		masked := make([]slog.Attr, len(group))
		for i, ga := range group {
			masked[i] = maskAttr(ga)
		}
		return slog.Attr{Key: a.Key, Value: slog.GroupValue(masked...)}
	case slog.KindAny:
		switch v := a.Value.Any().(type) {
		case json.RawMessage:
			return slog.Any(a.Key, json.RawMessage(MaskBytes(slices.Clone(v))))
		case []byte:
			return slog.Any(a.Key, MaskBytes(slices.Clone(v)))
		case http.Header:
			return slog.Any(a.Key, http.Header(maskMultiMap(v)))
		case url.Values:
			return slog.Any(a.Key, url.Values(maskMultiMap(v)))
		case map[string][]string:
			return slog.Any(a.Key, maskMultiMap(v))
		case map[string]string:
			return slog.Any(a.Key, maskMap(v))
		case error:
			return slog.String(a.Key, MaskString(errorText(v)))
		}
	}
	return a
}

// errorText returns err.Error(), or "<nil>" for a nil pointer error that panics,
// as slog handlers do
func errorText(err error) (s string) {
	defer func() {
		if recover() != nil {
			s = "<nil>"
		}
	}()
	return err.Error()
}

// maskKV returns v masked as a whole under a secret key k, otherwise masked as text
func maskKV(k, v string) string {
	if secretKeyRe.MatchString(k) {
		return MaskValue
	}
	return MaskString(v)
}

// maskMap returns a copy of m with values masked by maskKV
func maskMap(m map[string]string) map[string]string {
	res := make(map[string]string, len(m))
	for k, v := range m {
		res[k] = maskKV(k, v)
	}
	return res
}

// maskMultiMap returns a copy of m with values masked by maskKV
func maskMultiMap(m map[string][]string) map[string][]string {
	res := make(map[string][]string, len(m))
	for k, vv := range m {
		masked := make([]string, len(vv))
		for i, v := range vv {
			masked[i] = maskKV(k, v)
		}
		res[k] = masked
	}
	return res
}
