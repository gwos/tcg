package clients

import (
	"fmt"
	"github.com/gwos/tng/config"
	"github.com/gwos/tng/log"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
)

// DSOperations defines DalekServices operations interface
type DSOperations interface {
	Connect() error
	FetchConnector(agentID string) ([]byte, error)
	FetchGWConnections(agentID string) ([]byte, error)
	ValidateToken(appName, apiToken string) error
}

// Define entrypoints for DSOperations
const (
	DSEntrypointConnect       = "/tng/login"
	DSEntrypointConnector     = "/tng/connector/name/:agentID?deep=true"
	DSEntrypointGWConnections = "/tng/connections/:agentID"
	DSEntrypointValidateToken = "/tng/validate-token"
)

// DSClient implements DSOperations interface
type DSClient struct {
	AppName string
	*config.DSConnection
	sync.Mutex
	token string
}

// Connect implements DSOperations.Connect.
func (client *DSClient) Connect() error {
	prevToken := client.token
	/* restrict by mutex for one-thread at one-time */
	client.Lock()
	defer client.Unlock()
	if prevToken != client.token {
		/* token already changed */
		return nil
	}

	formValues := map[string]string{
		"gwos-app-name": client.AppName,
		"user":          client.DSConnection.UserName,
		"password":      client.DSConnection.Password,
	}

	headers := map[string]string{
		"Accept":       "text/plain",
		"Content-Type": "application/x-www-form-urlencoded",
	}

	entrypoint := url.URL{
		Scheme: "http",
		Host:   client.DSConnection.HostName,
		Path:   DSEntrypointConnect,
	}
	statusCode, byteResponse, err := SendRequest(http.MethodPost, entrypoint.String(), headers, formValues, nil)
	if err != nil {
		return err
	}

	if statusCode == 200 {
		client.token = string(byteResponse)
		return nil
	}

	return fmt.Errorf(string(byteResponse))
}

// ValidateToken implements DSOperations.ValidateToken.
func (client *DSClient) ValidateToken(appName, apiToken string) error {
	headers := map[string]string{
		"Accept":       "text/plain",
		"Content-Type": "application/x-www-form-urlencoded",
	}

	formValues := map[string]string{
		"gwos-app-name":  appName,
		"gwos-api-token": apiToken,
	}

	entrypoint := url.URL{
		Scheme: "http",
		Host:   client.DSConnection.HostName,
		Path:   DSEntrypointValidateToken,
	}

	statusCode, byteResponse, err := SendRequest(http.MethodPost, entrypoint.String(), headers, formValues, nil)

	if err == nil {
		if statusCode == 200 {
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

// FetchGWConnections implements GWOperations.FetchGWConnections.
func (client *DSClient) FetchGWConnections(agentID string) ([]byte, error) {
	entrypoint := strings.ReplaceAll(DSEntrypointGWConnections, ":agentID", agentID)
	return client.fetchData(entrypoint)
}

// FetchConnector implements GWOperations.FetchConnector
func (client *DSClient) FetchConnector(agentID string) ([]byte, error) {
	entrypoint := strings.ReplaceAll(DSEntrypointConnector, ":agentID", agentID)
	return client.fetchData(entrypoint)
}

func (client *DSClient) fetchData(entrypoint string) ([]byte, error) {
	headers := map[string]string{
		"Accept":         "application/json",
		"Content-Type":   "application/json",
		"GWOS-APP-NAME":  client.AppName,
		"GWOS-API-TOKEN": client.token,
	}
	reqURL := (&url.URL{
		Scheme: "http",
		Host:   client.DSConnection.HostName,
		Path:   entrypoint,
	}).String()

	statusCode, byteResponse, err := SendRequest(http.MethodGet, reqURL, headers, nil, nil)
	if statusCode == 401 {
		err = client.Connect()
		if err != nil {
			return nil, err
		}
		headers["GWOS-API-TOKEN"] = client.token
		statusCode, byteResponse, err = SendRequest(http.MethodGet, reqURL, headers, nil, nil)
	}

	logEntry := log.With(log.Fields{
		"error":      err,
		"response":   string(byteResponse),
		"statusCode": statusCode,
	}).WithDebug(log.Fields{
		"headers": headers,
		"reqURL":  reqURL,
	})
	logEntryLevel := log.InfoLevel
	defer func() {
		logEntry.Log(logEntryLevel, "DSClient: fetchData")
	}()

	if err != nil {
		logEntryLevel = log.ErrorLevel
		return nil, err
	}
	if statusCode != 200 {
		logEntryLevel = log.WarnLevel
	}
	return byteResponse, nil
}
