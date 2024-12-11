package clients

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	tcgerr "github.com/gwos/tcg/sdk/errors"
	sdklog "github.com/gwos/tcg/sdk/log"
)

const EnvHttpClientTimeoutGW = "TCG_HTTP_CLIENT_TIMEOUT_GW"

var HttpClientGW = func() *http.Client {
	c := new(http.Client)
	*c = *HttpClient
	c.Timeout = func() time.Duration {
		if s, ok := os.LookupEnv(EnvHttpClientTimeoutGW); ok {
			if v, err := time.ParseDuration(s); err == nil {
				return v
			}
		}
		if s, ok := os.LookupEnv(EnvHttpClientTimeout); ok {
			if v, err := time.ParseDuration(s); err == nil {
				return v
			}
		}
		return time.Second * 40 // 40s by default
	}()
	return c
}()

// GWEntrypoint defines entrypoint
type GWEntrypoint string

// GWEntrypoint
const (
	GWEntrypointAuthenticatePassword       GWEntrypoint = "/api/users/authenticatePassword"
	GWEntrypointConnect                    GWEntrypoint = "/api/auth/login"
	GWEntrypointDisconnect                 GWEntrypoint = "/api/auth/logout"
	GWEntrypointClearInDowntime            GWEntrypoint = "/api/biz/clearindowntime"
	GWEntrypointSetInDowntime              GWEntrypoint = "/api/biz/setindowntime"
	GWEntrypointSendEvents                 GWEntrypoint = "/api/events"
	GWEntrypointSendEventsAck              GWEntrypoint = "/api/events/ack"
	GWEntrypointSendEventsUnack            GWEntrypoint = "/api/events/unack"
	GWEntrypointSendResourceWithMetrics    GWEntrypoint = "/api/monitoring"
	GWEntrypointSendResourceWithMetricsDyn GWEntrypoint = "/api/monitoring?dynamic=true"
	GWEntrypointSynchronizeInventory       GWEntrypoint = "/api/synchronizer"
	GWEntrypointServices                   GWEntrypoint = "/api/services"
	GWEntrypointHostgroups                 GWEntrypoint = "/api/hostgroups"
	GWEntrypointValidateToken              GWEntrypoint = "/api/auth/validatetoken"
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
	HTTPEncode          bool   `yaml:"-"`
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
		Description string `json:"description"`
		HostName    string `json:"hostName"`
		// skip other fields as not used for today
	} `json:"services"`
}

// GWClient implements GW API operations
type GWClient struct {
	AppName string
	AppType string
	GWConnection

	mu   sync.Mutex
	once sync.Once

	token string

	uriAuthenticatePassword       string
	uriConnect                    string
	uriDisconnect                 string
	uriClearInDowntime            string
	uriSetInDowntime              string
	uriSendEvents                 string
	uriSendEventsAck              string
	uriSendEventsUnack            string
	uriSendResourceWithMetrics    string
	uriSendResourceWithMetricsDyn string
	uriSynchronizeInventory       string
	uriServices                   string
	uriHostgroups                 string
	uriValidateToken              string
}

// Connect calls API
func (client *GWClient) Connect() error {
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

	req := new(Req)
	err := client.doReq(context.Background(), req, http.MethodPost, GWEntrypointConnect, "",
		headers, formValues, nil)

	switch {
	case err != nil:
		sdklog.Logger.LogAttrs(context.Background(), slog.LevelError, "could not connect local groundwork", req.LogAttrs()...)
		if tcgerr.IsErrorConnection(err) || tcgerr.IsErrorTimedOut(err) {
			return "", fmt.Errorf("%w: %v", tcgerr.ErrTransient, err.Error())
		}
		return "", err

	case req.Status == 401 ||
		(req.Status == 404 && bytes.Contains(req.Response, []byte("password"))):
		eee := fmt.Errorf("%w: %v", tcgerr.ErrUnauthorized, string(req.Response))
		req.Err = eee
		sdklog.Logger.LogAttrs(context.Background(), slog.LevelWarn, "could not connect local groundwork", req.LogAttrs()...)
		return "", eee

	case req.Status == 502 || req.Status == 504:
		eee := fmt.Errorf("%w: %v", tcgerr.ErrGateway, string(req.Response))
		req.Err = eee
		sdklog.Logger.LogAttrs(context.Background(), slog.LevelWarn, "could not connect local groundwork", req.LogAttrs()...)
		return "", eee

	case req.Status == 503:
		eee := fmt.Errorf("%w: %v", tcgerr.ErrSynchronizer, string(req.Response))
		req.Err = eee
		sdklog.Logger.LogAttrs(context.Background(), slog.LevelWarn, "could not connect local groundwork", req.LogAttrs()...)
		return "", eee

	case req.Status != 200:
		eee := fmt.Errorf("%w: %v", tcgerr.ErrUndecided, string(req.Response))
		req.Err = eee
		sdklog.Logger.LogAttrs(context.Background(), slog.LevelWarn, "could not connect local groundwork", req.Details()...)
		return "", eee
	}
	sdklog.Logger.LogAttrs(context.Background(), slog.LevelDebug, "connect local groundwork", req.LogAttrs()...)
	return string(req.Response), nil
}

// AuthenticatePassword calls API and returns token
func (client *GWClient) AuthenticatePassword(username, password string) (string, error) {
	payload, _ := json.Marshal(map[string]string{
		"name":     username,
		"password": password,
	})
	headers := map[string]string{
		"Accept":        "application/json",
		"Content-Type":  "application/json",
		"GWOS-APP-NAME": client.AppName,
	}

	req := new(Req)
	err := client.doReq(context.Background(), req, http.MethodPut, GWEntrypointAuthenticatePassword, "",
		headers, nil, payload)

	switch {
	case err != nil:
		sdklog.Logger.LogAttrs(context.Background(), slog.LevelError, "could not authenticate password", req.LogAttrs()...)
		if tcgerr.IsErrorConnection(err) || tcgerr.IsErrorTimedOut(err) {
			return "", fmt.Errorf("%w: %v", tcgerr.ErrTransient, err.Error())
		}
		return "", err

	case req.Status == 401 ||
		(req.Status == 404 && bytes.Contains(req.Response, []byte("password"))):
		eee := fmt.Errorf("%w: %v", tcgerr.ErrUnauthorized, string(req.Response))
		req.Err = eee
		sdklog.Logger.LogAttrs(context.Background(), slog.LevelWarn, "could not authenticate password", req.LogAttrs()...)
		return "", eee

	case req.Status == 502 || req.Status == 504:
		eee := fmt.Errorf("%w: %v", tcgerr.ErrGateway, string(req.Response))
		req.Err = eee
		sdklog.Logger.LogAttrs(context.Background(), slog.LevelWarn, "could not authenticate password", req.LogAttrs()...)
		return "", eee

	case req.Status == 503:
		eee := fmt.Errorf("%w: %v", tcgerr.ErrSynchronizer, string(req.Response))
		req.Err = eee
		sdklog.Logger.LogAttrs(context.Background(), slog.LevelWarn, "could not authenticate password", req.LogAttrs()...)
		return "", eee

	case req.Status != 200:
		eee := fmt.Errorf("%w: %v", tcgerr.ErrUndecided, string(req.Response))
		req.Err = eee
		sdklog.Logger.LogAttrs(context.Background(), slog.LevelWarn, "could not authenticate password", req.Details()...)
		return "", eee
	}

	type UserResponse struct {
		Name        string `json:"name"`
		AccessToken string `json:"accessToken"`
	}
	user := UserResponse{AccessToken: ""}
	if err := json.Unmarshal(req.Response, &user); err != nil {
		sdklog.Logger.LogAttrs(context.Background(), slog.LevelWarn, fmt.Sprintf("could not authenticate password: parsingError: %s", err), req.Details()...)
		return "", fmt.Errorf("%w: %v", tcgerr.ErrUndecided, err)
	}
	sdklog.Logger.LogAttrs(context.Background(), slog.LevelDebug, "authenticate password", req.LogAttrs()...)
	return user.AccessToken, nil
}

// Disconnect calls API
func (client *GWClient) Disconnect() error {
	formValues := map[string]string{
		"gwos-app-name":  client.AppName,
		"gwos-api-token": client.token,
	}
	headers := map[string]string{
		"Accept":       "text/plain",
		"Content-Type": "application/x-www-form-urlencoded",
	}

	req := new(Req)
	err := client.doReq(context.Background(), req, http.MethodPost, GWEntrypointDisconnect, "",
		headers, formValues, nil)

	switch {
	case err != nil:
		sdklog.Logger.LogAttrs(context.Background(), slog.LevelError, "could not disconnect groundwork", req.LogAttrs()...)
		if tcgerr.IsErrorConnection(err) || tcgerr.IsErrorTimedOut(err) {
			return fmt.Errorf("%w: %v", tcgerr.ErrTransient, err.Error())
		}
		return err

	case req.Status == 401:
		eee := fmt.Errorf("%w: %v", tcgerr.ErrUnauthorized, string(req.Response))
		req.Err = eee
		sdklog.Logger.LogAttrs(context.Background(), slog.LevelWarn, "could not disconnect groundwork", req.LogAttrs()...)
		return eee

	case req.Status == 502 || req.Status == 504:
		eee := fmt.Errorf("%w: %v", tcgerr.ErrGateway, string(req.Response))
		req.Err = eee
		sdklog.Logger.LogAttrs(context.Background(), slog.LevelWarn, "could not disconnect groundwork", req.LogAttrs()...)
		return eee

	case req.Status == 503:
		eee := fmt.Errorf("%w: %v", tcgerr.ErrSynchronizer, string(req.Response))
		req.Err = eee
		sdklog.Logger.LogAttrs(context.Background(), slog.LevelWarn, "could not disconnect groundwork", req.LogAttrs()...)
		return eee

	case req.Status != 200:
		eee := fmt.Errorf("%w: %v", tcgerr.ErrUndecided, string(req.Response))
		req.Err = eee
		sdklog.Logger.LogAttrs(context.Background(), slog.LevelWarn, "could not disconnect groundwork", req.Details()...)
		return eee
	}
	sdklog.Logger.LogAttrs(context.Background(), slog.LevelDebug, "disconnect groundwork", req.LogAttrs()...)
	return nil
}

// ValidateToken calls API
func (client *GWClient) ValidateToken(appName, apiToken string) error {
	headers := map[string]string{
		"Accept":       "text/plain",
		"Content-Type": "application/x-www-form-urlencoded",
	}
	formValues := map[string]string{
		"gwos-app-name":  appName,
		"gwos-api-token": apiToken,
	}

	req := new(Req)
	err := client.doReq(context.Background(), req, http.MethodPost, GWEntrypointValidateToken, "",
		headers, formValues, nil)

	if err == nil {
		if req.Status == 200 {
			if b, e := strconv.ParseBool(string(req.Response)); e == nil && b {
				sdklog.Logger.LogAttrs(context.Background(), slog.LevelDebug, "validate groundwork token", req.LogAttrs()...)
				return nil
			}
			eee := fmt.Errorf("%w: %v", tcgerr.ErrUnauthorized, "invalid gwos-app-name or gwos-api-token")
			req.Err = eee
			sdklog.Logger.LogAttrs(context.Background(), slog.LevelWarn, "could not validate groundwork token", req.LogAttrs()...)
			return eee
		}
		eee := fmt.Errorf("%w: %v", tcgerr.ErrUndecided, string(req.Response))
		req.Err = eee
		sdklog.Logger.LogAttrs(context.Background(), slog.LevelWarn, "could not validate groundwork token", req.Details()...)
		return eee
	}

	sdklog.Logger.LogAttrs(context.Background(), slog.LevelError, "could not validate groundwork token", req.LogAttrs()...)
	if tcgerr.IsErrorConnection(err) || tcgerr.IsErrorTimedOut(err) {
		return fmt.Errorf("%w: %v", tcgerr.ErrTransient, err.Error())
	}
	return err
}

// SynchronizeInventory calls API
func (client *GWClient) SynchronizeInventory(ctx context.Context, payload []byte) ([]byte, error) {
	headers := []string{}
	hdrCompressed := ""
	if h := ctx.Value(CtxHeaders); h != nil {
		if h, ok := h.(interface{ Get(string) string }); ok {
			if hdrCompressed = h.Get(HdrCompressed); hdrCompressed != "" {
				headers = append(headers, "Content-Encoding", hdrCompressed)
			}
		}
	}
	mergeParam := make(map[string]string)
	mergeHosts := client.GWConnection.MergeHosts
	if client.GWConnection.HTTPEncode && hdrCompressed == "" {
		var buf bytes.Buffer
		var err error
		ctx, err = GZIP(ctx, &buf, payload)
		if err != nil {
			return nil, err
		}
		payload = buf.Bytes()
		headers = append(headers, "Content-Encoding", "gzip")
	}
	mergeParam["merge"] = strconv.FormatBool(mergeHosts)
	if client.PrefixResourceNames && client.ResourceNamePrefix != "" {
		headers = append(headers, "HostNamePrefix", client.ResourceNamePrefix)
	}
	return client.sendRequest(ctx, http.MethodPost, GWEntrypointSynchronizeInventory, BuildQueryParams(mergeParam),
		payload, headers...)
}

// SendResourcesWithMetrics calls API
func (client *GWClient) SendResourcesWithMetrics(ctx context.Context, payload []byte) ([]byte, error) {
	headers := []string{}
	hdrCompressed := ""
	if h := ctx.Value(CtxHeaders); h != nil {
		if h, ok := h.(interface{ Get(string) string }); ok {
			if hdrCompressed = h.Get(HdrCompressed); hdrCompressed != "" {
				headers = append(headers, "Content-Encoding", hdrCompressed)
			}
		}
	}
	if client.GWConnection.HTTPEncode && hdrCompressed == "" {
		var buf bytes.Buffer
		var err error
		if ctx, err = GZIP(ctx, &buf, payload); err != nil {
			return nil, err
		}
		payload = buf.Bytes()
		headers = append(headers, "Content-Encoding", "gzip")
	}
	entrypoint := GWEntrypointSendResourceWithMetrics
	if client.IsDynamicInventory {
		entrypoint = GWEntrypointSendResourceWithMetricsDyn
	}
	if client.PrefixResourceNames && client.ResourceNamePrefix != "" {
		headers = append(headers, "HostNamePrefix", client.ResourceNamePrefix)
	}
	return client.sendRequest(ctx, http.MethodPost, entrypoint, "", payload, headers...)
}

// ClearInDowntime calls API
func (client *GWClient) ClearInDowntime(ctx context.Context, payload []byte) ([]byte, error) {
	if client.PrefixResourceNames && client.ResourceNamePrefix != "" {
		return client.sendRequest(ctx, http.MethodPost, GWEntrypointClearInDowntime, "", payload,
			"HostNamePrefix", client.ResourceNamePrefix,
		)
	}
	return client.sendRequest(ctx, http.MethodPost, GWEntrypointClearInDowntime, "", payload)
}

// SetInDowntime calls API
func (client *GWClient) SetInDowntime(ctx context.Context, payload []byte) ([]byte, error) {
	if client.PrefixResourceNames && client.ResourceNamePrefix != "" {
		return client.sendRequest(ctx, http.MethodPost, GWEntrypointSetInDowntime, "", payload,
			"HostNamePrefix", client.ResourceNamePrefix,
		)
	}
	return client.sendRequest(ctx, http.MethodPost, GWEntrypointSetInDowntime, "", payload)
}

// SendEvents calls API
func (client *GWClient) SendEvents(ctx context.Context, payload []byte) ([]byte, error) {
	if client.PrefixResourceNames && client.ResourceNamePrefix != "" {
		return client.sendRequest(ctx, http.MethodPost, GWEntrypointSendEvents, "", payload,
			"HostNamePrefix", client.ResourceNamePrefix,
		)
	}
	return client.sendRequest(ctx, http.MethodPost, GWEntrypointSendEvents, "", payload)
}

// SendEventsAck calls API
func (client *GWClient) SendEventsAck(ctx context.Context, payload []byte) ([]byte, error) {
	if client.PrefixResourceNames && client.ResourceNamePrefix != "" {
		return client.sendRequest(ctx, http.MethodPost, GWEntrypointSendEventsAck, "", payload,
			"HostNamePrefix", client.ResourceNamePrefix,
		)
	}
	return client.sendRequest(ctx, http.MethodPost, GWEntrypointSendEventsAck, "", payload)
}

// SendEventsUnack calls API
func (client *GWClient) SendEventsUnack(ctx context.Context, payload []byte) ([]byte, error) {
	if client.PrefixResourceNames && client.ResourceNamePrefix != "" {
		return client.sendRequest(ctx, http.MethodPost, GWEntrypointSendEventsUnack, "", payload,
			"HostNamePrefix", client.ResourceNamePrefix,
		)
	}
	return client.sendRequest(ctx, http.MethodPost, GWEntrypointSendEventsUnack, "", payload)
}

// GetServicesByAgent calls API
func (client *GWClient) GetServicesByAgent(agentID string, gwServices *GWServices) error {
	params := make(map[string]string)
	params["query"] = "agentid = '" + agentID + "'"
	params["depth"] = "Shallow"

	response, err := client.sendRequest(context.Background(), http.MethodGet, GWEntrypointServices, BuildQueryParams(params), nil)
	if err != nil {
		sdklog.Logger.LogAttrs(context.Background(), slog.LevelError, "could not get GW services", slog.String("error", err.Error()))
		return err
	}
	err = json.Unmarshal(response, gwServices)
	if err != nil {
		sdklog.Logger.LogAttrs(context.Background(), slog.LevelError, "could not parse received GW services", slog.String("error", err.Error()))
		return err
	}
	return nil
}

// GetHostGroupsByAppTypeAndHostNames calls API
func (client *GWClient) GetHostGroupsByAppTypeAndHostNames(appType string, hostNames []string, gwHostGroups *GWHostGroups) error {
	if len(hostNames) == 0 {
		return errors.New("unable to get host groups of host: host names are not provided")
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

	response, err := client.sendRequest(context.Background(), http.MethodGet, GWEntrypointHostgroups, BuildQueryParams(params), nil)
	if err != nil {
		sdklog.Logger.LogAttrs(context.Background(), slog.LevelError, "could not get GW host groups", slog.String("error", err.Error()))
		return err
	}
	err = json.Unmarshal(response, gwHostGroups)
	if err != nil {
		sdklog.Logger.LogAttrs(context.Background(), slog.LevelError, "could not parse received GW host groups", slog.String("error", err.Error()))
		return err
	}
	return nil
}

func (client *GWClient) sendRequest(ctx context.Context, httpMethod string, entrypoint GWEntrypoint, queryStr string,
	payload []byte, additionalHeaders ...string) ([]byte, error) {

	headers := map[string]string{
		"Accept":         "application/json",
		"Content-Type":   "application/json",
		"GWOS-APP-NAME":  client.AppName,
		"GWOS-API-TOKEN": client.token,
	}
	for i := 0; i < len(additionalHeaders)-1; i += 2 {
		k, v := additionalHeaders[i], additionalHeaders[i+1]
		headers[k] = v
	}

	req := new(Req)
	err := client.doReq(ctx, req, httpMethod, entrypoint, queryStr, headers, nil, payload)
	if err == nil && req.Status == 401 {
		sdklog.Logger.LogAttrs(ctx, slog.LevelDebug, "could not send request: reconnecting")
		if err := client.Connect(); err != nil {
			sdklog.Logger.LogAttrs(ctx, slog.LevelError, "could not send request: could not reconnect", slog.String("error", err.Error()))
			return nil, err
		}
		req.Headers["GWOS-API-TOKEN"] = client.token
		err = req.SendWithContext(ctx)
	}

	switch {
	case err != nil:
		sdklog.Logger.LogAttrs(ctx, slog.LevelError, "could not send request", req.LogAttrs()...)
		if tcgerr.IsErrorConnection(err) || tcgerr.IsErrorTimedOut(err) {
			return nil, fmt.Errorf("%w: %v", tcgerr.ErrTransient, err.Error())
		}
		return nil, err

	case req.Status == 401:
		eee := fmt.Errorf("%w: %v", tcgerr.ErrUnauthorized, string(req.Response))
		req.Err = eee
		sdklog.Logger.LogAttrs(ctx, slog.LevelWarn, "could not send request", req.LogAttrs()...)
		return nil, eee

	case req.Status == 502 || req.Status == 504:
		eee := fmt.Errorf("%w: %v", tcgerr.ErrGateway, string(req.Response))
		req.Err = eee
		sdklog.Logger.LogAttrs(ctx, slog.LevelWarn, "could not send request", req.LogAttrs()...)
		return nil, eee

	case req.Status == 503:
		eee := fmt.Errorf("%w: %v", tcgerr.ErrSynchronizer, string(req.Response))
		req.Err = eee
		sdklog.Logger.LogAttrs(ctx, slog.LevelWarn, "could not send request", req.LogAttrs()...)
		return nil, eee

	case req.Status != 200:
		eee := fmt.Errorf("%w: %v", tcgerr.ErrUndecided, string(req.Response))
		req.Err = eee
		sdklog.Logger.LogAttrs(ctx, slog.LevelWarn, "could not send request", req.Details()...)
		return nil, eee
	}

	sdklog.Logger.LogAttrs(ctx, slog.LevelDebug, "send request", req.LogAttrs()...)
	return req.Response, nil
}

func (client *GWClient) doReq(ctx context.Context, req *Req, httpMethod string, entrypoint GWEntrypoint, queryStr string,
	headers map[string]string, form map[string]string, payload []byte) error {

	client.once.Do(func() { client.buildURIs() })

	var uri string
	switch entrypoint {
	case GWEntrypointAuthenticatePassword:
		uri = client.uriAuthenticatePassword
	case GWEntrypointConnect:
		uri = client.uriConnect
	case GWEntrypointDisconnect:
		uri = client.uriDisconnect
	case GWEntrypointClearInDowntime:
		uri = client.uriClearInDowntime
	case GWEntrypointSetInDowntime:
		uri = client.uriSetInDowntime
	case GWEntrypointSendEvents:
		uri = client.uriSendEvents
	case GWEntrypointSendEventsAck:
		uri = client.uriSendEventsAck
	case GWEntrypointSendEventsUnack:
		uri = client.uriSendEventsUnack
	case GWEntrypointSendResourceWithMetrics:
		uri = client.uriSendResourceWithMetrics
	case GWEntrypointSendResourceWithMetricsDyn:
		uri = client.uriSendResourceWithMetricsDyn
	case GWEntrypointServices:
		uri = client.uriServices
	case GWEntrypointSynchronizeInventory:
		uri = client.uriSynchronizeInventory
	case GWEntrypointHostgroups:
		uri = client.uriHostgroups
	case GWEntrypointValidateToken:
		uri = client.uriValidateToken
	}

	*req = Req{
		URL:     uri + queryStr,
		Method:  httpMethod,
		Headers: headers,
		Form:    form,
		Payload: payload,
	}
	return req.SetClient(HttpClientGW).SendWithContext(ctx)
}

func (client *GWClient) buildURIs() {
	var b strings.Builder
	client.uriAuthenticatePassword = buildURI(&b, client.GWConnection.HostName, GWEntrypointAuthenticatePassword)
	client.uriConnect = buildURI(&b, client.GWConnection.HostName, GWEntrypointConnect)
	client.uriDisconnect = buildURI(&b, client.GWConnection.HostName, GWEntrypointDisconnect)
	client.uriClearInDowntime = buildURI(&b, client.GWConnection.HostName, GWEntrypointClearInDowntime)
	client.uriSetInDowntime = buildURI(&b, client.GWConnection.HostName, GWEntrypointSetInDowntime)
	client.uriSendEvents = buildURI(&b, client.GWConnection.HostName, GWEntrypointSendEvents)
	client.uriSendEventsAck = buildURI(&b, client.GWConnection.HostName, GWEntrypointSendEventsAck)
	client.uriSendEventsUnack = buildURI(&b, client.GWConnection.HostName, GWEntrypointSendEventsUnack)
	client.uriSendResourceWithMetrics = buildURI(&b, client.GWConnection.HostName, GWEntrypointSendResourceWithMetrics)
	client.uriSendResourceWithMetricsDyn = buildURI(&b, client.GWConnection.HostName, GWEntrypointSendResourceWithMetricsDyn)
	client.uriSynchronizeInventory = buildURI(&b, client.GWConnection.HostName, GWEntrypointSynchronizeInventory)
	client.uriServices = buildURI(&b, client.GWConnection.HostName, GWEntrypointServices)
	client.uriHostgroups = buildURI(&b, client.GWConnection.HostName, GWEntrypointHostgroups)
	client.uriValidateToken = buildURI(&b, client.GWConnection.HostName, GWEntrypointValidateToken)
}

func buildURI(b *strings.Builder, hostname string, entrypoint GWEntrypoint) string {
	b.Reset()
	if !strings.HasPrefix(hostname, "http") {
		_, _ = b.WriteString("https://")
	}
	fmt.Fprintf(b, "%s/%s",
		strings.TrimSuffix(strings.TrimRight(hostname, "/"), "/api"),
		strings.TrimLeft(string(entrypoint), "/"))
	return b.String()
}
