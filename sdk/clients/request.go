package clients

import (
	"bytes"
	"compress/gzip"
	"context"
	"crypto/tls"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	sdklog "github.com/gwos/tcg/sdk/log"
)

const (
	EnvHttpClientTimeout = "TCG_HTTP_CLIENT_TIMEOUT"
	EnvTlsClientInsecure = "TCG_TLS_CLIENT_INSECURE"

	EnvHttpClientIdleConnTimeout = "TCG_HTTP_CLIENT_IDLE_CONN_TIMEOUT"
	EnvHttpClientKeepAlive       = "TCG_HTTP_CLIENT_KEEPALIVE"
)

// httpClientKeepAlive enables HTTP connection reuse (keep-alive).
// Disabled by default to preserve historical behavior ("Connection: close" on every request);
// set TCG_HTTP_CLIENT_KEEPALIVE=true to reuse connections on the hot send path.
// IdleConnTimeout below bounds how long reused connections may sit idle
// so they can't go stale behind a load balancer.
var httpClientKeepAlive = envBool(EnvHttpClientKeepAlive)

var HttpClientTransport = &http.Transport{
	IdleConnTimeout: envDuration(time.Second*90, EnvHttpClientIdleConnTimeout), // 90s by default
	TLSClientConfig: &tls.Config{
		InsecureSkipVerify: envBool(EnvTlsClientInsecure),

		RootCAs: nil, // If RootCAs is nil, TLS uses the host's root CA set.
	},
}

var HttpClient = &http.Client{
	Timeout: envDuration(time.Second*5, EnvHttpClientTimeout), // 5s by default

	Transport: HttpClientTransport,
}

// envBool reports whether the env var holds a true boolean value
func envBool(key string) bool {
	v, err := strconv.ParseBool(os.Getenv(key))
	return err == nil && v
}

// envDuration returns the duration from the first env var in keys that holds a valid one, or def
func envDuration(def time.Duration, keys ...string) time.Duration {
	for _, key := range keys {
		if s, ok := os.LookupEnv(key); ok {
			if v, err := time.ParseDuration(s); err == nil {
				return v
			}
		}
	}
	return def
}

var HookRequestContext = func(ctx context.Context, req *http.Request) (context.Context, *http.Request) {
	return ctx, req
}

// pooledGZip is a gzip writer that writes through its own sink,
// so it is reset once per use and doesn't keep the caller's writer while pooled
type pooledGZip struct {
	gw *gzip.Writer
	w  io.Writer
}

func (pz *pooledGZip) Write(p []byte) (int, error) { return pz.w.Write(p) }

// gzipWriters reuses gzip writers, as each one allocates sizable compressor state
var gzipWriters = sync.Pool{New: func() any {
	pz := new(pooledGZip)
	pz.gw = gzip.NewWriter(pz)
	return pz
}}

// GZipTo compresses p into w with a pooled gzip writer
func GZipTo(w io.Writer, p []byte) error {
	pz := gzipWriters.Get().(*pooledGZip)
	pz.w = w
	pz.gw.Reset(pz)
	_, err := pz.gw.Write(p)
	_ = pz.gw.Close()
	pz.w = nil
	gzipWriters.Put(pz)
	return err
}

var GZip = func(ctx context.Context, w io.Writer, p []byte) (context.Context, error) {
	return ctx, GZipTo(w, p)
}

// IsGZipped detects if payload was compressed with gzip
// by magic number: 1st byte is 0x1f and 2nd is 0x8b
func IsGZipped(p []byte) bool {
	return len(p) > 2 && p[0] == 31 && p[1] == 139
}

// IsJSON does quick check for JSON-like content
func IsJSON(s []byte) bool {
	return len(s) >= 2 &&
		((s[0] == '{' && s[len(s)-1] == '}') ||
			(s[0] == '[' && s[len(s)-1] == ']'))
}

// SendRequest wraps HTTP methods
func SendRequest(httpMethod string, requestURL string,
	headers map[string]string, formValues map[string]string, body []byte) (int, []byte, error) {
	return SendRequestWithContext(context.Background(), httpMethod, requestURL, headers, formValues, body)
}

// SendRequestWithContext wraps HTTP methods
func SendRequestWithContext(ctx context.Context, httpMethod string, requestURL string,
	headers map[string]string, formValues map[string]string, body []byte) (int, []byte, error) {

	req := Req{
		URL:     requestURL,
		Method:  httpMethod,
		Headers: headers,
		Form:    formValues,
		Payload: body,
	}
	_ = req.SendWithContext(ctx)
	return req.Status, req.Response, req.Err
}

// BuildQueryParams makes the query parameters string
func BuildQueryParams(params map[string]string) string {
	if len(params) == 0 {
		return ""
	}
	values := make(url.Values, len(params))
	for k, v := range params {
		values.Set(k, v)
	}
	return "?" + values.Encode()
}

// Req defines request context
type Req struct {
	Err      error
	Form     map[string]string
	Headers  map[string]string
	Method   string
	Payload  []byte
	Response []byte
	Status   int
	URL      string

	client   *http.Client
	duration time.Duration
	header   http.Header
}

// SetClient sets http.Client to use
func (q *Req) SetClient(c *http.Client) *Req {
	q.client = c
	return q
}

// Send sends request
func (q *Req) Send() error {
	return q.SendWithContext(context.Background())
}

// SendWithContext sends request
func (q *Req) SendWithContext(ctx context.Context) error {
	var (
		body     io.Reader
		err      error
		request  *http.Request
		response *http.Response
	)

	if q.Form != nil {
		urlValues := url.Values{}
		for k, v := range q.Form {
			urlValues.Add(k, v)
		}
		body = strings.NewReader(urlValues.Encode())
	} else if q.Payload != nil {
		body = bytes.NewReader(q.Payload)
	}

	request, err = http.NewRequestWithContext(ctx, q.Method, q.URL, body)
	if err != nil {
		q.Status, q.Err = -1, err
		return err
	}
	if h, ok := HeaderFromCtx(ctx); ok {
		// clone values too, so adding request headers never writes into the ctx header
		request.Header = h.Clone()
	}
	if !httpClientKeepAlive {
		request.Header.Set("Connection", "close")
	}
	for k, v := range q.Headers {
		request.Header.Add(k, v)
	}
	_, request = HookRequestContext(ctx, request)

	t0 := time.Now()
	if q.client != nil {
		response, err = q.client.Do(request)
	} else {
		response, err = HttpClient.Do(request)
	}
	// taking data for logging
	q.header = request.Header
	q.duration = time.Since(t0).Truncate(time.Millisecond)
	if err != nil {
		q.Status, q.Err = -1, err
		return err
	}

	defer response.Body.Close()
	responseBody, err := io.ReadAll(response.Body)
	if err != nil {
		q.Status, q.Err = -1, err
		return err
	}
	q.Status, q.Response = response.StatusCode, responseBody
	return nil
}

func (q Req) Details() []slog.Attr {
	return q.logAttrs(true)
}

func (q Req) LogAttrs() []slog.Attr {
	return q.logAttrs(false)
}

func (q Req) logAttrs(forceDetails bool) []slog.Attr {
	attrs := []slog.Attr{
		slog.String("url", q.URL),
		slog.String("method", q.Method),
		slog.Int("status", q.Status),
		slog.Duration("duration", q.duration),
	}
	if q.Err != nil {
		attrs = append(attrs, slog.String("error", q.Err.Error()))
	}
	if q.Status >= 400 || forceDetails ||
		sdklog.Logger.Enabled(context.Background(), slog.LevelDebug) {
		if len(q.header) > 0 {
			attrs = append(attrs, slog.Any("header", q.header))
		}

		if len(q.Form) > 0 {
			attrs = append(attrs, slog.Any("form", q.Form))
		}

		if p := bytes.TrimSpace(q.Payload); IsGZipped(p) {
			attrs = append(attrs, slog.String("payload", "encoded:gzip"))
		} else if IsJSON(p) {
			attrs = append(attrs, slog.Any("payload", json.RawMessage(p)))
		} else {
			attrs = append(attrs, slog.String("payload", string(p)))
		}

		if p := bytes.TrimSpace(q.Response); IsJSON(p) {
			attrs = append(attrs, slog.Any("response", json.RawMessage(p)))
		} else {
			attrs = append(attrs, slog.String("response", string(p)))
		}
	}

	// TODO: prepare wrapped attrs to avoid unnecessary work in disabled log calls.
	// https://pkg.go.dev/log/slog#hdr-Performance_considerations
	return attrs
}
