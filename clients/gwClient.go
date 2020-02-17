package clients

import (
	"fmt"
	"github.com/gwos/tng/config"
	"github.com/gwos/tng/log"
	"net/http"
	"net/url"
	"strconv"
	"sync"
)

// GWOperations defines Groundwork operations interface
type GWOperations interface {
	Connect() error
	Disconnect() error
	ValidateToken(appName, apiToken string) error
	SendEvents(payload []byte) ([]byte, error)
	SendEventsAck(payload []byte) ([]byte, error)
	SendEventsUnack(payload []byte) ([]byte, error)
	SendResourcesWithMetrics(payload []byte) ([]byte, error)
	SynchronizeInventory(payload []byte) ([]byte, error)
}

// Define entrypoints for GWOperations
const (
	GWEntrypointConnect                 = "/api/auth/login"
	GWEntrypointDisconnect              = "/api/auth/logout"
	GWEntrypointSendEvents              = "/api/events"
	GWEntrypointSendEventsAck           = "/api/events/ack"
	GWEntrypointSendEventsUnack         = "/api/events/unack"
	GWEntrypointSendResourceWithMetrics = "/api/monitoring"
	GWEntrypointSynchronizeInventory    = "/api/synchronizer"
	GWEntrypointValidateToken           = "/api/auth/validatetoken"
)

// GWClient implements GWOperations interface
type GWClient struct {
	AppName string
	*config.GWConnection
	sync.Mutex
	token string
}

// Connect implements GWOperations.Connect.
func (client *GWClient) Connect() error {
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
		"user":          client.GWConnection.UserName,
		"password":      client.GWConnection.Password,
	}

	headers := map[string]string{
		"Accept":       "text/plain",
		"Content-Type": "application/x-www-form-urlencoded",
	}

	entrypoint := url.URL{
		Scheme: "http",
		Host:   client.GWConnection.HostName,
		Path:   GWEntrypointConnect,
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

// Disconnect implements GWOperations.Disconnect.
func (client *GWClient) Disconnect() error {
	formValues := map[string]string{
		"gwos-app-name":  client.AppName,
		"gwos-api-token": client.token,
	}

	headers := map[string]string{
		"Accept":       "text/plain",
		"Content-Type": "application/x-www-form-urlencoded",
	}

	entrypoint := url.URL{
		Scheme: "http",
		Host:   client.GWConnection.HostName,
		Path:   GWEntrypointDisconnect,
	}
	statusCode, byteResponse, err := SendRequest(http.MethodPost, entrypoint.String(), headers, formValues, nil)
	if err != nil {
		return err
	}

	if statusCode == 200 {
		return nil
	}
	return fmt.Errorf(string(byteResponse))
}

// ValidateToken implements GWOperations.ValidateToken.
func (client *GWClient) ValidateToken(appName, apiToken string) error {
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
		Host:   client.GWConnection.HostName,
		Path:   GWEntrypointValidateToken,
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

// SynchronizeInventory implements GWOperations.SynchronizeInventory.
func (client *GWClient) SynchronizeInventory(payload []byte) ([]byte, error) {
	return client.sendData(GWEntrypointSynchronizeInventory, payload)
}

// SendResourcesWithMetrics implements GWOperations.SendResourcesWithMetrics.
func (client *GWClient) SendResourcesWithMetrics(payload []byte) ([]byte, error) {
	return client.sendData(GWEntrypointSendResourceWithMetrics, payload)
}

// SendEvents implements GWOperations.SendEvents.
func (client *GWClient) SendEvents(payload []byte) ([]byte, error) {
	return client.sendData(GWEntrypointSendEvents, payload)
}

// SendEventsAck implements GWOperations.SendEventsAck.
func (client *GWClient) SendEventsAck(payload []byte) ([]byte, error) {
	return client.sendData(GWEntrypointSendEventsAck, payload)
}

// SendEventsUnack implements GWOperations.SendEventsUnack.
func (client *GWClient) SendEventsUnack(payload []byte) ([]byte, error) {
	return client.sendData(GWEntrypointSendEventsUnack, payload)
}

func (client *GWClient) sendData(entrypoint string, payload []byte) ([]byte, error) {
	headers := map[string]string{
		"Accept":         "application/json",
		"Content-Type":   "application/json",
		"GWOS-APP-NAME":  client.AppName,
		"GWOS-API-TOKEN": client.token,
	}
	reqURL := (&url.URL{
		Scheme: "http",
		Host:   client.GWConnection.HostName,
		Path:   entrypoint,
	}).String()

	statusCode, byteResponse, err := SendRequest(http.MethodPost, reqURL, headers, nil, payload)
	if statusCode == 401 {
		if err := client.Connect(); err != nil {
			return nil, err
		}
		headers["GWOS-API-TOKEN"] = client.token
		statusCode, byteResponse, err = SendRequest(http.MethodPost, reqURL, headers, nil, payload)
	}

	logEntry := log.With(log.Fields{
		"error":      err,
		"response":   string(byteResponse),
		"statusCode": statusCode,
	}).WithDebug(log.Fields{
		"headers": headers,
		"payload": string(payload),
		"reqURL":  reqURL,
	})
	logEntryLevel := log.InfoLevel
	defer func() {
		logEntry.Log(logEntryLevel, "GWClient: sendData")
	}()

	if err != nil {
		logEntryLevel = log.ErrorLevel
		return nil, err
	}
	if statusCode != 200 {
		logEntryLevel = log.WarnLevel
		return nil, fmt.Errorf(string(byteResponse))
	}
	return byteResponse, nil
}
