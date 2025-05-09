package clients

import (
	"bytes"
	"compress/gzip"
	"context"
	"crypto/tls"
	"encoding/json"
	"io"
	"log/slog"
	"maps"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"time"

	sdklog "github.com/gwos/tcg/sdk/log"
)

const (
	EnvHttpClientTimeout = "TCG_HTTP_CLIENT_TIMEOUT"
	EnvTlsClientInsecure = "TCG_TLS_CLIENT_INSECURE"
)

var HttpClientTransport = &http.Transport{
	TLSClientConfig: &tls.Config{
		InsecureSkipVerify: func(env string) bool {
			v, err := strconv.ParseBool(os.Getenv(env))
			return err == nil && v
		}(EnvTlsClientInsecure),

		RootCAs: nil, // If RootCAs is nil, TLS uses the host's root CA set.
	},
}

var HttpClient = &http.Client{
	Timeout: func(env string) time.Duration {
		if s, ok := os.LookupEnv(env); ok {
			if v, err := time.ParseDuration(s); err == nil {
				return v
			}
		}
		return time.Duration(5 * time.Second)
	}(EnvHttpClientTimeout),

	Transport: HttpClientTransport,
}

var HookRequestContext = func(ctx context.Context, req *http.Request) (context.Context, *http.Request) {
	return ctx, req
}

var GZip = func(ctx context.Context, w io.Writer, p []byte) (context.Context, error) {
	gw := gzip.NewWriter(w)
	_, err := gw.Write(p)
	_ = gw.Close()
	return ctx, err
}

// IsGZipped detects if payload was compressed with gzip
// by magic number: 1st byte is 0x1f and 2nd is 0x8b
func IsGZipped(p []byte) bool {
	return len(p) > 2 && p[0] == 31 && p[1] == 139
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
	var query string
	for paramName, paramValue := range params {
		query = appendSeparator(query) + url.QueryEscape(paramName) + "=" + url.QueryEscape(paramValue)
	}
	return query
}

func appendSeparator(params string) string {
	if params != "" {
		params = params + "&"
	} else {
		params = "?"
	}
	return params
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
		body = bytes.NewBuffer([]byte(urlValues.Encode()))
	} else if q.Payload != nil {
		body = bytes.NewBuffer(q.Payload)
	}

	request, err = http.NewRequestWithContext(ctx, q.Method, q.URL, body)
	if err != nil {
		q.Status, q.Err = -1, err
		return err
	}
	if h, ok := HeaderFromCtx(ctx); ok {
		maps.Copy(request.Header, h)
	}
	request.Header.Set("Connection", "close")
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
	q.duration = time.Since(t0).Truncate(1 * time.Millisecond)
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
			attrs = append(attrs, slog.Any("header", slog.AnyValue(q.header)))
		}
		if len(q.Form) > 0 {
			attrs = append(attrs, slog.Any("form", q.Form))
		}
		if len(q.Payload) > 0 {
			if IsGZipped(q.Payload) {
				attrs = append(attrs, slog.String("payload", "encoded:gzip"))
			} else if bytes.HasPrefix(q.Payload, []byte(`{`)) {
				attrs = append(attrs, slog.Any("payload", json.RawMessage(q.Payload)))
			} else {
				attrs = append(attrs, slog.String("payload", string(q.Payload)))
			}
		}
		if len(q.Response) > 0 {
			if bytes.HasPrefix(q.Response, []byte(`{`)) {
				attrs = append(attrs, slog.Any("response", json.RawMessage(q.Response)))
			} else {
				attrs = append(attrs, slog.String("response", string(q.Response)))
			}
		}
	}

	// TODO: prepare wrapped attrs to avoid unnecessary work in disabled log calls.
	// https://pkg.go.dev/log/slog#hdr-Performance_considerations
	return attrs
}
