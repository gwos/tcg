package clients

import (
	"fmt"
	"github.com/gwos/tng/config"
	"gopkg.in/yaml.v3"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
)

// DSOperations defines DalekServices operations interface
type DSOperations interface {
	Connect() error
	GetGWConnections(agentID string) (config.GWConnections, error)
	ValidateToken(appName, apiToken string) error
}

// Define entrypoints for DSOperations
const (
	DSEntrypointConnect       = "/tng/login"
	DSEntrypointGWConnections = "/tng/gw-connections/:agentID"
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

// GetGWConnections implements GWOperations.GetGWConnections.
func (client *DSClient) GetGWConnections(agentID string) (config.GWConnections, error) {
	headers := map[string]string{
		"Accept":         "application/json",
		"Content-Type":   "application/json",
		"GWOS-API-TOKEN": client.DSConnection.Password,
		"GWOS-APP-NAME":  client.AppName,
	}

	entrypoint := url.URL{
		Scheme: "http",
		Host:   client.DSConnection.HostName,
		Path:   strings.ReplaceAll(DSEntrypointGWConnections, ":agentID", agentID),
	}
	statusCode, byteResponse, err := SendRequest(http.MethodGet, entrypoint.String(), headers, nil, nil)
	if statusCode == 401 {
		err = client.Connect()
		if err != nil {
			return nil, err
		}
		headers["GWOS-API-TOKEN"] = client.token
		statusCode, byteResponse, err = SendRequest(http.MethodGet, entrypoint.String(), headers, nil, nil)
	}
	if err != nil {
		return nil, err
	}
	if statusCode != 200 {
		return nil, fmt.Errorf(string(byteResponse))
	}

	var res config.GWConnections
	if err := yaml.Unmarshal(byteResponse, &res); err != nil {
		return nil, err
	}
	return res, nil
}
