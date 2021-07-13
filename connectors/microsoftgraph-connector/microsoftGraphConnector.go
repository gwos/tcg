package main

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"github.com/PaesslerAG/jsonpath"
	"github.com/gwos/tcg/connectors"
	"github.com/gwos/tcg/log"
	"github.com/gwos/tcg/milliseconds"
	"github.com/gwos/tcg/transit"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"time"
)

type MicrosoftGraphView string

const (
	ViewServices MicrosoftGraphView = "ViewServices"
)

// ExtConfig defines the MonitorConnection extensions configuration
// extended with general configuration fields
type ExtConfig struct {
	TenantId     string                                                     `json:"officeTenantId"`
	ClientId     string                                                     `json:"officeClientId"`
	ClientSecret string                                                     `json:"officeClientSecret"`
	Ownership    transit.HostOwnershipType                                  `json:"ownership,omitempty"`
	Groups       []transit.ResourceGroup                                    `json:"groups"`
	Views        map[MicrosoftGraphView]map[string]transit.MetricDefinition `json:"views"`
}

type MicrosoftGraphConnector struct {
	config      ExtConfig
	ctx         context.Context
	officeToken string
	graphToken  string
}

type MicrosoftGraphResource struct {
	Name     string
	Type     transit.ResourceType
	Status   transit.MonitorStatus
	Message  string
	Labels   map[string]string
	Services map[string]transit.DynamicMonitoredService
}

type ODataServicePayload struct {
	Context  string         `json:"@odata.context"`
	Services []ODataService `json:"value"`
}

type ODataService struct {
	DisplayName string
	Id          string
	// Features []Feature
}

type ODataStatus struct {
	Context  string          `json:"@odata.context"`
	Services []ServiceStatus `json:"value"`
}

type ServiceStatus struct {
	Id                  string
	WorkloadDisplayName string
	Status              string
	StatusDisplayName   string
	StatusTime          string
}

type AuthRecord struct {
	GrantType    string `json:"grant_type"`
	ClientID     string `json:"client_id"`
	ClientSecret string `json:"client_secret"`
	Resource     string `json:"resource"`
}

var httpClient = &http.Client{
	Transport: &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	},
}

var (
	tenantID     = "" // The Directory ID from Azure AD
	clientID     = "" // The Application ID of the registered app
	clientSecret = "" // The secret key of the registered app
	enableOneDriveMetrics = false
	enableLicensingMetrics = false
	enableSharePointMetrics = false
	enableEmailMetrics = false
	sharePointSite = ""
	sharePointSubSite = ""
	outlookEmailAddress = ""
)

const (
	officeResource    = "https://manage.office.com"
	graphResource     = "https://graph.microsoft.com"
	officeEndPoint    = "https://manage.office.com/api/v1.0/"
	servicesPath      = "/ServiceComms/Services"
	currentStatusPath = "/ServiceComms/CurrentStatus"
	microsoftGroup    = "Microsoft Apps"
	office365App      = "Office365"
	interacApp        = "Interac_OS365"
)

func (connector *MicrosoftGraphConnector) SetCredentials(tenant string, client string, secret string) {
	tenantID = tenant
	clientID = client
	clientSecret = secret
}

func (connector *MicrosoftGraphConnector) SetOptions(oneDriveMetrics bool, licensingMetrics bool, sharePointMetrics bool, emailMetrics bool,
		sharePointSiteParam string, sharePointSubSiteParam string, outlookEmailAddressParam string) {
	enableOneDriveMetrics = oneDriveMetrics
	enableLicensingMetrics = licensingMetrics
	enableSharePointMetrics = sharePointMetrics
	enableEmailMetrics = emailMetrics
	sharePointSite = sharePointSiteParam
	sharePointSubSite = sharePointSubSiteParam
	outlookEmailAddress = outlookEmailAddressParam
}

func (connector *MicrosoftGraphConnector) Initialize() error {
	if connector.officeToken != "" {
		return nil
	}
	token, err := login(tenantID, clientID, clientSecret, officeResource)
	if err != nil {
		return nil
	}
	connector.officeToken = token
	token, err = login(tenantID, clientID, clientSecret, graphResource)
	if err != nil {
		return nil
	}
	connector.graphToken = token
	log.Info(fmt.Sprintf("initialized MS Graph connection with  %s and %s", officeResource, graphResource))
	return nil
}

func (connector *MicrosoftGraphConnector) Ping() error {
	return nil
}

func (connector *MicrosoftGraphConnector) Shutdown() {
}

func login(tenantID string, clientID string, clientSecret string, resource string) (string, error) {
	var request *http.Request
	var response *http.Response

	endPoint := "https://login.microsoftonline.com/" + tenantID + "/oauth2/token"
	auth := AuthRecord{
		GrantType:    "client_credentials",
		ClientID:     clientID,
		ClientSecret: clientSecret,
		Resource:     resource,
	}
	form := url.Values{}
	form.Add("grant_type", "client_credentials")
	form.Add("client_secret", auth.ClientSecret)
	form.Add("client_id", auth.ClientID)
	form.Add("resource", auth.Resource)
	byteBody := []byte(form.Encode())
	var body io.Reader
	body = bytes.NewBuffer(byteBody)
	request, err := http.NewRequest(http.MethodPost, endPoint, body)
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	response, err = httpClient.Do(request)
	if err != nil {
		return "", err
	}
	defer response.Body.Close()
	responseBody, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return "", err
	}
	v := interface{}(nil)
	json.Unmarshal(responseBody, &v)
	token, err := jsonpath.Get("$.access_token", v)
	if err != nil {
		return "", err
	}
	return token.(string), nil
}

// Collect inventory and metrics for all kinds of Kubernetes resources. Sort resources into groups and return inventory of host resources and inventory of groups
func (connector *MicrosoftGraphConnector) Collect(cfg *ExtConfig) ([]transit.DynamicInventoryResource, []transit.DynamicMonitoredResource, []transit.ResourceGroup) {
	connector.Initialize()
	// gather inventory and Metrics
	monitoredState := make(map[string]MicrosoftGraphResource)
	groups := make(map[string]transit.ResourceGroup)
	msGroup := transit.ResourceGroup{
		GroupName: microsoftGroup,
		Type:      transit.HostGroup,
		Resources: make([]transit.MonitoredResourceRef, 0),
	}
	connector.collectInventory(monitoredState, &msGroup)
	connector.collectStatus(monitoredState[office365App].Services)
	connector.collectBuiltins(monitoredState, &msGroup)
	groups[microsoftGroup] = msGroup

	inventory := make([]transit.DynamicInventoryResource, len(monitoredState))
	monitored := make([]transit.DynamicMonitoredResource, len(monitoredState))
	hostGroups := make([]transit.ResourceGroup, len(groups))
	index := 0
	for _, resource := range monitoredState {
		// convert inventory
		services := make([]transit.DynamicInventoryService, len(resource.Services))
		serviceIndex := 0
		for _, service := range resource.Services {
			services[serviceIndex] = connectors.CreateInventoryService(service.Name, service.Owner)
			serviceIndex = serviceIndex + 1
		}
		inventory[index] = connectors.CreateInventoryResource(resource.Name, services)
		// convert monitored state
		mServices := make([]transit.DynamicMonitoredService, len(resource.Services))
		serviceIndex = 0
		for _, service := range resource.Services {
			mServices[serviceIndex] = service
			serviceIndex = serviceIndex + 1
		}
		monitored[index] = transit.DynamicMonitoredResource{
			BaseResource: transit.BaseResource{
				BaseTransitData: transit.BaseTransitData{
					Name: resource.Name,
					Type: resource.Type,
				},
			},
			Status:           resource.Status,
			LastCheckTime:    milliseconds.MillisecondTimestamp{time.Now()},
			NextCheckTime:    milliseconds.MillisecondTimestamp{time.Now()}, // TODO: interval
			LastPlugInOutput: resource.Message,
			Services:         mServices,
		}
		index = index + 1
	}
	index = 0
	for _, group := range groups {
		hostGroups[index] = group
		index = index + 1
	}
	return inventory, monitored, hostGroups
}

func (connector *MicrosoftGraphConnector) collectBuiltins(
		monitoredState map[string]MicrosoftGraphResource, group *transit.ResourceGroup) error {

	hostResource := MicrosoftGraphResource{
		Name:    interacApp,
		Type:    transit.Host,
		Status:  transit.HostUp,
		Message: "UP - Healthy",
		// Labels:   labels,
		Services: make(map[string]transit.DynamicMonitoredService),
	}
	monitoredState[interacApp] = hostResource
	group.Resources = append(group.Resources, transit.MonitoredResourceRef{
		Name:  hostResource.Name,
		Owner: group.GroupName,
		Type:  transit.Host,
	})

	// create one Drive metrics
	if enableOneDriveMetrics {
		serviceName := "OneDrive for Business"
		serviceProperties := make(map[string]interface{})
		serviceProperties["isGraphed"] = true
		monitoredService, _ := connectors.CreateService(serviceName, interacApp, []transit.TimeSeries{}, serviceProperties)
		OneDrive(monitoredService, connector.graphToken)
		monitoredService.LastPlugInOutput = fmt.Sprintf("One Drive free space is %f%%",
			monitoredService.Metrics[2].Value.DoubleValue)
		if monitoredService != nil {
			hostResource.Services[monitoredService.Name] = *monitoredService
		}
	}

	// create License metrics
	if enableLicensingMetrics {
		serviceName2 := "License Activities"
		serviceProperties2 := make(map[string]interface{})
		serviceProperties2["isGraphed"] = true
		monitoredService2, _ := connectors.CreateService(serviceName2, interacApp, []transit.TimeSeries{}, serviceProperties2)
		AddonLicenseMetrics(monitoredService2, connector.graphToken)
		monitoredService2.Status = transit.ServiceOk
		monitoredService2.LastPlugInOutput = fmt.Sprintf("Using %.1f licenses of %.1f",
			monitoredService2.Metrics[1].Value.DoubleValue, monitoredService2.Metrics[0].Value.DoubleValue)
		if monitoredService2 != nil {
			hostResource.Services[monitoredService2.Name] = *monitoredService2
		}
	}

	// create SharePoint metrics
	if enableSharePointMetrics {
		serviceName3 := "SharePoint Online"
		serviceProperties3 := make(map[string]interface{})
		serviceProperties3["isGraphed"] = true
		monitoredService3, _ := connectors.CreateService(serviceName3, interacApp, []transit.TimeSeries{}, serviceProperties3)
		SharePoint(monitoredService3, connector.graphToken)
		monitoredService3.LastPlugInOutput = fmt.Sprintf("SharePoint free space is %f%%", monitoredService3.Metrics[2].Value.DoubleValue)
		if monitoredService3 != nil {
			hostResource.Services[monitoredService3.Name] = *monitoredService3
		}
	}

	// create email metrics
	if enableEmailMetrics {
		serviceName4 := "Emails"
		serviceProperties4 := make(map[string]interface{})
		serviceProperties4["isGraphed"] = true
		monitoredService4, _ := connectors.CreateService(serviceName4, interacApp, []transit.TimeSeries{}, serviceProperties4)
		Emails(monitoredService4, connector.graphToken)
		monitoredService4.LastPlugInOutput = fmt.Sprintf("%.1f Emails unread",
			monitoredService4.Metrics[0].Value.DoubleValue)
		if monitoredService4 != nil {
			hostResource.Services[monitoredService4.Name] = *monitoredService4
		}
	}
	return nil
}

func (connector *MicrosoftGraphConnector) collectInventory(
	monitoredState map[string]MicrosoftGraphResource, group *transit.ResourceGroup) error {

	body, err := ExecuteRequest(officeEndPoint+tenantID+servicesPath, connector.officeToken)
	if err != nil {
		return err
	}
	hostResource := MicrosoftGraphResource{
		Name:    office365App,
		Type:    transit.Host,
		Status:  transit.HostUp,
		Message: "UP - Healthy",
		// Labels:   labels,
		Services: make(map[string]transit.DynamicMonitoredService),
	}
	monitoredState[office365App] = hostResource
	group.Resources = append(group.Resources, transit.MonitoredResourceRef{
		Name:  hostResource.Name,
		Owner: group.GroupName,
		Type:  transit.Host,
	})
	odata := ODataServicePayload{}
	//randomizer := rand.New(rand.NewSource(time.Now().UnixNano()))
	json.Unmarshal(body, &odata)
	for _, ods := range odata.Services {
		metricBuilder := connectors.MetricBuilder{
			Name:     ods.DisplayName,
			Value:    0,
			UnitType: transit.UnitCounter,
			// TODO: add these after merge
			//ComputeType: metricDefinition.ComputeType,
			//Expression:  metricDefinition.Expression,
			Warning:  -1,
			Critical: -1,
			Graphed:  true, // TODO: calculate this from configs
		}
		monitoredService, _ := connectors.BuildServiceForMetric(hostResource.Name, metricBuilder)
		//switch ods.DisplayName {
		//case "Microsoft 365 Apps":
		//	AddonLicenseMetrics(monitoredService, connector, cfg, connector.graphToken)
		//case "OneDrive for Business":
		//	OneDrive(monitoredService, connector, cfg, connector.graphToken)
		//}
		if monitoredService != nil {
			hostResource.Services[metricBuilder.Name] = *monitoredService
		}
	}
	return nil
}

func (connector *MicrosoftGraphConnector) collectStatus(monitoredServices map[string]transit.DynamicMonitoredService) error {
	body, err := ExecuteRequest(officeEndPoint+tenantID+currentStatusPath, connector.officeToken)
	if err != nil {
		return err
	}
	odata := ODataStatus{}
	json.Unmarshal(body, &odata)
	for _, ods := range odata.Services {
		if monitoredService, ok := monitoredServices[ods.WorkloadDisplayName]; ok {
			monitoredService.Status, monitoredService.LastPlugInOutput = connector.translateServiceStatus(ods.Status)
			monitoredServices[ods.WorkloadDisplayName] = monitoredService
		}
		// TODO: get additional metrics
	}
	return nil
}

func (connector *MicrosoftGraphConnector) translateServiceStatus(odStatus string) (transit.MonitorStatus, string) {
	var message string = "Service Status is Unknown"
	var status transit.MonitorStatus = transit.ServiceUnknown
	switch odStatus {
	case "ServiceRestored", "ServiceOperational":
		status = transit.ServiceOk
		message = "OK - Healthy"
	case "ServiceInterruption":
		status = transit.ServiceUnscheduledCritical
		message = "An issue occurred that affects the ability for users to access the service"
		// TODO: look up message
	case "Investigating", "VerifyingService":
		status = transit.ServiceWarning
		message = "We're aware of a potential issue and are gathering more information about what's going on and the scope of impact"
	case "ServiceDegradation":
		status = transit.ServiceUnscheduledCritical
		message = "We've confirmed that there is an issue that may affect use of this service"
		// TODO: look up message
	case "RestoringService":
		status = transit.ServiceWarning
		message = "The cause of the issue has been identified, we know what corrective action to take and are in the process of bringing the service back to a healthy state."
	case "ExtendedRecovery":
		status = transit.ServiceWarning
		message = "A corrective action is in progress to restore service to most users but will take some time to reach all the affected systems"
	}
	return status, message
}

func ExecuteRequest(graphUri string, token string) ([]byte, error) {
	request, _ := http.NewRequest("GET", graphUri, nil)
	request.Header.Set(	"accept", "application/json; odata.metadata=full")
	request.Header.Set("Authorization", "Bearer " + token)
	response, error := httpClient.Do(request)
	if error != nil {
		return nil, error
	}
	if response.StatusCode != 200 {
		log.Info("[MSGraph Connector]:  Retrying Authentication...")
		connector.officeToken = ""
		connector.graphToken = ""
		connector.Initialize()
		request.Header.Set("Authorization", "Bearer " + connector.officeToken)
		response, error = httpClient.Do(request)
		if error != nil {
			return nil, error
		}
	}
	body, error:= ioutil.ReadAll(response.Body)
	defer response.Body.Close()
	if error != nil {
		return nil, error
	}
	return body, nil
}
