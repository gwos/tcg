package main

import (
	"fmt"
	"net/http"

	"github.com/gwos/tcg/sdk/clients"
)

var loginURL = "/api/auth/login"

type credentials struct {
	gwosAppName  string
	gwosAPIToken string
}

func login(host, user, password, gwosAppName string) (*credentials, error) {
	formValues := map[string]string{
		"user":          user,
		"password":      password,
		"gwos-app-name": gwosAppName,
	}
	headers := map[string]string{
		"Content-Type": "application/x-www-form-urlencoded",
	}

	statusCode, body, err := clients.SendRequest(http.MethodPost, host+loginURL, headers, formValues, nil)
	if err != nil {
		return nil, err
	}
	if statusCode != 200 {
		return nil, fmt.Errorf("[ERROR]: Http request failed. [Status code]: %d, [Response]: %s",
			statusCode, string(body))
	}

	return &credentials{
		gwosAppName:  gwosAppName,
		gwosAPIToken: string(body),
	}, nil
}
