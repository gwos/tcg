package clients

import (
	"bytes"
	"compress/gzip"
	"context"
	"crypto/tls"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"time"

	"github.com/gwos/tcg/sdk/logper"
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

var GZIP = func(ctx context.Context, p []byte) (context.Context, []byte, error) {
	buf := &bytes.Buffer{}
	gw := gzip.NewWriter(buf)
	_, err := gw.Write(p)
	_ = gw.Close()
	return ctx, buf.Bytes(), err
}

// SendRequest wraps HTTP methods
func SendRequest(httpMethod string, requestURL string,
	headers map[string]string, formValues map[string]string, body []byte) (int, []byte, error) {
	return SendRequestWithContext(context.Background(), httpMethod, requestURL, headers, formValues, body)
}

// SendRequestWithContext wraps HTTP methods
func SendRequestWithContext(ctx context.Context, httpMethod string, requestURL string,
	headers map[string]string, formValues map[string]string, body []byte) (int, []byte, error) {

	req, err := (&Req{
		URL:     requestURL,
		Method:  httpMethod,
		Headers: headers,
		Form:    formValues,
		Payload: body,
	}).SendWithContext(ctx)
	return req.Status, req.Response, err
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

	client *http.Client
}

// SetClient sets http.Client to use
func (q *Req) SetClient(c *http.Client) *Req {
	q.client = c
	return q
}

// Send sends request
func (q *Req) Send() (*Req, error) {
	return q.SendWithContext(context.Background())
}

// SendWithContext sends request
func (q *Req) SendWithContext(ctx context.Context) (*Req, error) {
	var (
		body     = q.Payload
		err      error
		request  *http.Request
		response *http.Response
	)

	urlValues := url.Values{}
	if q.Form != nil {
		for k, v := range q.Form {
			urlValues.Add(k, v)
		}
		body = []byte(urlValues.Encode())
	}

	var bodyBuf io.Reader
	if body != nil {
		bodyBuf = bytes.NewBuffer(body)
	}
	request, err = http.NewRequestWithContext(ctx, q.Method, q.URL, bodyBuf)
	if err != nil {
		q.Status, q.Err = -1, err
		return q, err
	}
	request.Header.Set("Connection", "close")
	for k, v := range q.Headers {
		request.Header.Add(k, v)
	}
	_, request = HookRequestContext(ctx, request)

	if q.client != nil {
		response, err = q.client.Do(request)
	} else {
		response, err = HttpClient.Do(request)
	}
	if err != nil {
		q.Status, q.Err = -1, err
		return q, err
	}

	defer response.Body.Close()
	responseBody, err := io.ReadAll(response.Body)
	if err != nil {
		q.Status, q.Err = -1, err
		return q, err
	}
	q.Status, q.Response = response.StatusCode, responseBody
	return q, nil
}

// LogFields returns fields maps
func (q Req) LogFields() (fields map[string]interface{}, rawJSON map[string][]byte) {
	rawJSON = map[string][]byte{}
	fields = map[string]interface{}{
		"url":    q.URL,
		"method": q.Method,
		"status": q.Status,
	}
	if q.Err != nil {
		fields["error"] = q.Err
	}
	if q.Status >= 400 || logper.IsDebugEnabled() {
		if len(q.Headers) > 0 {
			fields["headers"] = q.Headers
		}
		if len(q.Form) > 0 {
			fields["form"] = q.Form
		}
		if len(q.Payload) > 0 {
			if v, ok := q.Headers["Content-Encoding"]; ok && v != "" {
				fields["payload"] = "encoded:" + v
			} else if bytes.HasPrefix(q.Payload, []byte(`{`)) {
				rawJSON["payload"] = q.Payload
			} else {
				fields["payload"] = string(q.Payload)
			}
		}
		if len(q.Response) > 0 {
			if bytes.HasPrefix(q.Response, []byte(`{`)) {
				rawJSON["response"] = q.Response
			} else {
				fields["response"] = string(q.Response)
			}
		}
	}
	return
}

func (q Req) Details() ReqDetails {
	return (ReqDetails)(q)
}

// ReqDetails defines an alias for logging with forced details
type ReqDetails Req

// LogFields returns fields maps
func (q ReqDetails) LogFields() (fields map[string]interface{}, rawJSON map[string][]byte) {
	rawJSON = map[string][]byte{}
	fields = map[string]interface{}{
		"url":    q.URL,
		"method": q.Method,
		"status": q.Status,
	}
	if q.Err != nil {
		fields["error"] = q.Err
	}
	if len(q.Headers) > 0 {
		fields["headers"] = q.Headers
	}
	if len(q.Form) > 0 {
		fields["form"] = q.Form
	}
	if len(q.Payload) > 0 {
		if v, ok := q.Headers["Content-Encoding"]; ok && v != "" {
			fields["payload"] = "encoded:" + v
		} else if bytes.HasPrefix(q.Payload, []byte(`{`)) {
			rawJSON["payload"] = q.Payload
		} else {
			fields["payload"] = string(q.Payload)
		}
	}
	if len(q.Response) > 0 {
		if bytes.HasPrefix(q.Response, []byte(`{`)) {
			rawJSON["response"] = q.Response
		} else {
			fields["response"] = string(q.Response)
		}
	}
	return
}
