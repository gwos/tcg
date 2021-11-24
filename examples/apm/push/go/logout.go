package main

import (
	"net/http"

	"github.com/gwos/tcg/sdk/clients"
	"github.com/rs/zerolog/log"
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
		log.Err(err).Msg("could not send request")
	}
	log.Warn().
		Int("statusCode", statusCode).
		Msg(string(body))
}
