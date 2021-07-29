package clients

import (
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/gwos/tcg/config"
	"github.com/rs/zerolog/log"
)

// DSOperations defines DalekServices operations interface
type DSOperations interface {
	Reload(agentID string) error
	ValidateToken(appName, apiToken string) error
}

// Define entrypoints for DSOperations
const (
	DSEntrypointReload        = "/dalekservices/connectors/reload/:agentID"
	DSEntrypointValidateToken = "/dalekservices/validate-token"
)

// DSClient implements DSOperations interface
type DSClient struct {
	*config.DSConnection
}

// ValidateToken implements DSOperations.ValidateToken.
func (client *DSClient) ValidateToken(appName, apiToken string, dalekServicesURL string) error {
	if len(client.HostName) == 0 {
		log.Info().Msg("omit ValidateToken on demand config")
		return nil
	}

	headers := map[string]string{
		"Accept":       "text/plain",
		"Content-Type": "application/x-www-form-urlencoded",
	}
	formValues := map[string]string{
		"gwos-app-name":  appName,
		"gwos-api-token": apiToken,
	}
	if dalekServicesURL == "" {
		dalekServicesURL = client.DSConnection.HostName
	}
	entrypoint := url.URL{
		Scheme: makeDalekServicesScheme(client.DSConnection.HostName),
		Host:   dalekServicesURL,
		Path:   DSEntrypointValidateToken,
	}
	req, err := (&Req{
		URL:     entrypoint.String(),
		Method:  http.MethodPost,
		Headers: headers,
		Form:    formValues,
	}).Send()

	if err == nil {
		if req.Status == 201 {
			if b, e := strconv.ParseBool(string(req.Response)); e == nil && b {
				req.LogWith(log.Debug()).Msg("validate token")
				return nil
			}
			eee := fmt.Errorf("invalid gwos-app-name or gwos-api-token")
			req.LogWith(log.Warn()).Err(eee).Msg("could not validate token")
			return eee
		}
		eee := fmt.Errorf(string(req.Response))
		req.LogDetailsWith(log.Warn()).Err(eee).Msg("could not validate token")
		return eee
	}
	req.LogWith(log.Warn()).Msg("could not validate token")
	return err
}

// Reload implements DSOperations.Reload.
func (client *DSClient) Reload(agentID string) error {
	headers := map[string]string{
		"Accept":       "application/json",
		"Content-Type": "application/json",
	}
	entrypoint := url.URL{
		Scheme: makeDalekServicesScheme(client.DSConnection.HostName),
		Host:   client.DSConnection.HostName,
		Path:   strings.ReplaceAll(DSEntrypointReload, ":agentID", agentID),
	}
	req, err := (&Req{
		URL:     entrypoint.String(),
		Method:  http.MethodPost,
		Headers: headers,
	}).Send()

	if err == nil {
		if req.Status == 201 {
			req.LogWith(log.Info()).Msg("request for reload")
			return nil
		}
		eee := fmt.Errorf(string(req.Response))
		if req.Status == 404 {
			req.LogWith(log.Warn()).Err(eee).Msg("could not request for reload: check AgentID")
		}
		req.LogDetailsWith(log.Warn()).Err(eee).Msg("could not request for reload")
		return eee
	}
	req.LogWith(log.Warn()).Msg("could not request for reload")
	return err
}

// Create the scheme (http or https) based on hostName prefix
func makeDalekServicesScheme(hostName string) string {
	scheme := "https"
	if strings.HasPrefix(hostName, "dalekservices") {
		scheme = "http"
	}
	return scheme
}
