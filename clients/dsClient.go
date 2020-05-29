package clients

import (
	"fmt"
	"github.com/gwos/tcg/config"
	"github.com/gwos/tcg/log"
	"net/http"
	"net/url"
	"strconv"
	"strings"
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
		log.Warn("DSClient: Omit ValidateToken on demand config")
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

	statusCode, byteResponse, err := SendRequest(http.MethodPost, entrypoint.String(), headers, formValues, nil)

	logEntry := log.With(log.Fields{
		"error":      err,
		"statusCode": statusCode,
	}).WithDebug(log.Fields{
		"response":   string(byteResponse),
		"headers": headers,
		"reqURL":  entrypoint.String(),
	})
	logEntryLevel := log.InfoLevel
	defer func() {
		logEntry.Log(logEntryLevel, "DSClient: ValidateToken")
	}()

	if err == nil {
		if statusCode == 201 {
			b, _ := strconv.ParseBool(string(byteResponse))
			if b {
				return nil
			}
			return fmt.Errorf("invalid gwos-app-name or gwos-api-token")
		}
		return fmt.Errorf(string(byteResponse))
	}

	return err
}

// Reload implements DSOperations.Reload.
func (client *DSClient) Reload(agentID string) error {
	headers := map[string]string{
		"Accept":       "application/json",
		"Content-Type": "application/json",
	}
	reqURL := (&url.URL{
		Scheme: makeDalekServicesScheme(client.DSConnection.HostName),
		Host:   client.DSConnection.HostName,
		Path:   strings.ReplaceAll(DSEntrypointReload, ":agentID", agentID),
	}).String()

	statusCode, byteResponse, err := SendRequest(http.MethodPost, reqURL, headers, nil, nil)

	logEntry := log.With(log.Fields{
		"error":      err,
		"statusCode": statusCode,
	}).WithDebug(log.Fields{
		"response":   string(byteResponse),
		"headers": headers,
		"reqURL":  reqURL,
	})
	logEntryLevel := log.InfoLevel
	defer func() {
		logEntry.Log(logEntryLevel, "DSClient: Reload")
	}()

	if statusCode == 404 {
		logEntry.WithField("Hint", "Check AgentAI")
	}
	if err != nil {
		logEntryLevel = log.ErrorLevel
		return err
	}
	if statusCode != 201 {
		logEntryLevel = log.WarnLevel
		return fmt.Errorf(string(byteResponse))
	}
	return nil
}

// Create the scheme (http or https) based on hostName prefix
func makeDalekServicesScheme(hostName string) string {
	scheme := "https"
	if strings.HasPrefix(hostName, "dalekservices") {
		scheme =  "http"
	}
	return scheme
}
