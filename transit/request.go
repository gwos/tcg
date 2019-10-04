package transit

import (
	"bytes"
	"crypto/tls"
	"io/ioutil"
	"net/http"
	"net/url"
)

func sendRequest(httpMethod string, requestUrl string, headers map[string]string, formValues map[string]string,
	byteBody []byte) (int, []byte, error) {
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}

	client := http.Client{Transport: tr}

	var request *http.Request
	var err error

	urlValues := url.Values{}
	if formValues != nil {
		for key, value := range formValues {
			urlValues.Add(key, value)
		}
		byteBody = []byte(urlValues.Encode())
	}

	switch httpMethod {
	case http.MethodGet:
		request, err = http.NewRequest(http.MethodGet, requestUrl, nil)
		if err != nil {
			return -1, nil, err
		}
	case http.MethodPost:
		request, err = http.NewRequest(http.MethodPost, requestUrl, bytes.NewBuffer(byteBody))
		if err != nil {
			return -1, nil, err
		}
	}

	for key, value := range headers {
		request.Header.Add(key, value)
	}

	response, err := client.Do(request)
	if err != nil {
		return -1, nil, err
	}

	defer request.Body.Close()

	responseBody, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return -1, nil, err
	}

	return response.StatusCode, responseBody, nil
}