package transit

import (
	"bytes"
	"crypto/tls"
	"io/ioutil"
	"log"
	"net/http"
)

func sendRequest(httpMethod string, url string, headers map[string]string, byteBody []byte) ([]byte, error) {
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}

	client := http.Client{Transport: tr}

	var request *http.Request
	var err error

	switch httpMethod {
	case http.MethodGet:
		request, err = http.NewRequest(http.MethodGet, url, nil)
		if err != nil {
			log.Fatal(err)
		}
	case http.MethodPost:
		request, err = http.NewRequest(http.MethodPost, url, bytes.NewBuffer(byteBody))
		if err != nil {
			log.Fatal(err)
		}
	}

	for key, value := range headers {
		request.Header.Set(key, value)
	}

	response, err := client.Do(request)
	if err != nil {
		log.Fatal(err)
	}

	defer request.Body.Close()

	responseBody, err := ioutil.ReadAll(response.Body)
	if err != nil {
		log.Fatal(err)
	}

	return responseBody, nil
}
