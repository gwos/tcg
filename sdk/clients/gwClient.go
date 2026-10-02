package clients

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
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
	c.Timeout = envDuration(time.Second*40, EnvHttpClientTimeoutGW, EnvHttpClientTimeout) // 40s by default
	return c
}()

// GWEntrypoint defines entrypoint
type GWEntrypoint string

// GWEntrypoint
const (
	GWEntrypointAuthenticate    GWEntrypoint = "/api/users/authenticatePassword"
	GWEntrypointConnect         GWEntrypoint = "/api/auth/login"
	GWEntrypointDisconnect      GWEntrypoint = "/api/auth/logout"
	GWEntrypointClearInDowntime GWEntrypoint = "/api/biz/clearindowntime"
	GWEntrypointSetInDowntime   GWEntrypoint = "/api/biz/setindowntime"
	GWEntrypointEvents          GWEntrypoint = "/api/events"
	GWEntrypointEventsAck       GWEntrypoint = "/api/events/ack"
	GWEntrypointEventsUnack     GWEntrypoint = "/api/events/unack"
	GWEntrypointMonitoring      GWEntrypoint = "/api/monitoring"
	GWEntrypointMonitoringDyn   GWEntrypoint = "/api/monitoring?dynamic=true"
	GWEntrypointSynchronizer    GWEntrypoint = "/api/synchronizer"
	GWEntrypointServices        GWEntrypoint = "/api/services"
	GWEntrypointHostgroups      GWEntrypoint = "/api/hostgroups"
	GWEntrypointValidateToken   GWEntrypoint = "/api/auth/validatetoken"
)

// gwEntrypoints lists the entrypoints whose URIs are built once per client
var gwEntrypoints = []GWEntrypoint{
	GWEntrypointAuthenticate,
	GWEntrypointConnect,
	GWEntrypointDisconnect,
	GWEntrypointClearInDowntime,
	GWEntrypointSetInDowntime,
	GWEntrypointEvents,
	GWEntrypointEventsAck,
	GWEntrypointEventsUnack,
	GWEntrypointMonitoring,
	GWEntrypointMonitoringDyn,
	GWEntrypointSynchronizer,
	GWEntrypointServices,
	GWEntrypointHostgroups,
	GWEntrypointValidateToken,
}

// GWConnection defines Groundwork Connection configuration
type GWConnection struct {
	ID int `env:"ID" yaml:"id"`
	// HostName accepts value for combined "host:port"
	// used as `url.URL{HostName}`
	HostName            string `env:"HOSTNAME" yaml:"hostName"`
	UserName            string `env:"USERNAME" yaml:"userName"`
	Password            string `env:"PASSWORD" yaml:"password"`
	Enabled             bool   `env:"ENABLED" yaml:"enabled"`
	IsChild             bool   `env:"ISCHILD" yaml:"isChild"`
	DisplayName         string `env:"DISPLAYNAME" yaml:"displayName"`
	MergeHosts          bool   `env:"MERGEHOSTS" yaml:"mergeHosts"`
	LocalConnection     bool   `env:"LOCALCONNECTION" yaml:"localConnection"`
	DeferOwnership      string `env:"DEFEROWNERSHIP" yaml:"deferOwnership"`
	PrefixResourceNames bool   `env:"PREFIXRESOURCENAMES" yaml:"prefixResourceNames"`
	ResourceNamePrefix  string `env:"RESOURCENAMEPREFIX" yaml:"resourceNamePrefix"`
	SendAllInventory    bool   `env:"SENDALLINVENTORY" yaml:"sendAllInventory"`
	IsDynamicInventory  bool   `env:"ISDYNAMICINVENTORY" yaml:"-"`
	HTTPEncode          bool   `env:"HTTPENCODE" yaml:"-"`
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

	once sync.Once
	uris map[GWEntrypoint]string
}

// tokens shares tokens between client instances
var tokens = new(sync.Map)

// connectMu serializes Connect across GWClient instances that share a token key
var connectMu = new(sync.Map) // tokenKey -> *sync.Mutex

// tokenKey identifies a shared token in the tokens map
func (client *GWClient) tokenKey() string {
	return client.AppName + ";" + client.UserName + ";" + client.HostName
}

func (client *GWClient) token() string {
	if v, ok := tokens.Load(client.tokenKey()); ok {
		return v.(string)
	}
	return ""
}

// formHeaders returns new headers for form requests, as callers may extend them
func formHeaders() map[string]string {
	return map[string]string{
		"Accept":       "text/plain",
		"Content-Type": "application/x-www-form-urlencoded",
	}
}

// jsonHeaders returns new headers for JSON requests, as callers may extend them
func jsonHeaders() map[string]string {
	return map[string]string{
		"Accept":       "application/json",
		"Content-Type": "application/json",
	}
}

// isTransientNet reports network errors that are worth retrying
func isTransientNet(err error) bool {
	return tcgerr.IsErrorDNS(err) || tcgerr.IsErrorConnection(err) || tcgerr.IsErrorTimedOut(err)
}

// transportErr logs a failed request and wraps network errors worth retrying as transient
func transportErr(ctx context.Context, req *Req, msg string, err error) error {
	sdklog.Logger.LogAttrs(ctx, slog.LevelError, msg, req.LogAttrs()...)
	if isTransientNet(err) {
		return fmt.Errorf("%w: %v", tcgerr.ErrTransient, err.Error())
	}
	return err
}

// isAuthNotFound matches the 404 that Foundation returns for a wrong password on login
func isAuthNotFound(req *Req) bool {
	return req.Status == http.StatusNotFound && bytes.Contains(req.Response, []byte("password"))
}

// statusErr maps a response status other than 200 OK to an error, records it in req and logs msg.
// With authNotFound, a 404 about the password is reported as unauthorized.
func statusErr(ctx context.Context, req *Req, msg string, authNotFound bool) error {
	var base error
	switch {
	case req.Status == http.StatusOK:
		return nil
	case req.Status == http.StatusUnauthorized, authNotFound && isAuthNotFound(req):
		base = tcgerr.ErrUnauthorized
	case req.Status == http.StatusNotFound:
		base = tcgerr.ErrNotFound
	case req.Status == http.StatusBadGateway || req.Status == http.StatusGatewayTimeout:
		base = tcgerr.ErrGateway
	case req.Status == http.StatusServiceUnavailable:
		base = tcgerr.ErrSynchronizer
	default:
		base = tcgerr.ErrUndecided
	}
	req.Err = fmt.Errorf("%w: %v", base, string(req.Response))
	attrs := req.LogAttrs()
	if base == tcgerr.ErrUndecided {
		attrs = req.Details()
	}
	sdklog.Logger.LogAttrs(ctx, slog.LevelWarn, msg, attrs...)
	return req.Err
}

// checkTokenResponse checks the response of a validate-token request that succeeds with okStatus
func checkTokenResponse(ctx context.Context, req *Req, okStatus int, msg string) error {
	if req.Status != okStatus {
		req.Err = fmt.Errorf("%w: %v", tcgerr.ErrUndecided, string(req.Response))
		sdklog.Logger.LogAttrs(ctx, slog.LevelWarn, "could not "+msg, req.Details()...)
		return req.Err
	}
	if b, err := strconv.ParseBool(string(req.Response)); err != nil || !b {
		req.Err = fmt.Errorf("%w: %v", tcgerr.ErrUnauthorized, "invalid gwos-app-name or gwos-api-token")
		sdklog.Logger.LogAttrs(ctx, slog.LevelWarn, "could not "+msg, req.LogAttrs()...)
		return req.Err
	}
	sdklog.Logger.LogAttrs(ctx, slog.LevelDebug, msg, req.LogAttrs()...)
	return nil
}

// Connect calls API
func (client *GWClient) Connect() error {
	return client.reconnect(client.token())
}

// reconnect logs in unless the shared token no longer equals stale,
// which means another request has already replaced it.
func (client *GWClient) reconnect(stale string) error {
	k := client.tokenKey()
	muAny, _ := connectMu.LoadOrStore(k, new(sync.Mutex))
	mu := muAny.(*sync.Mutex)

	/* restrict by mutex for one-thread at one-time, shared across instances */
	mu.Lock()
	defer mu.Unlock()
	if stale != client.token() {
		/* token already changed */
		return nil
	}
	var (
		token string
		err   error
	)
	if client.LocalConnection {
		token, err = client.connectLocal()
	} else {
		token, err = client.AuthenticatePassword(client.GWConnection.UserName, client.GWConnection.Password)
	}
	if err == nil {
		tokens.Store(k, token)
	}
	return err
}

func (client *GWClient) connectLocal() (string, error) {
	formValues := map[string]string{
		"gwos-app-name": client.AppName,
		"user":          client.GWConnection.UserName,
		"password":      client.GWConnection.Password,
	}

	ctx, req := context.TODO(), Req{}
	const msg = "could not connect local groundwork"
	if err := client.doReq(ctx, &req, http.MethodPost, GWEntrypointConnect, "",
		formHeaders(), formValues, nil); err != nil {
		return "", transportErr(ctx, &req, msg, err)
	}
	if err := statusErr(ctx, &req, msg, true); err != nil {
		return "", err
	}
	sdklog.Logger.LogAttrs(ctx, slog.LevelDebug, "connect local groundwork", req.LogAttrs()...)
	return string(req.Response), nil
}

// AuthenticatePassword calls API and returns token
func (client *GWClient) AuthenticatePassword(username, password string) (string, error) {
	payload, _ := json.Marshal(map[string]string{
		"name":     username,
		"password": password,
	})
	headers := jsonHeaders()
	headers["GWOS-APP-NAME"] = client.AppName

	ctx, req := context.TODO(), Req{}
	const msg = "could not authenticate password"
	if err := client.doReq(ctx, &req, http.MethodPut, GWEntrypointAuthenticate, "",
		headers, nil, payload); err != nil {
		return "", transportErr(ctx, &req, msg, err)
	}
	if err := statusErr(ctx, &req, msg, true); err != nil {
		return "", err
	}

	var user struct {
		Name        string `json:"name"`
		AccessToken string `json:"accessToken"`
	}
	if err := json.Unmarshal(req.Response, &user); err != nil {
		sdklog.Logger.LogAttrs(ctx, slog.LevelWarn, fmt.Sprintf("could not authenticate password: parsingError: %s", err), req.Details()...)
		return "", fmt.Errorf("%w: %v", tcgerr.ErrUndecided, err)
	}
	sdklog.Logger.LogAttrs(ctx, slog.LevelDebug, "authenticate password", req.LogAttrs()...)
	return user.AccessToken, nil
}

// Disconnect calls API
func (client *GWClient) Disconnect() error {
	formValues := map[string]string{
		"gwos-app-name":  client.AppName,
		"gwos-api-token": client.token(),
	}

	ctx, req := context.TODO(), Req{}
	const msg = "could not disconnect groundwork"
	if err := client.doReq(ctx, &req, http.MethodPost, GWEntrypointDisconnect, "",
		formHeaders(), formValues, nil); err != nil {
		return transportErr(ctx, &req, msg, err)
	}
	if err := statusErr(ctx, &req, msg, false); err != nil {
		return err
	}
	sdklog.Logger.LogAttrs(ctx, slog.LevelDebug, "disconnect groundwork", req.LogAttrs()...)
	return nil
}

// ValidateToken calls API
func (client *GWClient) ValidateToken(appName, apiToken string) error {
	formValues := map[string]string{
		"gwos-app-name":  appName,
		"gwos-api-token": apiToken,
	}

	ctx, req := context.TODO(), Req{}
	if err := client.doReq(ctx, &req, http.MethodPost, GWEntrypointValidateToken, "",
		formHeaders(), formValues, nil); err != nil {
		return transportErr(ctx, &req, "could not validate groundwork token", err)
	}
	return checkTokenResponse(ctx, &req, http.StatusOK, "validate groundwork token")
}

// httpEncodeMinSize is the payload size above which the request body is gzip-compressed
// even when the connection isn't configured for it (HTTPEncode=false).
// This protects direct SDK callers (e.g. the Telegraf output plugin)
// that never go through Put2Nats's own size-triggered compression,
// and generally reduces the odds of exceeding Foundation's entity-size limit.
// Matches Undertow's default MAX_ENTITY_SIZE on the Foundation side.
const httpEncodeMinSize = 2 * 1024 * 1024

// resolveEncoding gzip-compresses payload when required -
// either because HTTPEncode is enabled for the connection,
// the payload is already gzipped, or payload exceeds httpEncodeMinSize -
// and returns the (possibly compressed) payload along with the headers needed to declare it.
func (client *GWClient) resolveEncoding(ctx context.Context, payload []byte) (context.Context, []byte, []string, error) {
	switch {
	case IsGZipped(payload):
	case client.GWConnection.HTTPEncode || len(payload) > httpEncodeMinSize:
		var (
			buf bytes.Buffer
			err error
		)
		buf.Grow(len(payload) / 4)
		if ctx, err = GZip(ctx, &buf, payload); err != nil {
			return ctx, nil, nil, err
		}
		payload = buf.Bytes()
	default:
		return ctx, payload, nil, nil
	}
	return ctx, payload, []string{"Content-Encoding", "gzip"}, nil
}

// postPayload sends payload to entrypoint, declaring or applying gzip encoding as resolveEncoding decides,
// so a payload that arrives already gzipped (e.g. from Put2Nats) is never sent undeclared
func (client *GWClient) postPayload(ctx context.Context, entrypoint GWEntrypoint, queryStr string, payload []byte) ([]byte, error) {
	ctx, payload, headers, err := client.resolveEncoding(ctx, payload)
	if err != nil {
		return nil, err
	}
	if client.PrefixResourceNames && client.ResourceNamePrefix != "" {
		headers = append(headers, "HostNamePrefix", client.ResourceNamePrefix)
	}
	return client.SendRequest(ctx, http.MethodPost, entrypoint, queryStr, payload, headers...)
}

// SynchronizeInventory calls API
func (client *GWClient) SynchronizeInventory(ctx context.Context, payload []byte) ([]byte, error) {
	return client.postPayload(ctx, GWEntrypointSynchronizer,
		"?merge="+strconv.FormatBool(client.GWConnection.MergeHosts), payload)
}

// SendResourcesWithMetrics calls API
func (client *GWClient) SendResourcesWithMetrics(ctx context.Context, payload []byte) ([]byte, error) {
	entrypoint := GWEntrypointMonitoring
	if client.IsDynamicInventory {
		entrypoint = GWEntrypointMonitoringDyn
	}
	return client.postPayload(ctx, entrypoint, "", payload)
}

// ClearInDowntime calls API
func (client *GWClient) ClearInDowntime(ctx context.Context, payload []byte) ([]byte, error) {
	return client.postPayload(ctx, GWEntrypointClearInDowntime, "", payload)
}

// SetInDowntime calls API
func (client *GWClient) SetInDowntime(ctx context.Context, payload []byte) ([]byte, error) {
	return client.postPayload(ctx, GWEntrypointSetInDowntime, "", payload)
}

// SendEvents calls API
func (client *GWClient) SendEvents(ctx context.Context, payload []byte) ([]byte, error) {
	return client.postPayload(ctx, GWEntrypointEvents, "", payload)
}

// SendEventsAck calls API
func (client *GWClient) SendEventsAck(ctx context.Context, payload []byte) ([]byte, error) {
	return client.postPayload(ctx, GWEntrypointEventsAck, "", payload)
}

// SendEventsUnack calls API
func (client *GWClient) SendEventsUnack(ctx context.Context, payload []byte) ([]byte, error) {
	return client.postPayload(ctx, GWEntrypointEventsUnack, "", payload)
}

// getJSON queries the entrypoint and decodes the JSON response into out, what names it in logs
func (client *GWClient) getJSON(entrypoint GWEntrypoint, query string, what string, out any) error {
	ctx := context.TODO()
	params := map[string]string{
		"query": query,
		"depth": "Shallow",
	}
	response, err := client.SendRequest(ctx, http.MethodGet, entrypoint, BuildQueryParams(params), nil)
	if err != nil {
		sdklog.Logger.LogAttrs(ctx, slog.LevelError, "could not get GW "+what, slog.String("error", err.Error()))
		return err
	}
	if err := json.Unmarshal(response, out); err != nil {
		sdklog.Logger.LogAttrs(ctx, slog.LevelError, "could not parse received GW "+what, slog.String("error", err.Error()))
		return err
	}
	return nil
}

// GetServicesByAgent calls API
func (client *GWClient) GetServicesByAgent(agentID string, gwServices *GWServices) error {
	return client.getJSON(GWEntrypointServices, "agentid='"+agentID+"'", "services", gwServices)
}

// GetHostGroupsByAppTypeAndHostNames calls API
func (client *GWClient) GetHostGroupsByAppTypeAndHostNames(appType string, hostNames []string, gwHostGroups *GWHostGroups) error {
	if len(hostNames) == 0 {
		return errors.New("unable to get host groups of host: host names are not provided")
	}
	query := "( hosts.hostName in ('" + strings.Join(hostNames, "','") + "') ) and appType = '" + appType + "'"
	return client.getJSON(GWEntrypointHostgroups, query, "host groups", gwHostGroups)
}

func (client *GWClient) SendRequest(ctx context.Context, httpMethod string, entrypoint GWEntrypoint, queryStr string,
	payload []byte, additionalHeaders ...string) ([]byte, error) {

	headers := jsonHeaders()
	headers["GWOS-APP-NAME"] = client.AppName
	headers["GWOS-API-TOKEN"] = client.token()
	if len(additionalHeaders)%2 != 0 {
		sdklog.Logger.LogAttrs(ctx, slog.LevelWarn,
			"SendRequest: odd number of additionalHeaders; last value ignored",
			slog.Int("count", len(additionalHeaders)))
	}
	for i := 0; i < len(additionalHeaders)-1; i += 2 {
		k, v := additionalHeaders[i], additionalHeaders[i+1]
		headers[k] = v
	}

	req := Req{}
	err := client.doReq(ctx, &req, httpMethod, entrypoint, queryStr, headers, nil, payload)
	if err == nil && req.Status == http.StatusUnauthorized {
		sdklog.Logger.LogAttrs(ctx, slog.LevelDebug, "could not send request: reconnecting")
		if err := client.reconnect(headers["GWOS-API-TOKEN"]); err != nil {
			sdklog.Logger.LogAttrs(ctx, slog.LevelError, "could not send request: could not reconnect", slog.String("error", err.Error()))
			return nil, err
		}
		req.Headers["GWOS-API-TOKEN"] = client.token()
		err = req.SendWithContext(ctx)
	}

	const msg = "could not send request"
	if err != nil {
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) || isTransientNet(err) {
			sdklog.Logger.LogAttrs(ctx, slog.LevelWarn, msg, req.LogAttrs()...)
			return nil, fmt.Errorf("%w: %v", tcgerr.ErrTransient, err.Error())
		}
		sdklog.Logger.LogAttrs(ctx, slog.LevelError, msg, req.LogAttrs()...)
		return nil, err
	}
	if err := statusErr(ctx, &req, msg, false); err != nil {
		return nil, err
	}

	if sdklog.Logger.Enabled(ctx, slog.LevelDebug) {
		sdklog.Logger.LogAttrs(ctx, slog.LevelDebug, "send request", req.LogAttrs()...)
	}
	return req.Response, nil
}

func (client *GWClient) doReq(ctx context.Context, req *Req, httpMethod string, entrypoint GWEntrypoint, queryStr string,
	headers map[string]string, form map[string]string, payload []byte) error {

	client.once.Do(func() {
		var b strings.Builder
		client.uris = make(map[GWEntrypoint]string, len(gwEntrypoints))
		for _, ep := range gwEntrypoints {
			client.uris[ep] = buildURI(&b, client.GWConnection.HostName, ep)
		}
	})
	uri, ok := client.uris[entrypoint]
	if !ok {
		// support any API entrypoint for tests
		uri = buildURI(&strings.Builder{}, client.GWConnection.HostName, entrypoint)
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
