package clients

import (
	"bytes"
	"context"
	"crypto/tls"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptrace"
	"net/url"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"go.opentelemetry.io/contrib/instrumentation/net/http/httptrace/otelhttptrace"
)

var httpClient = &http.Client{
	Transport: &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	},
}

// SendRequest wraps HTTP methods
func SendRequest(httpMethod string, requestURL string,
	headers map[string]string, formValues map[string]string, byteBody []byte) (int, []byte, error) {
	return SendRequestWithContext(nil, httpMethod, requestURL, headers, formValues, byteBody)
}

// SendRequestWithContext wraps HTTP methods
func SendRequestWithContext(ctx context.Context, httpMethod string, requestURL string,
	headers map[string]string, formValues map[string]string, byteBody []byte) (int, []byte, error) {

	var request *http.Request
	var response *http.Response
	var err error

	urlValues := url.Values{}
	if formValues != nil {
		for key, value := range formValues {
			urlValues.Add(key, value)
		}
		byteBody = []byte(urlValues.Encode())
	}

	var body io.Reader
	if byteBody != nil {
		body = bytes.NewBuffer(byteBody)
	} else {
		body = nil
	}

	if ctx == nil {
		request, err = http.NewRequest(httpMethod, requestURL, body)
	} else {
		ctx = httptrace.WithClientTrace(ctx, otelhttptrace.NewClientTrace(ctx))
		request, err = http.NewRequestWithContext(ctx, httpMethod, requestURL, body)
	}
	if err != nil {
		return -1, nil, err
	}

	request.Header.Set("Connection", "close")
	if headers != nil {
		for key, value := range headers {
			request.Header.Add(key, value)
		}
	}

	response, err = httpClient.Do(request)
	if err != nil {
		return -1, nil, err
	}

	defer response.Body.Close()

	responseBody, err := ioutil.ReadAll(response.Body)
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

// LogDetailsWith adds data to log event
// if argument is nil creates Info or Error depending on Err
func (q Req) LogDetailsWith(e *zerolog.Event) *zerolog.Event {
	switch {
	case e == nil && q.Err != nil:
		e = log.Error()
	case e == nil && q.Status >= 400:
		e = log.Warn()
	case e == nil:
		e = log.Info()
	}
	e.Err(q.Err).
		Int("status", q.Status).
		Str("method", q.Method).
		Str("url", q.URL)
	if len(q.Headers) > 0 {
		e.Interface("headers", q.Headers)
	}
	if len(q.Form) > 0 {
		e.Interface("form", q.Form)
	}
	if len(q.Payload) > 0 {
		if bytes.HasPrefix(q.Payload, []byte(`{`)) {
			e.RawJSON("payload", q.Payload)
		} else {
			e.Bytes("payload", q.Payload)
		}
	}
	if len(q.Response) > 0 {
		if bytes.HasPrefix(q.Response, []byte(`{`)) {
			e.RawJSON("response", q.Response)
		} else {
			e.Bytes("response", q.Response)
		}
	}
	return e
}

// LogWith adds data to log event
// if argument is nil creates Info or Error depending on Err
// add details on Status >= 400 or Debug
func (q Req) LogWith(e *zerolog.Event) *zerolog.Event {
	switch {
	case e == nil && q.Err != nil:
		e = log.Error()
	case e == nil && q.Status >= 400:
		e = log.Warn()
	case e == nil:
		e = log.Info()
	}
	e.Err(q.Err).
		Int("status", q.Status).
		Str("method", q.Method).
		Str("url", q.URL)
	if q.Status >= 400 || zerolog.GlobalLevel() <= zerolog.DebugLevel {
		if len(q.Headers) > 0 {
			e.Interface("headers", q.Headers)
		}
		if len(q.Form) > 0 {
			e.Interface("form", q.Form)
		}
		if len(q.Payload) > 0 {
			if bytes.HasPrefix(q.Payload, []byte(`{`)) {
				e.RawJSON("payload", q.Payload)
			} else {
				e.Bytes("payload", q.Payload)
			}
		}
		if len(q.Response) > 0 {
			if bytes.HasPrefix(q.Response, []byte(`{`)) {
				e.RawJSON("response", q.Response)
			} else {
				e.Bytes("response", q.Response)
			}
		}
	}
	return e
}

// Send sends request
func (q *Req) Send() (*Req, error) {
	return q.SendWithContext(nil)
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
