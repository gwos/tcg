package clients

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"sync"

	tcgerr "github.com/gwos/tcg/sdk/errors"
	"github.com/gwos/tcg/sdk/logper"
)

// Define entrypoints
const (
	GWEntrypointAuthenticatePassword           = "/api/users/authenticatePassword"
	GWEntrypointConnect                        = "/api/auth/login"
	GWEntrypointDisconnect                     = "/api/auth/logout"
	GWEntrypointClearInDowntime                = "/api/biz/clearindowntime"
	GWEntrypointSetInDowntime                  = "/api/biz/setindowntime"
	GWEntrypointSendEvents                     = "/api/events"
	GWEntrypointSendEventsAck                  = "/api/events/ack"
	GWEntrypointSendEventsUnack                = "/api/events/unack"
	GWEntrypointSendResourceWithMetrics        = "/api/monitoring"
	GWEntrypointSendResourceWithMetricsDynamic = "/api/monitoring?dynamic=true"
	GWEntrypointSynchronizeInventory           = "/api/synchronizer"
	GWEntrypointServices                       = "/api/services"
	GWEntrypointHostgroups                     = "/api/hostgroups"
	GWEntrypointValidateToken                  = "/api/auth/validatetoken"
)

// GWConnection defines Groundwork Connection configuration
type GWConnection struct {
	ID int `yaml:"id"`
	// HostName accepts value for combined "host:port"
	// used as `url.URL{HostName}`
	HostName            string `yaml:"hostName"`
	UserName            string `yaml:"userName"`
	Password            string `yaml:"password"`
	Enabled             bool   `yaml:"enabled"`
	IsChild             bool   `yaml:"isChild"`
	DisplayName         string `yaml:"displayName"`
	MergeHosts          bool   `yaml:"mergeHosts"`
	LocalConnection     bool   `yaml:"localConnection"`
	DeferOwnership      string `yaml:"deferOwnership"`
	PrefixResourceNames bool   `yaml:"prefixResourceNames"`
	ResourceNamePrefix  string `yaml:"resourceNamePrefix"`
	SendAllInventory    bool   `yaml:"sendAllInventory"`
	IsDynamicInventory  bool   `yaml:"-"`
}

// GWHostGroups defines collection
type GWHostGroups struct {
	HostGroups []struct {
		Name  string `json:"name"`
		Hosts []struct {
			HostName string `json:"hostName"`
		} `json:"hosts"`
	} `json:"hostGroups"`
}

// GWServices defines collection
type GWServices struct {
	Services []struct {
		HostName string `json:"hostName"`
	} `json:"services"`
}

// GWClient implements GW API operations
type GWClient struct {
	AppName string
	AppType string
	*GWConnection

	mu   sync.Mutex
	once sync.Once

	token                             string
	uriAuthenticatePassword           string
	uriConnect                        string
	uriDisconnect                     string
	uriClearInDowntime                string
	uriSetInDowntime                  string
	uriSendEvents                     string
	uriSendEventsAck                  string
	uriSendEventsUnack                string
	uriSendResourceWithMetrics        string
	uriSendResourceWithMetricsDynamic string
	uriSynchronizeInventory           string
	uriServices                       string
	uriHostGroups                     string
	uriValidateToken                  string
}

// Connect calls API
func (client *GWClient) Connect() error {
	client.buildURIs()
	prevToken := client.token
	/* restrict by mutex for one-thread at one-time */
	client.mu.Lock()
	defer client.mu.Unlock()
	if prevToken != client.token {
		/* token already changed */
		return nil
	}
	if client.LocalConnection {
		token, err := client.connectLocal()
		if err == nil {
			client.token = token
		}
		return err
	}
	token, err := client.AuthenticatePassword(
		client.GWConnection.UserName,
		client.GWConnection.Password)
	if err == nil {
		client.token = token
	}
	return err
}

func (client *GWClient) connectLocal() (string, error) {
	formValues := map[string]string{
		"gwos-app-name": client.AppName,
		"user":          client.GWConnection.UserName,
		"password":      client.GWConnection.Password,
	}
	headers := map[string]string{
		"Accept":       "text/plain",
		"Content-Type": "application/x-www-form-urlencoded",
	}
	req, err := (&Req{
		URL:     client.uriConnect,
		Method:  http.MethodPost,
		Headers: headers,
		Form:    formValues,
	}).Send()

	switch {
	case err != nil:
		logper.Error(req, "could not connect local groundwork")
		if tcgerr.IsErrorConnection(err) {
			return "", fmt.Errorf("%w: %v", tcgerr.ErrTransient, err.Error())
		}
		return "", err

	case req.Status == 401 ||
		(req.Status == 404 && bytes.Contains(req.Response, []byte("password"))):
		eee := fmt.Errorf("%w: %v", tcgerr.ErrUnauthorized, string(req.Response))
		req.Err = eee
		logper.Warn(req, "could not connect local groundwork")
		return "", eee

	case req.Status == 502 || req.Status == 504:
		eee := fmt.Errorf("%w: %v", tcgerr.ErrGateway, string(req.Response))
		req.Err = eee
		logper.Warn(req, "could not connect local groundwork")
		return "", eee

	case req.Status == 503:
		eee := fmt.Errorf("%w: %v", tcgerr.ErrSynchronizer, string(req.Response))
		req.Err = eee
		logper.Warn(req, "could not connect local groundwork")
		return "", eee

	case req.Status != 200:
		eee := fmt.Errorf("%w: %v", tcgerr.ErrUndecided, string(req.Response))
		req.Err = eee
		logper.Warn(req.Details(), "could not connect local groundwork")
		return "", eee
	}
	logper.Debug(req, "connect local groundwork")
	return string(req.Response), nil
}

// AuthenticatePassword calls API and returns token
func (client *GWClient) AuthenticatePassword(username, password string) (string, error) {
	client.buildURIs()
	payload, _ := json.Marshal(map[string]string{
		"name":     username,
		"password": password,
	})
	headers := map[string]string{
		"Accept":        "application/json",
		"Content-Type":  "application/json",
		"GWOS-APP-NAME": client.AppName,
	}
	req, err := (&Req{
		URL:     client.uriAuthenticatePassword,
		Method:  http.MethodPut,
		Headers: headers,
		Payload: payload,
	}).Send()

	switch {
	case err != nil:
		logper.Error(req, "could not authenticate password")
		if tcgerr.IsErrorConnection(err) {
			return "", fmt.Errorf("%w: %v", tcgerr.ErrTransient, err.Error())
		}
		return "", err

	case req.Status == 401 ||
		(req.Status == 404 && bytes.Contains(req.Response, []byte("password"))):
		eee := fmt.Errorf("%w: %v", tcgerr.ErrUnauthorized, string(req.Response))
		req.Err = eee
		logper.Warn(req, "could not authenticate password")
		return "", eee

	case req.Status == 502 || req.Status == 504:
		eee := fmt.Errorf("%w: %v", tcgerr.ErrGateway, string(req.Response))
		req.Err = eee
		logper.Warn(req, "could not authenticate password")
		return "", eee

	case req.Status == 503:
		eee := fmt.Errorf("%w: %v", tcgerr.ErrSynchronizer, string(req.Response))
		req.Err = eee
		logper.Warn(req, "could not authenticate password")
		return "", eee

	case req.Status != 200:
		eee := fmt.Errorf("%w: %v", tcgerr.ErrUndecided, string(req.Response))
		req.Err = eee
		logper.Warn(req.Details(), "could not authenticate password")
		return "", eee
	}

	type UserResponse struct {
		Name        string `json:"name"`
		AccessToken string `json:"accessToken"`
	}
	user := UserResponse{AccessToken: ""}
	if err := json.Unmarshal(req.Response, &user); err != nil {
		logper.Warn(req.Details(), "could not authenticate password: parsingError: %s", err)
		return "", fmt.Errorf("%w: %v", tcgerr.ErrUndecided, err)
	}
	if logper.IsDebugEnabled() {
		logper.Debug(req, "authenticate password: userName: %s", user.Name)
	} else {
		logper.Debug(req, "authenticate password")
	}
	return user.AccessToken, nil
}

// Disconnect calls API
func (client *GWClient) Disconnect() error {
	client.buildURIs()
	formValues := map[string]string{
		"gwos-app-name":  client.AppName,
		"gwos-api-token": client.token,
	}
	headers := map[string]string{
		"Accept":       "text/plain",
		"Content-Type": "application/x-www-form-urlencoded",
	}
	req, err := (&Req{
		URL:     client.uriDisconnect,
		Method:  http.MethodPost,
		Headers: headers,
		Form:    formValues,
	}).Send()

	switch {
	case err != nil:
		logper.Error(req, "could not disconnect groundwork")
		if tcgerr.IsErrorConnection(err) {
			return fmt.Errorf("%w: %v", tcgerr.ErrTransient, err.Error())
		}
		return err

	case req.Status == 401:
		eee := fmt.Errorf("%w: %v", tcgerr.ErrUnauthorized, string(req.Response))
		req.Err = eee
		logper.Warn(req, "could not disconnect groundwork")
		return eee

	case req.Status == 502 || req.Status == 504:
		eee := fmt.Errorf("%w: %v", tcgerr.ErrGateway, string(req.Response))
		req.Err = eee
		logper.Warn(req, "could not disconnect groundwork")
		return eee

	case req.Status == 503:
		eee := fmt.Errorf("%w: %v", tcgerr.ErrSynchronizer, string(req.Response))
		req.Err = eee
		logper.Warn(req, "could not disconnect groundwork")
		return eee

	case req.Status != 200:
		eee := fmt.Errorf("%w: %v", tcgerr.ErrUndecided, string(req.Response))
		req.Err = eee
		logper.Warn(req.Details(), "could not disconnect groundwork")
		return eee
	}
	logper.Debug(req, "disconnect groundwork")
	return nil
}

// ValidateToken calls API
func (client *GWClient) ValidateToken(appName, apiToken string) error {
	client.buildURIs()
	headers := map[string]string{
		"Accept":       "text/plain",
		"Content-Type": "application/x-www-form-urlencoded",
	}
	formValues := map[string]string{
		"gwos-app-name":  appName,
		"gwos-api-token": apiToken,
	}
	req, err := (&Req{
		URL:     client.uriValidateToken,
		Method:  http.MethodPost,
		Headers: headers,
		Form:    formValues,
	}).Send()

	if err == nil {
		if req.Status == 200 {
			if b, e := strconv.ParseBool(string(req.Response)); e == nil && b {
				logper.Debug(req, "validate groundwork token")
				return nil
			}
			eee := fmt.Errorf("%w: %v", tcgerr.ErrUnauthorized, "invalid gwos-app-name or gwos-api-token")
			req.Err = eee
			logper.Warn(req, "could not validate groundwork token")
			return eee
		}
		eee := fmt.Errorf("%w: %v", tcgerr.ErrUndecided, string(req.Response))
		req.Err = eee
		logper.Warn(req.Details(), "could not validate groundwork token")
		return eee
	}

	logper.Error(req, "could not validate groundwork token")
	if tcgerr.IsErrorConnection(err) {
		return fmt.Errorf("%w: %v", tcgerr.ErrTransient, err.Error())
	}
	return err
}

// SynchronizeInventory calls API
func (client *GWClient) SynchronizeInventory(ctx context.Context, payload []byte) ([]byte, error) {
	client.buildURIs()
	mergeParam := make(map[string]string)
	mergeHosts := true
	if client.GWConnection != nil {
		mergeHosts = client.GWConnection.MergeHosts
	}
	mergeParam["merge"] = strconv.FormatBool(mergeHosts)
	reqURL := client.uriSynchronizeInventory + BuildQueryParams(mergeParam)
	if client.PrefixResourceNames && client.ResourceNamePrefix != "" {
		return client.sendData(ctx, reqURL, payload,
			header{
				"HostNamePrefix",
				client.ResourceNamePrefix,
			},
		)
	}
	response, err := client.sendData(ctx, reqURL, payload)
	return response, err
}

// SendResourcesWithMetrics calls API
func (client *GWClient) SendResourcesWithMetrics(ctx context.Context, payload []byte) ([]byte, error) {
	client.buildURIs()
	if client.PrefixResourceNames && client.ResourceNamePrefix != "" {
		return client.sendData(ctx, client.uriSendResourceWithMetrics, payload,
			header{
				"HostNamePrefix",
				client.ResourceNamePrefix,
			},
		)
	}

	if client.IsDynamicInventory {
		return client.sendData(ctx, client.uriSendResourceWithMetricsDynamic, payload)
	}

	return client.sendData(ctx, client.uriSendResourceWithMetrics, payload)
}

// ClearInDowntime calls API
func (client *GWClient) ClearInDowntime(ctx context.Context, payload []byte) ([]byte, error) {
	client.buildURIs()
	if client.PrefixResourceNames && client.ResourceNamePrefix != "" {
		return client.sendData(ctx, client.uriClearInDowntime, payload,
			header{
				"HostNamePrefix",
				client.ResourceNamePrefix,
			},
		)
	}
	return client.sendData(ctx, client.uriClearInDowntime, payload)
}

// SetInDowntime calls API
func (client *GWClient) SetInDowntime(ctx context.Context, payload []byte) ([]byte, error) {
	client.buildURIs()
	if client.PrefixResourceNames && client.ResourceNamePrefix != "" {
		return client.sendData(ctx, client.uriSetInDowntime, payload,
			header{
				"HostNamePrefix",
				client.ResourceNamePrefix,
			},
		)
	}
	return client.sendData(ctx, client.uriSetInDowntime, payload)
}

// SendEvents calls API
func (client *GWClient) SendEvents(ctx context.Context, payload []byte) ([]byte, error) {
	client.buildURIs()
	if client.PrefixResourceNames && client.ResourceNamePrefix != "" {
		return client.sendData(ctx, client.uriSendEvents, payload,
			header{
				"HostNamePrefix",
				client.ResourceNamePrefix,
			},
		)
	}
	return client.sendData(ctx, client.uriSendEvents, payload)
}

// SendEventsAck calls API
func (client *GWClient) SendEventsAck(ctx context.Context, payload []byte) ([]byte, error) {
	client.buildURIs()
	if client.PrefixResourceNames && client.ResourceNamePrefix != "" {
		return client.sendData(ctx, client.uriSendEventsAck, payload,
			header{
				"HostNamePrefix",
				client.ResourceNamePrefix,
			},
		)
	}
	return client.sendData(ctx, client.uriSendEventsAck, payload)
}

// SendEventsUnack calls API
func (client *GWClient) SendEventsUnack(ctx context.Context, payload []byte) ([]byte, error) {
	client.buildURIs()
	if client.PrefixResourceNames && client.ResourceNamePrefix != "" {
		return client.sendData(ctx, client.uriSendEventsUnack, payload,
			header{
				"HostNamePrefix",
				client.ResourceNamePrefix,
			},
		)
	}
	return client.sendData(ctx, client.uriSendEventsUnack, payload)
}

// GetServicesByAgent calls API
func (client *GWClient) GetServicesByAgent(agentID string) (*GWServices, error) {
	params := make(map[string]string)
	params["query"] = "agentid = '" + agentID + "'"
	params["depth"] = "Shallow"
	client.buildURIs()
	reqURL := client.uriServices + BuildQueryParams(params)
	response, err := client.sendRequest(context.Background(), http.MethodGet, reqURL, nil)
	if err != nil {
		logper.Error(obj{"error": err}, "could not get GW services")
		return nil, err
	}
	var gwServices GWServices
	err = json.Unmarshal(response, &gwServices)
	if err != nil {
		logper.Error(obj{"error": err}, "could not parse received GW services")
		return nil, err
	}
	return &gwServices, nil
}

// GetHostGroupsByHostNamesAndAppType calls API
func (client *GWClient) GetHostGroupsByHostNamesAndAppType(hostNames []string, appType string) (*GWHostGroups, error) {
	if len(hostNames) == 0 {
		return nil, errors.New("unable to get host groups of host: host names are not provided")
	}
	query := "( hosts.hostName in ("
	for i, hostName := range hostNames {
		query = query + "'" + hostName + "'"
		if i != len(hostNames)-1 {
			query = query + ","
		} else {
			query = query + ")"
		}
	}
	query = query + " ) and appType = '" + appType + "'"
	params := make(map[string]string)
	params["query"] = query
	params["depth"] = "Shallow"
	client.buildURIs()
	reqURL := client.uriHostGroups + BuildQueryParams(params)
	response, err := client.sendRequest(context.Background(), http.MethodGet, reqURL, nil)
	if err != nil {
		logper.Error(obj{"error": err}, "could not get GW host groups")
		return nil, err
	}
	var gwHostGroups GWHostGroups
	err = json.Unmarshal(response, &gwHostGroups)
	if err != nil {
		logper.Error(obj{"error": err}, "could not parse received GW host groups")
		return nil, err
	}
	return &gwHostGroups, nil
}

func (client *GWClient) sendData(ctx context.Context, reqURL string, payload []byte, additionalHeaders ...header) ([]byte, error) {
	return client.sendRequest(ctx, http.MethodPost, reqURL, payload, additionalHeaders...)
}

type header struct {
	key   string
	value string
}

func (client *GWClient) sendRequest(ctx context.Context, httpMethod string, reqURL string, payload []byte, additionalHeaders ...header) ([]byte, error) {
	headers := map[string]string{
		"Accept":         "application/json",
		"Content-Type":   "application/json",
		"GWOS-APP-NAME":  client.AppName,
		"GWOS-API-TOKEN": client.token,
	}
	for _, header := range additionalHeaders {
		headers[header.key] = header.value
	}
	req, err := (&Req{
		URL:     reqURL,
		Method:  httpMethod,
		Headers: headers,
		Payload: payload,
	}).SendWithContext(ctx)

	if err == nil && req.Status == 401 {
		logper.Debug(nil, "could not send request: reconnecting")
		if err := client.Connect(); err != nil {
			logper.Error(obj{"error": err}, "could not send request: could not reconnect")
			return nil, err
		}
		req.Headers["GWOS-API-TOKEN"] = client.token
		req, err = req.SendWithContext(ctx)
	}

	switch {
	case err != nil:
		logper.Error(req, "could not send request")
		if tcgerr.IsErrorConnection(err) {
			return nil, fmt.Errorf("%w: %v", tcgerr.ErrTransient, err.Error())
		}
		return nil, err

	case req.Status == 401:
		eee := fmt.Errorf("%w: %v", tcgerr.ErrUnauthorized, string(req.Response))
		req.Err = eee
		logper.Warn(req, "could not send request")
		return nil, eee

	case req.Status == 502 || req.Status == 504:
		eee := fmt.Errorf("%w: %v", tcgerr.ErrGateway, string(req.Response))
		req.Err = eee
		logper.Warn(req, "could not send request")
		return nil, eee

	case req.Status == 503:
		eee := fmt.Errorf("%w: %v", tcgerr.ErrSynchronizer, string(req.Response))
		req.Err = eee
		logper.Warn(req, "could not send request")
		return nil, eee

	case req.Status != 200:
		eee := fmt.Errorf("%w: %v", tcgerr.ErrUndecided, string(req.Response))
		req.Err = eee
		logper.Warn(req.Details(), "could not send request")
		return nil, eee
	}
	logper.Debug(req, "send request")
	return req.Response, nil
}

func (client *GWClient) buildURIs() {
	client.once.Do(func() {
		client.uriAuthenticatePassword = buildURI(client.GWConnection.HostName, GWEntrypointAuthenticatePassword)
		client.uriConnect = buildURI(client.GWConnection.HostName, GWEntrypointConnect)
		client.uriDisconnect = buildURI(client.GWConnection.HostName, GWEntrypointDisconnect)
		client.uriClearInDowntime = buildURI(client.GWConnection.HostName, GWEntrypointClearInDowntime)
		client.uriSetInDowntime = buildURI(client.GWConnection.HostName, GWEntrypointSetInDowntime)
		client.uriSendEvents = buildURI(client.GWConnection.HostName, GWEntrypointSendEvents)
		client.uriSendEventsAck = buildURI(client.GWConnection.HostName, GWEntrypointSendEventsAck)
		client.uriSendEventsUnack = buildURI(client.GWConnection.HostName, GWEntrypointSendEventsUnack)
		client.uriSendResourceWithMetrics = buildURI(client.GWConnection.HostName, GWEntrypointSendResourceWithMetrics)
		client.uriSendResourceWithMetricsDynamic = buildURI(client.GWConnection.HostName, GWEntrypointSendResourceWithMetricsDynamic)
		client.uriServices = buildURI(client.GWConnection.HostName, GWEntrypointServices)
		client.uriSynchronizeInventory = buildURI(client.GWConnection.HostName, GWEntrypointSynchronizeInventory)
		client.uriHostGroups = buildURI(client.GWConnection.HostName, GWEntrypointHostgroups)
		client.uriValidateToken = buildURI(client.GWConnection.HostName, GWEntrypointValidateToken)
	})
}

func buildURI(hostname, entrypoint string) string {
	s := strings.TrimSuffix(strings.TrimRight(hostname, "/"), "/api")
	if !strings.HasPrefix(s, "http") {
		s = "https://" + s
	}
	return fmt.Sprintf("%s/%s", s, strings.TrimLeft(entrypoint, "/"))
}

// obj defines a short alias
type obj map[string]interface{}
