package clients

import (
	"bytes"
	"context"
	"crypto/tls"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
)

// SendRequest wraps HTTP methods
func SendRequest(httpMethod string, requestURL string,
	headers map[string]string, formValues map[string]string, byteBody []byte) (int, []byte, error) {
	return SendRequestWithContext(nil, httpMethod, requestURL, headers, formValues, byteBody)
}

// SendRequestWithContext wraps HTTP methods
func SendRequestWithContext(ctx context.Context, httpMethod string, requestURL string,
	headers map[string]string, formValues map[string]string, byteBody []byte) (int, []byte, error) {

	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}

	client := http.Client{Transport: tr}

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

	request, err = http.NewRequest(httpMethod, requestURL, body)
	if err != nil {
		return -1, nil, err
	}
	request.Header.Set("Connection", "close")

	if headers != nil {
		for key, value := range headers {
			request.Header.Add(key, value)
		}
	}

	response, err = client.Do(request)
	if err != nil {
		return -1, nil, err
	}

	responseBody, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return -1, nil, err
	}
	defer response.Body.Close()
	return response.StatusCode, responseBody, nil
}

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
