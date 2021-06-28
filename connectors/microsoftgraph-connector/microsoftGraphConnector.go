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
	"math/rand"
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
	AppType   string
	AppName   string
	AgentID   string
	EndPoint  string
	Ownership transit.HostOwnershipType
	Views     map[MicrosoftGraphView]map[string]transit.MetricDefinition
	Groups    []transit.ResourceGroup
}

type MicrosoftGraphConnector struct {
	config     ExtConfig
	ctx        context.Context
	officeToken	string
	graphToken	string
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
	Context string `json:"@odata.context"`
	Services []ODataService `json:"value"`
}

type ODataService struct {
	DisplayName string
	Id string
	// Features []Feature
}

type ODataStatus struct {
	Context string `json:"@odata.context"`
	Services []ServiceStatus `json:"value"`
}

type ServiceStatus struct {
	Id string
	WorkloadDisplayName string
	Status string
	StatusDisplayName string
	StatusTime string
}

type AuthRecord struct {
	GrantType   	string `json:"grant_type"`
	ClientID   		string `json:"client_id"`
	ClientSecret	string `json:"client_secret"`
	Resource		string `json:"resource"`
}

var httpClient = &http.Client{
	Transport: &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	},
}

var (
	tenantID = ""  // The Directory ID from Azure AD
	clientID = ""  // The Application ID of the registered app
	clientSecret = ""  // The secret key of the registered app
)

const (
	officeResource = "https://manage.office.com"
	graphResource = "https://graph.microsoft.com"
	officeEndPoint = "https://manage.office.com/api/v1.0/"
	servicesPath = "/ServiceComms/Services"
	currentStatusPath = "/ServiceComms/CurrentStatus"
	microsoftGroup = "Microsoft Apps"
	office365App = "Office365"
)

func (connector *MicrosoftGraphConnector)  SetCredentials(tenant string, client string, secret string) {
	tenantID = tenant
	clientID = client
	clientSecret = secret
}

func (connector *MicrosoftGraphConnector) Initialize(config ExtConfig) error {
	if connector.officeToken != "" {
		return nil
	}
	token, err := login(tenantID, clientID, clientSecret,  officeResource)
	if err != nil {
		return nil
	}
	connector.officeToken = token
	token, err = login(tenantID, clientID, clientSecret,  graphResource)
	if err != nil {
		return nil
	}
	connector.graphToken = token
	fmt.Printf("initialized MS Graph connection with  %s and %s", officeResource, graphResource)
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
	auth := AuthRecord {
		GrantType: "client_credentials",
		ClientID: clientID,
		ClientSecret: clientSecret,
		Resource: resource,
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
	request.Header.Set(	"Content-Type", "application/x-www-form-urlencoded")
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
	connector.Initialize(*cfg)
	// gather inventory and Metrics
	monitoredState := make(map[string]MicrosoftGraphResource)
	groups := make(map[string]transit.ResourceGroup)
	msGroup := transit.ResourceGroup{
		GroupName: microsoftGroup,
		Type:      transit.HostGroup,
		Resources: make([]transit.MonitoredResourceRef, 0),
	}
	connector.collectInventory(cfg, monitoredState, &msGroup)
	connector.collectStatus(cfg, monitoredState[office365App].Services)
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

func (connector *MicrosoftGraphConnector) collectInventory(
	cfg *ExtConfig, monitoredState map[string]MicrosoftGraphResource, group *transit.ResourceGroup) error {

	request, _ := http.NewRequest(http.MethodGet, officeEndPoint + tenantID + servicesPath, nil)
	request.Header.Set(	"accept", "application/json; odata.metadata=full")
	request.Header.Set("Authorization", "Bearer " + connector.officeToken)
	response, err := httpClient.Do(request)
	if err != nil {
		return err
	}
	if response.StatusCode != 200 {
		log.Error("[MSGraph Connector]:  Retrying Authentication...")
		connector.officeToken = ""
		connector.graphToken = ""
		connector.Initialize(*cfg)
		request.Header.Set("Authorization", "Bearer " + connector.officeToken)

		response, err = httpClient.Do(request)
		if err != nil {
			return err
		}
	}
	responseBody, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return err
	}
	defer response.Body.Close()

	hostResource := MicrosoftGraphResource{
		Name:     office365App,
		Type:     transit.Host,
		Status:   transit.HostUp,
		Message:  "UP - Healthy",
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
	randomizer := rand.New(rand.NewSource(time.Now().UnixNano()))
	json.Unmarshal(responseBody, &odata)
	for _, ods := range odata.Services {
		metricBuilder := connectors.MetricBuilder{
			Name:       ods.DisplayName,
			Value:      randomizer.Intn(2) + 1,
			UnitType:   transit.UnitCounter,
			// TODO: add these after merge
			//ComputeType: metricDefinition.ComputeType,
			//Expression:  metricDefinition.Expression,
			Warning:  -1,
			Critical: -1,
			Graphed: true, // TODO: calculate this from configs
		}
		monitoredService, _ := connectors.BuildServiceForMetric(hostResource.Name, metricBuilder)
		switch ods.DisplayName {
		case "Microsoft 365 Apps":
			AddonLicenseMetrics(monitoredService, connector, cfg, connector.graphToken)
		case "OneDrive for Business":
			MicrosoftDrive(monitoredService, connector, cfg, connector.graphToken)
		}
		if monitoredService != nil {
			hostResource.Services[metricBuilder.Name] = *monitoredService
		}
	}
	return nil
}

func (connector *MicrosoftGraphConnector) collectStatus(cfg *ExtConfig, monitoredServices map[string]transit.DynamicMonitoredService) error {
	request, _ := http.NewRequest(http.MethodGet, officeEndPoint + tenantID + currentStatusPath, nil)
	request.Header.Set(	"accept", "application/json; odata.metadata=full")
	request.Header.Set("Authorization", "Bearer " + connector.officeToken)
	response, err := httpClient.Do(request)
	if err != nil {
		return err
	}
	if response.StatusCode != 200 {
		log.Error("[MSGraph Connector]:  Retrying Authentication...")
		connector.officeToken = ""
		connector.graphToken = ""
		connector.Initialize(*cfg)
		request.Header.Set("Authorization", "Bearer " + connector.officeToken)
		response, err = httpClient.Do(request)
		if err != nil {
			return err
		}
	}
	responseBody, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return err
	}
	defer response.Body.Close()

	odata := ODataStatus{}
	json.Unmarshal(responseBody, &odata)
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
		status = transit.ServicePending
		message = "A corrective action is in progress to restore service to most users but will take some time to reach all the affected systems"
	}
	return status, message
}
