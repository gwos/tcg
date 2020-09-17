package main

import "net/http"

type CustomHttpDoer struct {
	f func(*http.Request) (*http.Response, error)
}

func (d CustomHttpDoer) Do(request *http.Request) (*http.Response, error) {
	client := http.Client{}

	request.Header.Set("Connection", "close")

	for key, value := range headers {
		request.Header.Add(key, value)
	}

	return client.Do(request)
}
