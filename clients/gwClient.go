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

	"github.com/gwos/tcg/cache"
	"github.com/gwos/tcg/config"
	"github.com/gwos/tcg/log"
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
	GWEntrypointLicenseCheck                   = "/api/license/check"
	GWEntrypointValidateToken                  = "/api/auth/validatetoken"
)

// GWClient implements GWOperations interface
type GWClient struct {
	AppName string
	AppType string
	*config.GWConnection
	sync.Mutex
	token string
	sync.Once
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
	uriLicenseCheck                   string
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

type LicenseCheck struct {
	Success          bool   `json:"success"`
	Message          string `json:"message"`
	DevicesRequested int    `json:"devicesRequested"`
	LimitDevices     int    `json:"hardLimitDevices"`
	Devices          int    `json:"devices"`
}

// Returns whether it is OK to send inventory according to the latest license check result
// (false only if the latest license check was failure with "Device allocation over hard limit" message
// i.e. if license check is failure but because of license expired or something else it is still OK to send inventory)
func (license LicenseCheck) isOkToSendInventory() bool {
	if license.Success {
		return true
	} else {
		// if license is for example simply expired or anything - that shouldn't prevent to send inventory
		// we care only about device limit
		if license.Message == "Device allocation over hard limit" {
			return false
		}
		return true
	}
}

// Connect implements GWOperations.Connect.
func (client *GWClient) Connect() error {
	client.buildURIs()
	prevToken := client.token
	/* restrict by mutex for one-thread at one-time */
	client.Lock()
	defer client.Unlock()
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
	reqURL := client.uriConnect
	statusCode, byteResponse, err := SendRequest(http.MethodPost, reqURL, headers, formValues, nil)
	logEntry := log.With(log.Fields{
		"error":      err,
		"statusCode": statusCode,
	}).WithDebug(log.Fields{
		"response": string(byteResponse),
		"headers":  headers,
		"reqURL":   reqURL,
	})
	logEntryLevel := log.InfoLevel

	defer logEntry.Log(logEntryLevel, "GWClient: connectLocal")

	if err != nil {
		logEntryLevel = log.ErrorLevel
		return err
	}
	if statusCode == 200 {
		client.token = string(byteResponse)
		logEntry.WithDebug(log.Fields{
			"token": client.token,
		})
		return nil
	}
	if statusCode == 502 || statusCode == 504 {
		return fmt.Errorf("%w: %v", ErrGateway, string(byteResponse))
	}
	if statusCode == 401 || (statusCode == 404 && bytes.Contains(byteResponse, []byte("password"))) {
		return fmt.Errorf("%w: %v", ErrUnauthorized, string(byteResponse))
	}
	return fmt.Errorf("%w: %v", ErrUndecided, string(byteResponse))
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
	reqURL := client.uriConnect
	statusCode, byteResponse, err := SendRequest(http.MethodPut, reqURL, headers, nil, authBytes)
	logEntry := log.With(log.Fields{
		"error":      err,
		"statusCode": statusCode,
	}).WithDebug(log.Fields{
		"response": string(byteResponse),
		"headers":  headers,
		"reqURL":   reqURL,
	})
	logEntryLevel := log.InfoLevel

	defer logEntry.Log(logEntryLevel, "GWClient: connectRemote")

	if err != nil {
		logEntryLevel = log.ErrorLevel
		return err
	}
	user := UserResponse{AccessToken: ""}
	error2 := "unknown error"
	if statusCode == 200 {
		// client.token = string(byteResponse)
		error2 := json.Unmarshal(byteResponse, &user)
		if error2 == nil {
			client.token = user.AccessToken
			logEntry.WithDebug(log.Fields{
				"user": user,
			})
			return nil
		}
	}
	logEntry.WithInfo(log.Fields{
		"errorCode": error2,
	})
	if statusCode == 502 || statusCode == 504 {
		return fmt.Errorf("%w: %v", ErrGateway, error2)
	}
	if statusCode == 401 || (statusCode == 404 && bytes.Contains(byteResponse, []byte("password"))) {
		return fmt.Errorf("%w: %v", ErrUnauthorized, string(byteResponse))
	}
	return fmt.Errorf("%w: %v", ErrUndecided, error2)
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
	reqURL := client.uriDisconnect
	statusCode, byteResponse, err := SendRequest(http.MethodPost, reqURL, headers, formValues, nil)

	logEntry := log.With(log.Fields{
		"error":      err,
		"statusCode": statusCode,
	}).WithDebug(log.Fields{
		"response": string(byteResponse),
		"headers":  headers,
		"reqURL":   reqURL,
	})
	logEntryLevel := log.InfoLevel

	defer logEntry.Log(logEntryLevel, "GWClient: disconnect")

	if err != nil {
		return err
	}

	if statusCode == 200 {
		return nil
	}
	if statusCode == 502 || statusCode == 504 {
		return fmt.Errorf("%w: %v", ErrGateway, string(byteResponse))
	}
	if statusCode == 401 {
		return fmt.Errorf("%w: %v", ErrUnauthorized, string(byteResponse))
	}
	return fmt.Errorf("%w: %v", ErrUndecided, string(byteResponse))
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
	reqURL := client.uriValidateToken
	statusCode, byteResponse, err := SendRequest(http.MethodPost, reqURL, headers, formValues, nil)

	logEntry := log.With(log.Fields{
		"error":      err,
		"statusCode": statusCode,
	}).WithDebug(log.Fields{
		"response": string(byteResponse),
		"headers":  headers,
		"reqURL":   reqURL,
	})
	logEntryLevel := log.InfoLevel

	defer logEntry.Log(logEntryLevel, "GWClient: validate token")

	if err == nil {
		if statusCode == 200 {
			b, _ := strconv.ParseBool(string(byteResponse))
			if b {
				return nil
			}
			return fmt.Errorf("%w: %v", ErrUnauthorized, "invalid gwos-app-name or gwos-api-token")
		}
		return fmt.Errorf("%w: %v", ErrUndecided, string(byteResponse))
	}

	return err
}

// SynchronizeInventory implements GWOperations.SynchronizeInventory.
func (client *GWClient) SynchronizeInventory(ctx context.Context, payload []byte) ([]byte, error) {
	hostName := "unknown"
	if client.GWConnection != nil {
		hostName = client.GWConnection.HostName
	}

	// retrieve count of resources (hosts) to be sent and agentID
	var payloadResources struct {
		Context struct {
			AgentID string `json:"agentId"`
		} `json:"context"`
		Resources []interface{} `json:"resources"`
	}
	err := json.Unmarshal(payload, &payloadResources)
	if err != nil {
		log.Error("|gwClient.go| : [SynchronizeInventory] : Unable to parse SynchronizeInventory payload: ", err)
	}
	currentHostsCount := len(payloadResources.Resources)

	// get count of hosts sent last time from cache
	// (if doesn't exist in cache count hosts owned curr agent in GWOS
	// for a case if connector was stopped and then restarted)
	lastSentHostsCount := 0
	if lastSentHostsCountCache, exists := cache.LastSentHostsCountCache.Get(hostName); exists {
		lastSentHostsCount = lastSentHostsCountCache.(int)
	} else {
		lastSentHostsCount = client.getLastSentHostsCount(payloadResources.Context.AgentID)
		cache.LastSentHostsCountCache.SetDefault(hostName, lastSentHostsCount)
	}

	// count diff between curr hosts to send count and last i.e. how many hosts gonna be added
	hostsToAllocate := currentHostsCount - lastSentHostsCount
	// if gonna ad hosts
	if hostsToAllocate > 0 {
		// check whether license allows to allocate count of hosts
		licenseCheckPassed, err := client.checkLicenseForHostLimit(hostsToAllocate)
		if !licenseCheckPassed {
			return nil, err
		}
	}
	client.buildURIs()
	mergeParam := make(map[string]string)
	mergeHosts := true
	if client.GWConnection != nil {
		mergeHosts = client.GWConnection.MergeHosts
	}
	mergeParam["merge"] = strconv.FormatBool(mergeHosts)
	syncUri := client.uriSynchronizeInventory + BuildQueryParams(mergeParam)
	if client.PrefixResourceNames && client.ResourceNamePrefix != "" {
		return client.sendData(ctx, syncUri, payload,
			header{
				"HostNamePrefix",
				client.ResourceNamePrefix,
			},
		)
	}
	response, err := client.sendData(ctx, syncUri, payload)
	if err == nil {
		cache.LastSentHostsCountCache.SetDefault(hostName, currentHostsCount)
	}
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
	reqUrl := client.uriServices + BuildQueryParams(params)
	response, err := client.sendRequest(nil, http.MethodGet, reqUrl, nil)
	if err != nil {
		log.Warn("|gwClient.go| : [GetServicesByAgent] : Unable to get GW services: ", err)
		return nil, err
	}
	var gwServices GwServices
	err = json.Unmarshal(response, &gwServices)
	if err != nil {
		log.Warn("|gwClient.go| : [GetServicesByAgent] : Unable to parse received GW services: ", err)
		return nil, err
	}
	return &gwServices, nil
}

func (client *GWClient) GetHostGroupsByHostNamesAndAppType(hostNames []string, appType string) (*GwHostGroups, error) {
	if hostNames == nil || len(hostNames) == 0 {
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
	reqUrl := client.uriHostGroups + BuildQueryParams(params)
	response, err := client.sendRequest(nil, http.MethodGet, reqUrl, nil)
	if err != nil {
		log.Error("|gwClient.go| : [GetHostGroupsByHostNamesAndAppType] : Unable to get GW host groups: ", err)
		return nil, err
	}
	var gwHostGroups GwHostGroups
	err = json.Unmarshal(response, &gwHostGroups)
	if err != nil {
		log.Error("|gwClient.go| : [GetHostGroupsByHostNamesAndAppType] : Unable to parse received GW host groups: ", err)
		return nil, err
	}
	return &gwHostGroups, nil
}

func (client *GWClient) checkLicenseForHostLimit(hostsToAllocate int) (bool, error) {
	host := "unknown"
	if client.GWConnection != nil {
		host = client.GWConnection.HostName
	}
	log.Info("Checking license at ", host)

	params := make(map[string]string)
	params["allocate"] = strconv.Itoa(hostsToAllocate)
	client.buildURIs()
	reqUrl := client.uriLicenseCheck + BuildQueryParams(params)
	response, err := client.sendRequest(nil, http.MethodGet, reqUrl, nil)
	if err != nil {
		log.Error("|gwClient.go| : [checkLicenseForHostLimit] : Unable to check license at ", host, ": ", err)
		return false, errors.New("failed to check license at " + host + ": unable to get license")
	}
	var license LicenseCheck
	err = json.Unmarshal(response, &license)
	if err != nil {
		log.Error("|gwClient.go| : [checkLicenseForHostLimit] : Unable to parse received license at ", host, ": ", err)
		return false, errors.New("failed to check license at " + host + ": unable to get license")
	}
	if !license.isOkToSendInventory() {
		hostSpace := license.LimitDevices - license.Devices
		if hostSpace < 0 {
			hostSpace = 0
		}
		log.Error("|gwClient.go| : [checkLicenseForHostLimit] : Host allocation over hard limit at " + host +
			": hosts limit " + strconv.Itoa(license.LimitDevices) +
			", hosts monitored " + strconv.Itoa(license.Devices) +
			", license space for " + strconv.Itoa(hostSpace) + " hosts" +
			", hosts to allocate " + strconv.Itoa(hostsToAllocate))
		return false, errors.New("host allocation over hard limit at " + host +
			": hosts to allocate " + strconv.Itoa(hostsToAllocate) +
			" but only license space for " + strconv.Itoa(hostSpace))
	}
	return true, nil
}

func (client *GWClient) getLastSentHostsCount(agentID string) int {
	gwServices, err := client.GetServicesByAgent(agentID)
	if err != nil || gwServices == nil {
		log.Warn("|gwClient.go| : [getLastSentHostsCount] : Unable to get GW hosts to init  last hosts sent count.")
		if err != nil {
			log.Warn("|gwClient.go| : [getLastSentHostsCount] : ", err)
		} else {
			log.Warn("|gwClient.go| : [getLastSentHostsCount] : Response is nil.")
		}
		return 0
	}
	hostNames := make(map[string]struct{})
	for _, gwService := range gwServices.Services {
		if _, exists := hostNames[gwService.HostName]; !exists {
			hostNames[gwService.HostName] = struct{}{}
		}
	}
	return len(hostNames)
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

	statusCode, byteResponse, err := SendRequestWithContext(ctx, httpMethod, reqURL, headers, nil, payload)

	logEntry := log.With(log.Fields{
		"error":      err,
		"statusCode": statusCode,
	}).WithDebug(log.Fields{
		"response": string(byteResponse),
		"headers":  headers,
		"payload":  string(payload),
		"reqURL":   reqURL,
	})
	logEntryLevel := log.InfoLevel

	if statusCode == 401 {
		logEntry.Log(logEntryLevel, "GWClient: sendRequest: reconnect")

		if err := client.Connect(); err != nil {
			log.With(log.Fields{"error": err}).
				Log(log.ErrorLevel, "GWClient: reconnect error")
			return nil, err
		}

		headers["GWOS-API-TOKEN"] = client.token
		statusCode, byteResponse, err = SendRequestWithContext(ctx, httpMethod, reqURL, headers, nil, payload)

		logEntry = log.With(log.Fields{
			"error":      err,
			"statusCode": statusCode,
		}).WithDebug(log.Fields{
			"response": string(byteResponse),
			"headers":  headers,
			"payload":  string(payload),
			"reqURL":   reqURL,
		})
	}

	defer logEntry.Log(logEntryLevel, "GWClient: sendRequest")

	if err != nil {
		logEntryLevel = log.ErrorLevel
		return nil, err
	}
	if statusCode == 401 {
		logEntryLevel = log.WarnLevel
		return nil, fmt.Errorf("%w: %v", ErrUnauthorized, string(byteResponse))
	}
	if statusCode == 502 || statusCode == 504 {
		logEntryLevel = log.WarnLevel
		return nil, fmt.Errorf("%w: %v", ErrGateway, string(byteResponse))
	}
	if statusCode == 503 {
		logEntryLevel = log.WarnLevel
		return nil, fmt.Errorf("%w: %v", ErrSynchronizer, string(byteResponse))
	}
	if statusCode != 200 {
		logEntryLevel = log.WarnLevel
		return nil, fmt.Errorf("%w: %v", ErrUndecided, string(byteResponse))
	}
	return byteResponse, nil
}

func (client *GWClient) buildURIs() {
	client.Once.Do(func() {
		uriConnect := buildURI(client.GWConnection.HostName, GWEntrypointConnect)
		if !client.LocalConnection {
			uriConnect = buildURI(client.GWConnection.HostName, GWEntrypointConnectRemote)
		}
		uriDisconnect := buildURI(client.GWConnection.HostName, GWEntrypointDisconnect)
		uriClearInDowntime := buildURI(client.GWConnection.HostName, GWEntrypointClearInDowntime)
		uriSetInDowntime := buildURI(client.GWConnection.HostName, GWEntrypointSetInDowntime)
		uriSendEvents := buildURI(client.GWConnection.HostName, GWEntrypointSendEvents)
		uriSendEventsAck := buildURI(client.GWConnection.HostName, GWEntrypointSendEventsAck)
		uriSendEventsUnack := buildURI(client.GWConnection.HostName, GWEntrypointSendEventsUnack)
		uriSendResourceWithMetrics := buildURI(client.GWConnection.HostName, GWEntrypointSendResourceWithMetrics)
		uriSendResourceWithMetricsDynamic := buildURI(client.GWConnection.HostName, GWEntrypointSendResourceWithMetricsDynamic)
		uriSynchronizeInventory := buildURI(client.GWConnection.HostName, GWEntrypointSynchronizeInventory)
		uriServices := buildURI(client.GWConnection.HostName, GWEntrypointServices)
		uriHostGroups := buildURI(client.GWConnection.HostName, GWEntrypointHostgroups)
		uriLicenseCheck := buildURI(client.GWConnection.HostName, GWEntrypointLicenseCheck)
		uriValidateToken := buildURI(client.GWConnection.HostName, GWEntrypointValidateToken)
		client.Mutex.Lock()
		client.uriConnect = uriConnect
		client.uriDisconnect = uriDisconnect
		client.uriClearInDowntime = uriClearInDowntime
		client.uriSetInDowntime = uriSetInDowntime
		client.uriSendEvents = uriSendEvents
		client.uriSendEventsAck = uriSendEventsAck
		client.uriSendEventsUnack = uriSendEventsUnack
		client.uriSendResourceWithMetrics = uriSendResourceWithMetrics
		client.uriSendResourceWithMetricsDynamic = uriSendResourceWithMetricsDynamic
		client.uriSynchronizeInventory = uriSynchronizeInventory
		client.uriServices = uriServices
		client.uriHostGroups = uriHostGroups
		client.uriLicenseCheck = uriLicenseCheck
		client.uriValidateToken = uriValidateToken
		client.Mutex.Unlock()
	})
}

func buildURI(hostname, entrypoint string) string {
	s := strings.TrimSuffix(strings.TrimRight(hostname, "/"), "/api")
	if !strings.HasPrefix(s, "http") {
		s = "https://" + s
	}
	return fmt.Sprintf("%s/%s", s, strings.TrimLeft(entrypoint, "/"))
}
