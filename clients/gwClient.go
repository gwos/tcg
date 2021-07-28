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

	"github.com/gwos/tcg/config"
	tcgerr "github.com/gwos/tcg/errors"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

// GWOperations defines Groundwork operations interface
type GWOperations interface {
	Connect() error
	Disconnect() error
	GetServicesByAgent(agentID string) ([]byte, error)
	GetHostGroupsByHostNamesAndAppType(hostNames []string, appType string) ([]byte, error)
	ValidateToken(appName, apiToken string) error
	ClearInDowntime(ctx context.Context, payload []byte) ([]byte, error)
	SetInDowntime(ctx context.Context, payload []byte) ([]byte, error)
	SendEvents(ctx context.Context, payload []byte) ([]byte, error)
	SendEventsAck(ctx context.Context, payload []byte) ([]byte, error)
	SendEventsUnack(ctx context.Context, payload []byte) ([]byte, error)
	SendResourcesWithMetrics(ctx context.Context, payload []byte) ([]byte, error)
	SynchronizeInventory(ctx context.Context, payload []byte) ([]byte, error)
}

// Define entrypoints for GWOperations
const (
	GWEntrypointConnectRemote                  = "/api/users/authenticatePassword"
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

// GWClient implements GWOperations interface
type GWClient struct {
	AppName string
	AppType string
	*config.GWConnection

	mu   sync.Mutex
	once sync.Once

	token                             string
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

// AuthPayload defines Connect payload on nonLocalConnection
type AuthPayload struct {
	Name     string `json:"name"`
	Password string `json:"password"`
}

// UserResponse defines Connect response on nonLocalConnection
type UserResponse struct {
	Name        string `json:"name"`
	AccessToken string `json:"accessToken"`
}

type GwServices struct {
	Services []struct {
		HostName string `json:"hostName"`
	} `json:"services"`
}

type GwHostGroups struct {
	HostGroups []struct {
		Name  string `json:"name"`
		Hosts []struct {
			HostName string `json:"hostName"`
		} `json:"hosts"`
	} `json:"hostGroups"`
}

// Connect implements GWOperations.Connect.
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
		return client.connectLocal()
	}
	return client.connectRemote()
}

func (client *GWClient) connectLocal() error {
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
		req.LogWith(log.Error()).Msg("could not connect local groundwork")
		if tcgerr.IsErrorConnection(err) {
			return fmt.Errorf("%w: %v", tcgerr.ErrTransient, err.Error())
		}
		return err

	case req.Status == 401 ||
		(req.Status == 404 && bytes.Contains(req.Response, []byte("password"))):
		eee := fmt.Errorf("%w: %v", tcgerr.ErrUnauthorized, string(req.Response))
		req.LogWith(log.Warn()).Err(eee).Msg("could not connect local groundwork")
		return eee

	case req.Status == 502 || req.Status == 504:
		eee := fmt.Errorf("%w: %v", tcgerr.ErrGateway, string(req.Response))
		req.LogWith(log.Warn()).Err(eee).Msg("could not connect local groundwork")
		return eee

	case req.Status == 503:
		eee := fmt.Errorf("%w: %v", tcgerr.ErrSynchronizer, string(req.Response))
		req.LogWith(log.Warn()).Err(eee).Msg("could not connect local groundwork")
		return eee

	case req.Status != 200:
		eee := fmt.Errorf("%w: %v", tcgerr.ErrUndecided, string(req.Response))
		req.LogDetailsWith(log.Warn()).Err(eee).Msg("could not connect local groundwork")
		return eee
	}
	req.LogWith(log.Info()).Msg("connect local groundwork")
	client.token = string(req.Response)
	return nil
}

func (client *GWClient) connectRemote() error {
	authPayload := AuthPayload{
		Name:     client.GWConnection.UserName,
		Password: client.GWConnection.Password,
	}
	authBytes, _ := json.Marshal(authPayload)
	headers := map[string]string{
		"Accept":        "application/json",
		"Content-Type":  "application/json",
		"GWOS-APP-NAME": client.AppName,
	}
	req, err := (&Req{
		URL:     client.uriConnect,
		Method:  http.MethodPut,
		Headers: headers,
		Payload: authBytes,
	}).Send()

	switch {
	case err != nil:
		req.LogWith(log.Error()).Msg("could not connect remote groundwork")
		if tcgerr.IsErrorConnection(err) {
			return fmt.Errorf("%w: %v", tcgerr.ErrTransient, err.Error())
		}
		return err

	case req.Status == 401 ||
		(req.Status == 404 && bytes.Contains(req.Response, []byte("password"))):
		eee := fmt.Errorf("%w: %v", tcgerr.ErrUnauthorized, string(req.Response))
		req.LogWith(log.Warn()).Err(eee).Msg("could not connect remote groundwork")
		return eee

	case req.Status == 502 || req.Status == 504:
		eee := fmt.Errorf("%w: %v", tcgerr.ErrGateway, string(req.Response))
		req.LogWith(log.Warn()).Err(eee).Msg("could not connect remote groundwork")
		return eee

	case req.Status == 503:
		eee := fmt.Errorf("%w: %v", tcgerr.ErrSynchronizer, string(req.Response))
		req.LogWith(log.Warn()).Err(eee).Msg("could not connect remote groundwork")
		return eee

	case req.Status != 200:
		eee := fmt.Errorf("%w: %v", tcgerr.ErrUndecided, string(req.Response))
		req.LogDetailsWith(log.Warn()).Err(eee).Msg("could not connect remote groundwork")
		return eee
	}

	user := UserResponse{AccessToken: ""}
	if err := json.Unmarshal(req.Response, &user); err != nil {
		req.LogDetailsWith(log.Warn()).
			AnErr("parsingError", err).
			Msg("could not connect remote groundwork")
		return fmt.Errorf("%w: %v", tcgerr.ErrUndecided, err)
	}
	req.LogWith(log.Info()).
		Func(func(e *zerolog.Event) {
			if zerolog.GlobalLevel() <= zerolog.DebugLevel {
				e.Str("userName", user.Name).
					Str("userToken", user.AccessToken)
			}
		}).
		Msg("connect remote groundwork")
	client.token = user.AccessToken
	return nil
}

// Disconnect implements GWOperations.Disconnect.
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
		req.LogWith(log.Error()).Msg("could not disconnect groundwork")
		if tcgerr.IsErrorConnection(err) {
			return fmt.Errorf("%w: %v", tcgerr.ErrTransient, err.Error())
		}
		return err

	case req.Status == 401:
		eee := fmt.Errorf("%w: %v", tcgerr.ErrUnauthorized, string(req.Response))
		req.LogWith(log.Warn()).Err(eee).Msg("could not disconnect groundwork")
		return eee

	case req.Status == 502 || req.Status == 504:
		eee := fmt.Errorf("%w: %v", tcgerr.ErrGateway, string(req.Response))
		req.LogWith(log.Warn()).Err(eee).Msg("could not disconnect groundwork")
		return eee

	case req.Status == 503:
		eee := fmt.Errorf("%w: %v", tcgerr.ErrSynchronizer, string(req.Response))
		req.LogWith(log.Warn()).Err(eee).Msg("could not disconnect groundwork")
		return eee

	case req.Status != 200:
		eee := fmt.Errorf("%w: %v", tcgerr.ErrUndecided, string(req.Response))
		req.LogDetailsWith(log.Warn()).Err(eee).Msg("could not disconnect groundwork")
		return eee
	}
	req.LogWith(log.Info()).Msg("disconnect groundwork")
	return nil
}

// ValidateToken implements GWOperations.ValidateToken.
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
				req.LogWith(log.Debug()).Msg("validate groundwork token")
				return nil
			}
			eee := fmt.Errorf("%w: %v", tcgerr.ErrUnauthorized, "invalid gwos-app-name or gwos-api-token")
			req.LogWith(log.Warn()).Err(eee).Msg("could not validate groundwork token")
			return eee
		}
		eee := fmt.Errorf("%w: %v", tcgerr.ErrUndecided, string(req.Response))
		req.LogDetailsWith(log.Warn()).Err(eee).Msg("could not validate groundwork token")
		return eee
	}

	req.LogWith(log.Error()).Msg("could not validate groundwork token")
	if tcgerr.IsErrorConnection(err) {
		return fmt.Errorf("%w: %v", tcgerr.ErrTransient, err.Error())
	}
	return err
}

// SynchronizeInventory implements GWOperations.SynchronizeInventory.
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

// SendResourcesWithMetrics implements GWOperations.SendResourcesWithMetrics.
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

	if config.GetConfig().Connector.IsDynamicInventory {
		return client.sendData(ctx, client.uriSendResourceWithMetricsDynamic, payload)
	}

	return client.sendData(ctx, client.uriSendResourceWithMetrics, payload)
}

// ClearInDowntime implements GWOperations.ClearInDowntime.
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

// SetInDowntime implements GWOperations.SetInDowntime.
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

// SendEvents implements GWOperations.SendEvents.
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

// SendEventsAck implements GWOperations.SendEventsAck.
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

// SendEventsUnack implements GWOperations.SendEventsUnack.
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

// GetServicesByAgent implements GWOperations.GetServicesByAgent.
func (client *GWClient) GetServicesByAgent(agentID string) (*GwServices, error) {
	params := make(map[string]string)
	params["query"] = "agentid = '" + agentID + "'"
	params["depth"] = "Shallow"
	client.buildURIs()
	reqURL := client.uriServices + BuildQueryParams(params)
	response, err := client.sendRequest(context.Background(), http.MethodGet, reqURL, nil)
	if err != nil {
		log.Err(err).Msg("could not get GW services")
		return nil, err
	}
	var gwServices GwServices
	err = json.Unmarshal(response, &gwServices)
	if err != nil {
		log.Err(err).Msg("could not parse received GW services")
		return nil, err
	}
	return &gwServices, nil
}

func (client *GWClient) GetHostGroupsByHostNamesAndAppType(hostNames []string, appType string) (*GwHostGroups, error) {
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
		log.Err(err).Msg("could not get GW host groups")
		return nil, err
	}
	var gwHostGroups GwHostGroups
	err = json.Unmarshal(response, &gwHostGroups)
	if err != nil {
		log.Err(err).Msg("could not parse received GW host groups")
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
		log.Info().Msg("could not send request: reconnecting")
		if err := client.Connect(); err != nil {
			log.Err(err).Msg("could not send request: could not reconnect")
			return nil, err
		}
		req.Headers["GWOS-API-TOKEN"] = client.token
		req, err = req.SendWithContext(ctx)
	}

	switch {
	case err != nil:
		req.LogWith(log.Error()).Msg("could not send request")
		if tcgerr.IsErrorConnection(err) {
			return nil, fmt.Errorf("%w: %v", tcgerr.ErrTransient, err.Error())
		}
		return nil, err

	case req.Status == 401:
		eee := fmt.Errorf("%w: %v", tcgerr.ErrUnauthorized, string(req.Response))
		req.LogWith(log.Warn()).Err(eee).Msg("could not send request")
		return nil, eee

	case req.Status == 502 || req.Status == 504:
		eee := fmt.Errorf("%w: %v", tcgerr.ErrGateway, string(req.Response))
		req.LogWith(log.Warn()).Err(eee).Msg("could not send request")
		return nil, eee

	case req.Status == 503:
		eee := fmt.Errorf("%w: %v", tcgerr.ErrSynchronizer, string(req.Response))
		req.LogWith(log.Warn()).Err(eee).Msg("could not send request")
		return nil, eee

	case req.Status != 200:
		eee := fmt.Errorf("%w: %v", tcgerr.ErrUndecided, string(req.Response))
		req.LogDetailsWith(log.Warn()).Err(eee).Msg("could not send request")
		return nil, eee
	}
	req.LogWith(log.Info()).Msg("send request")
	return req.Response, nil
}

func (client *GWClient) buildURIs() {
	client.once.Do(func() {
		client.uriConnect = buildURI(client.GWConnection.HostName, GWEntrypointConnect)
		if !client.LocalConnection {
			client.uriConnect = buildURI(client.GWConnection.HostName, GWEntrypointConnectRemote)
		}
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
