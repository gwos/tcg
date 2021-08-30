package main

import (
	"context"
	"encoding/json"
	"github.com/gin-gonic/gin"
	"github.com/gwos/tcg/services"
	"net/http"
	"strings"
	"time"

	"github.com/gwos/tcg/connectors"
	"github.com/gwos/tcg/milliseconds"
	"github.com/gwos/tcg/transit"
	"github.com/rs/zerolog/log"
)

type MicrosoftGraphView string

const (
	ViewServices   MicrosoftGraphView = "Services"
	ViewOneDrive   MicrosoftGraphView = "OneDrive"
	ViewLicensing  MicrosoftGraphView = "Licensing"
	ViewSharePoint MicrosoftGraphView = "SharePoint"
	ViewEmail      MicrosoftGraphView = "Email"
	ViewSecurity   MicrosoftGraphView = "Security"
	ViewCustom     MicrosoftGraphView = "Custom"
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
	config ExtConfig
	ctx    context.Context
}

type MicrosoftGraphResource struct {
	Name     string
	Type     transit.ResourceType
	Status   transit.MonitorStatus
	Message  string
	Labels   map[string]string
	Services map[string]*transit.MonitoredService
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

type ODataFeatureStatus struct {
	FeatureServiceStatus string
	FeatureDisplayName   string
}

type ServiceStatus struct {
	Id                  string
	WorkloadDisplayName string
	Status              string
	StatusDisplayName   string
	StatusTime          string
	FeatureStatus       []ODataFeatureStatus
}

type AuthRecord struct {
	GrantType    string `json:"grant_type"`
	ClientID     string `json:"client_id"`
	ClientSecret string `json:"client_secret"`
	Resource     string `json:"resource"`
}

var (
	officeToken  string
	graphToken   string
	tenantID     = "" // The Directory ID from Azure AD
	clientID     = "" // The Application ID of the registered app
	clientSecret = "" // The secret key of the registered app
	viewStateMap = map[string]bool{
		string(ViewOneDrive):   false,
		string(ViewLicensing):  false,
		string(ViewSharePoint): false,
		string(ViewEmail):      false,
		string(ViewSecurity):   false,
	}
	sharePointSite      = "username.sharepoint.com"
	sharePointSubSite   = "GWOS"
	outlookEmailAddress = ""
)

const (
	officeResource    = "https://manage.office.com"
	graphResource     = "https://graph.microsoft.com"
	officeEndPoint    = "https://manage.office.com/api/v1.0/"
	servicesPath      = "/ServiceComms/Services"
	currentStatusPath = "/ServiceComms/CurrentStatus"
	microsoftGroup    = "Microsoft Apps"
	office365App      = "Office365Services"
	interacApp        = "Office365Graph"
)

func (connector *MicrosoftGraphConnector) SetCredentials(tenant, client, secret string) {
	tenantID = tenant
	clientID = client
	clientSecret = secret
}

func (connector *MicrosoftGraphConnector) SetOptions(sharePointSiteParam, sharePointSubSiteParam, outlookEmailAddressParam string) {
	sharePointSite = sharePointSiteParam
	sharePointSubSite = sharePointSubSiteParam
	outlookEmailAddress = outlookEmailAddressParam
}

func (connector *MicrosoftGraphConnector) Ping() error {
	return nil
}

func (connector *MicrosoftGraphConnector) Shutdown() {
}

// Collect inventory and metrics for all graph resources. Sort resources into groups and return inventory of host resources and inventory of groups
func (connector *MicrosoftGraphConnector) Collect(cfg *ExtConfig) ([]transit.InventoryResource, []transit.MonitoredResource, []transit.ResourceGroup) {
	log.Info().Msg("Starting collection...")
	_ = Initialize()
	log.Info().Msg("After init...")
	// gather inventory and Metrics
	monitoredState := make(map[string]MicrosoftGraphResource)
	groups := make(map[string]transit.ResourceGroup)
	msGroup := transit.ResourceGroup{
		GroupName: microsoftGroup,
		Type:      transit.HostGroup,
		Resources: make([]transit.MonitoredResourceRef, 0),
	}
	_ = connector.collectInventory(monitoredState, &msGroup)
	_ = connector.collectStatus(monitoredState[office365App].Services)
	_ = connector.collectBuiltins(monitoredState, &msGroup)
	groups[microsoftGroup] = msGroup
	log.Info().Msg("inventory and metrics gathered....")
	inventory := make([]transit.InventoryResource, len(monitoredState))
	monitored := make([]transit.MonitoredResource, len(monitoredState))
	hostGroups := make([]transit.ResourceGroup, len(groups))
	index := 0
	for _, resource := range monitoredState {
		// convert inventory
		srvs := make([]transit.InventoryService, len(resource.Services))
		serviceIndex := 0
		for _, service := range resource.Services {
			srvs[serviceIndex] = connectors.CreateInventoryService(service.Name, service.Owner)
			serviceIndex = serviceIndex + 1
		}
		inventory[index] = connectors.CreateInventoryResource(resource.Name, srvs)
		// convert monitored state
		mServices := make([]transit.MonitoredService, len(resource.Services))
		serviceIndex = 0
		for _, service := range resource.Services {
			mServices[serviceIndex] = *service
			serviceIndex = serviceIndex + 1
		}
		var timestamp = &milliseconds.MillisecondTimestamp{Time: time.Now()}
		monitored[index] = transit.MonitoredResource{
			BaseResource: transit.BaseResource{
				BaseTransitData: transit.BaseTransitData{
					Name: resource.Name,
					Type: resource.Type,
				},
			},
			Status:           resource.Status,
			LastCheckTime:    milliseconds.MillisecondTimestamp{Time: time.Now()},
			NextCheckTime:    milliseconds.MillisecondTimestamp{Time: timestamp.Add(connectors.CheckInterval)},
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
		Services: make(map[string]*transit.MonitoredService),
	}
	monitoredState[interacApp] = hostResource
	group.Resources = append(group.Resources, transit.MonitoredResourceRef{
		Name:  hostResource.Name,
		Owner: group.GroupName,
		Type:  transit.Host,
	})

	// create one Drive metrics
	if viewStateMap[string(ViewOneDrive)] {
		serviceName := "OneDrive for Business"
		serviceProperties := make(map[string]interface{})
		serviceProperties["isGraphed"] = true
		monitoredService, _ := connectors.CreateService(serviceName, interacApp, []transit.TimeSeries{}, serviceProperties)
		err := OneDrive(monitoredService, graphToken)
		if err == nil {
			// monitoredService.LastPlugInOutput = fmt.Sprintf("One Drive free space is %f%%", monitoredService.Metrics[2].Value.DoubleValue)
		} else {
			monitoredService.Status = transit.ServiceUnknown
			if err != nil {
				monitoredService.LastPlugInOutput = err.Error()
			} else {
				monitoredService.LastPlugInOutput = "No OneDrive metrics available"
			}
		}
		if monitoredService != nil {
			hostResource.Services[monitoredService.Name] = monitoredService
		}
	}

	// create License metrics
	if viewStateMap[string(ViewLicensing)] {
		serviceName := "License Activities"
		serviceProperties := make(map[string]interface{})
		serviceProperties["isGraphed"] = true
		monitoredService, _ := connectors.CreateService(serviceName, interacApp, []transit.TimeSeries{}, serviceProperties)
		err := AddonLicenseMetrics(monitoredService, graphToken)
		// TODO: calculate status by threshold
		if err == nil {
			// monitoredService.LastPlugInOutput = fmt.Sprintf("Using %.1f licenses of %.1f", monitoredService.Metrics[0].Value.DoubleValue, monitoredService.Metrics[1].Value.DoubleValue)
		} else {
			if err != nil {
				monitoredService.LastPlugInOutput = err.Error()
			} else {
				monitoredService.LastPlugInOutput = "No licensing metrics available"
			}
		}
		if monitoredService != nil {
			hostResource.Services[monitoredService.Name] = monitoredService
		}
	}

	// create SharePoint metrics
	if viewStateMap[string(ViewSharePoint)] {
		serviceName := "SharePoint Online"
		serviceProperties := make(map[string]interface{})
		serviceProperties["isGraphed"] = true
		monitoredService, _ := connectors.CreateService(serviceName, interacApp, []transit.TimeSeries{}, serviceProperties)
		err := SharePoint(monitoredService, graphToken, sharePointSite, sharePointSubSite) // TODO: params
		if err == nil {
			// monitoredService.LastPlugInOutput = fmt.Sprintf("SharePoint free space is %f%%", monitoredService.Metrics[2].Value.DoubleValue)
		} else {
			monitoredService.Status = transit.ServiceUnknown
			if err != nil {
				monitoredService.LastPlugInOutput = err.Error()
			} else {
				monitoredService.LastPlugInOutput = "No SharePoint metrics available"
			}
		}
		if monitoredService != nil {
			hostResource.Services[monitoredService.Name] = monitoredService
		}
	}

	// Create Email metrics
	if viewStateMap[string(ViewEmail)] {
		serviceName := "Emails"
		serviceProperties := make(map[string]interface{})
		serviceProperties["isGraphed"] = true
		monitoredService, _ := connectors.CreateService(serviceName, interacApp, []transit.TimeSeries{}, serviceProperties)
		err := Emails(monitoredService, graphToken, outlookEmailAddress)
		if err == nil {
			// monitoredService.LastPlugInOutput = fmt.Sprintf("%.1f Emails unread", monitoredService.Metrics[0].Value.DoubleValue)
		} else {
			monitoredService.Status = transit.ServiceUnknown
			if err != nil {
				monitoredService.LastPlugInOutput = err.Error()
			} else {
				monitoredService.LastPlugInOutput = "No EMAIL metrics available"
			}
		}
		if monitoredService != nil {
			hostResource.Services[monitoredService.Name] = monitoredService
		}
	}

	// Create Security metrics
	if viewStateMap[string(ViewSecurity)] {
		serviceName := "SecurityIndicators"
		serviceProperties := make(map[string]interface{})
		serviceProperties["isGraphed"] = true
		monitoredService, _ := connectors.CreateService(serviceName, interacApp, []transit.TimeSeries{}, serviceProperties)
		err := SecurityAssessments(monitoredService, graphToken)
		if err == nil {
			// monitoredService.LastPlugInOutput = fmt.Sprintf("%.1f Emails unread", monitoredService.Metrics[0].Value.DoubleValue)
		} else {
			if err != nil {
				monitoredService.LastPlugInOutput = err.Error()
			} else {
				monitoredService.LastPlugInOutput = "No Security metrics available"
			}
		}
		if monitoredService != nil {
			hostResource.Services[monitoredService.Name] = monitoredService
		}
	}
	count := 0
	for _, service := range hostResource.Services {
		if service.Status == transit.ServiceUnknown {
			count = count + 1
		}
	}
	if count == len(hostResource.Services) {
		hostResource.Status = transit.HostUnreachable
		hostResource.Message = "All services for this host are in unknown status"
	}
	return nil
}

func (connector *MicrosoftGraphConnector) collectInventory(
	monitoredState map[string]MicrosoftGraphResource, group *transit.ResourceGroup) error {

	body, err := ExecuteRequest(officeEndPoint+tenantID+servicesPath, officeToken)
	if err != nil {
		return err
	}
	hostResource := MicrosoftGraphResource{
		Name:    office365App,
		Type:    transit.Host,
		Status:  transit.HostUp,
		Message: "UP - Healthy",
		// Labels:   labels,
		Services: make(map[string]*transit.MonitoredService),
	}
	monitoredState[office365App] = hostResource
	group.Resources = append(group.Resources, transit.MonitoredResourceRef{
		Name:  hostResource.Name,
		Owner: group.GroupName,
		Type:  transit.Host,
	})
	odata := ODataServicePayload{}
	//randomizer := rand.New(rand.NewSource(time.Now().UnixNano()))
	_ = json.Unmarshal(body, &odata)
	for _, ods := range odata.Services {
		monitoredService, _ := connectors.CreateService(ods.DisplayName, hostResource.Name)
		if monitoredService != nil {
			hostResource.Services[ods.DisplayName] = monitoredService
		}
	}
	if len(odata.Services) == 0 {
		hostResource.Status = transit.HostUnreachable
		hostResource.Message = "Zero services found. Services are"
	}
	return nil
}

func (connector *MicrosoftGraphConnector) collectStatus(monitoredServices map[string]*transit.MonitoredService) error {
	body, err := ExecuteRequest(officeEndPoint+tenantID+currentStatusPath, officeToken)
	if err != nil {
		return err
	}
	odata := ODataStatus{}
	_ = json.Unmarshal(body, &odata)
	for _, ods := range odata.Services {
		if monitoredService, ok := monitoredServices[ods.WorkloadDisplayName]; ok {
			monitoredService.Status, monitoredService.LastPlugInOutput = connector.translateServiceStatus(ods.Status)
			monitoredServices[ods.WorkloadDisplayName] = monitoredService
			var upCount float64 = 0
			var totalCount float64 = 0
			for _, ofs := range ods.FeatureStatus {
				switch ofs.FeatureServiceStatus {
				case "ServiceRestored", "ServiceOperational":
					upCount++
				}
				totalCount++
			}
			var upPercent float64 = 0
			if totalCount > 0 {
				upPercent = (upCount / totalCount) * 100.0
			}
			metric := createMetricWithThresholds("features.up.percent", "", upPercent, 80, 60)
			monitoredService.Metrics = append(monitoredService.Metrics, *metric)
		}
	}
	return nil
}

func (connector *MicrosoftGraphConnector) translateServiceStatus(odStatus string) (transit.MonitorStatus, string) {
	var message = "Service Status is Unknown"
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

func containsMetric(metricDefinitions []transit.MetricDefinition, metricName string) (*transit.MetricDefinition, bool) {
	for _, v := range metricDefinitions {
		if v.Name == metricName {
			return &v, true
		}
	}
	return nil, false
}

func containsView(metricDefinitions []transit.MetricDefinition, viewName string) bool {
	for _, v := range metricDefinitions {
		if v.ServiceType == viewName && v.Name != "$view_Template#" {
			return true
		}
	}
	return false
}

// initializeEntrypoints - function for setting entrypoints,
// that will be available through the Server Connector API
func initializeEntrypoints() []services.Entrypoint {
	return []services.Entrypoint{
		{
			URL:    "/suggest/:viewName",
			Method: http.MethodGet,
			Handler: func(c *gin.Context) {
				c.JSON(http.StatusOK, availableMetrics()[c.Param("viewName")])
			},
		},
		{
			URL:    "/suggest/:viewName/:name",
			Method: http.MethodGet,
			Handler: func(c *gin.Context) {
				c.JSON(http.StatusOK, listSuggestions(c.Param("viewName"), c.Param("name")))
			},
		},
	}
}

func listSuggestions(viewName, name string) (result []string) {
	for _, metricName := range availableMetrics()[viewName] {
		if strings.Contains(metricName, name) {
			result = append(result, metricName)
		}
	}

	return result
}

var availableMetrics = func() map[string][]string {
	return map[string][]string{
		"OneDrive": {
			"onedrive.total",
			"onedrive.remaining",
			"onedrive.free",
		},
		"SharePoint": {
			"sharepoint.total",
			"sharepoint.remaining",
		},
		"Licensing": {
			"subscriptions.prepaid",
			"subscriptions.consumed",
		},
		"Email": {
			"unread.emails",
		},
	}
}
