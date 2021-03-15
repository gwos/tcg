package main

import (
	"github.com/gwos/tcg/clients"
	"github.com/gwos/tcg/log"
	"net/http"
)

var logoutUrl = "/api/auth/logout"

func logout(host, gwosAppName, gwosApiToken string) {
	formValues := map[string]string{
		"gwos-app-name":  gwosAppName,
		"gwos-api-token": gwosApiToken,
	}

	headers["Content-Type"] = "application/x-www-form-urlencoded"

	statusCode, body, err := clients.SendRequest(http.MethodPost, host+logoutUrl, headers, formValues, nil)
	if err != nil {
		log.Error(err.Error())
	}

	log.Warn(statusCode, string(body))
}
