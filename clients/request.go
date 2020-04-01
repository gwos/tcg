package clients

import (
	"bytes"
	"crypto/tls"
	"io/ioutil"
	"net/http"
	"net/url"
)

// SendRequest wraps HTTP methods
func SendRequest(httpMethod string, requestURL string, headers map[string]string, formValues map[string]string,
	byteBody []byte) (int, []byte, error) {
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

	request, err = http.NewRequest(httpMethod, requestURL, bytes.NewBuffer(byteBody))
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
