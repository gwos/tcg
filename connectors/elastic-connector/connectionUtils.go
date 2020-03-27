package main

import (
	"crypto/tls"
	"github.com/gwos/tng/log"
	"io"
	"io/ioutil"
	"net/http"
)

var client *http.Client

func initClient() {
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client = &http.Client{Transport: tr}
}

func executeRequest(method string, path string, body io.Reader, headers map[string]string) ([]byte, bool) {
	if client == nil {
		initClient()
	}
	if client == nil {
		log.Error("Could not create http client. ")
		return nil, false
	}

	var request *http.Request
	var response *http.Response
	var responseBody []byte
	var err error

	request, err = http.NewRequest(method, path, body)
	if err != nil {
		log.Error("Error creating request: ", err)
	}
	if request == nil {
		log.Error("Request is nil.")
		return nil, false
	}

	if headers != nil {
		for key, value := range headers {
			request.Header.Add(key, value)
		}
	}

	response, err = client.Do(request)
	if err != nil {
		log.Error("Error getting response: ", err)
	}
	if response == nil {
		log.Error("Response is nil.")
		return nil, false
	}
	successful := true
	if response.StatusCode != 200 {
		log.Error("Failure response code: ", response.StatusCode, " status: ", response.Status)
		successful = false
	}
	responseBody, err = ioutil.ReadAll(response.Body)
	if err != nil {
		log.Error("Error reading response: ", err)
	}
	err = response.Body.Close()
	if err != nil {
		log.Error("Error closing response: ", err)
	}

	return responseBody, successful
}
