package clients

import (
	"bytes"
	"compress/gzip"
	"context"
	"crypto/tls"
	"io"
	"net/http"
	"net/url"

	"github.com/gwos/tcg/sdk/logper"
)

var httpClient = &http.Client{
	Transport: &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	},
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
	headers map[string]string, formValues map[string]string, byteBody []byte) (int, []byte, error) {
	return SendRequestWithContext(context.Background(), httpMethod, requestURL, headers, formValues, byteBody)
}

// SendRequestWithContext wraps HTTP methods
func SendRequestWithContext(ctx context.Context, httpMethod string, requestURL string,
	headers map[string]string, formValues map[string]string, byteBody []byte) (int, []byte, error) {

	var request *http.Request
	var response *http.Response
	var err error

	urlValues := url.Values{}
	if formValues != nil {
		for k, v := range formValues {
			urlValues.Add(k, v)
		}
		byteBody = []byte(urlValues.Encode())
	}

	if headers["Content-Encoding"] == "gzip" {
		ctx, byteBody, err = GZIP(ctx, byteBody)
		if err != nil {
			return -1, nil, err
		}
	}

	var body io.Reader
	if byteBody != nil {
		body = bytes.NewBuffer(byteBody)
	}

	request, err = http.NewRequestWithContext(ctx, httpMethod, requestURL, body)
	if err != nil {
		return -1, nil, err
	}

	request.Header.Set("Connection", "close")
	for k, v := range headers {
		request.Header.Add(k, v)
	}

	_, request = HookRequestContext(ctx, request)
	response, err = httpClient.Do(request)
	if err != nil {
		return -1, nil, err
	}

	defer response.Body.Close()
	responseBody, err := io.ReadAll(response.Body)
	if err != nil {
		return -1, nil, err
	}
	return response.StatusCode, responseBody, nil
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
}

// Send sends request
func (q *Req) Send() (*Req, error) {
	return q.SendWithContext(context.Background())
}

// SendWithContext sends request
func (q *Req) SendWithContext(ctx context.Context) (*Req, error) {
	status, response, err := SendRequestWithContext(
		ctx,
		q.Method,
		q.URL,
		q.Headers,
		q.Form,
		q.Payload,
	)
	q.Err = err
	q.Response = response
	q.Status = status
	return q, err
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
			if bytes.HasPrefix(q.Payload, []byte(`{`)) {
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
		if bytes.HasPrefix(q.Payload, []byte(`{`)) {
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
