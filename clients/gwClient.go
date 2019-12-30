package clients

import (
	"encoding/json"
	"fmt"
	"github.com/gwos/tng/setup"
	"github.com/gwos/tng/transit"
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
	SendResourcesWithMetrics(request []byte) (*transit.OperationResults, error)
	SynchronizeInventory(request []byte) (*transit.OperationResults, error)
}

// Define entrypoints for GWOperations
const (
	GWEntrypointConnect                 = "/api/auth/login"
	GWEntrypointDisconnect              = "/api/auth/logout"
	GWEntrypointSynchronizeInventory    = "/api/synchronizer"
	GWEntrypointSendResourceWithMetrics = "/api/monitoring"
	GWEntrypointValidateToken           = "/api/auth/validatetoken"
)

// GWClient implements GWOperations interface
type GWClient struct {
	AppName string
	*setup.GWConnection
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
func (client *GWClient) SynchronizeInventory(inventory []byte) (*transit.OperationResults, error) {
	headers := map[string]string{
		"Accept":         "application/json",
		"Content-Type":   "application/json",
		"GWOS-API-TOKEN": client.token,
		"GWOS-APP-NAME":  client.AppName,
	}

	entrypoint := url.URL{
		Scheme: "http",
		Host:   client.GWConnection.HostName,
		Path:   GWEntrypointSynchronizeInventory,
	}
	statusCode, byteResponse, err := SendRequest(http.MethodPost, entrypoint.String(), headers, nil, inventory)
	if statusCode == 401 {
		err = client.Connect()
		if err != nil {
			return nil, err
		}
		headers["GWOS-API-TOKEN"] = client.token
		statusCode, byteResponse, err = SendRequest(http.MethodPost, entrypoint.String(), headers, nil, inventory)
	}
	if err != nil {
		return nil, err
	}
	if statusCode != 200 {
		return nil, fmt.Errorf(string(byteResponse))
	}

	var operationResults transit.OperationResults

	err = json.Unmarshal(byteResponse, &operationResults)
	if err != nil {
		return nil, err
	}

	return &operationResults, nil
}

// SendResourcesWithMetrics implements GWOperations.SendResourcesWithMetrics.
func (client *GWClient) SendResourcesWithMetrics(resources []byte) (*transit.OperationResults, error) {
	headers := map[string]string{
		"Accept":         "application/json",
		"Content-Type":   "application/json",
		"GWOS-API-TOKEN": client.token,
		"GWOS-APP-NAME":  client.AppName,
	}

	entrypoint := url.URL{
		Scheme: "http",
		Host:   client.GWConnection.HostName,
		Path:   GWEntrypointSendResourceWithMetrics,
	}
	statusCode, byteResponse, err := SendRequest(http.MethodPost, entrypoint.String(), headers, nil, resources)
	if statusCode == 401 {
		err = client.Connect()
		if err != nil {
			return nil, err
		}
		headers["GWOS-API-TOKEN"] = client.token
		statusCode, byteResponse, err = SendRequest(http.MethodPost, entrypoint.String(), headers, nil, resources)
	}
	if err != nil {
		return nil, err
	}
	if statusCode != 200 {
		return nil, fmt.Errorf(string(byteResponse))
	}

	var operationResults transit.OperationResults

	err = json.Unmarshal(byteResponse, &operationResults)
	if err != nil {
		return nil, err
	}

	return &operationResults, nil
}
