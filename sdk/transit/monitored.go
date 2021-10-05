package transit

import (
	"fmt"
)

type MonitoredInfo struct {
	// Restrict to a Groundwork Monitor Status
	Status MonitorStatus `json:"status"`
	// The last status check time on this resource
	LastCheckTime *Timestamp `json:"lastCheckTime,omitempty"`
	// The next status check time on this resource
	NextCheckTime *Timestamp `json:"nextCheckTime,omitempty"`
	// Nagios plugin output string
	LastPluginOutput string `json:"lastPluginOutput,omitempty"`
}

func (p *MonitoredInfo) SetStatus(s MonitorStatus) {
	p.Status = s
}

func (p *MonitoredInfo) SetLastPluginOutput(s string) {
	p.LastPluginOutput = s
}

func (p *MonitoredInfo) SetLastCheckTime(t *Timestamp) {
	p.LastCheckTime = t
}

func (p *MonitoredInfo) SetNextCheckTime(t *Timestamp) {
	p.NextCheckTime = t
}

// A MonitoredResource defines the current status and services of a resource during a metrics scan.
// Examples include:
//  * nagios host
//  * virtual machine instance
//  * RDS database
//  * storage devices such as disks
//  * cloud resources such as cloud apps, cloud functions(lambdas)
//
// A MonitoredResource is the representation of a specific monitored resource during a metric scan.
// Each MonitoredResource contains list of services (MonitoredService). A MonitoredResource does not have metrics,
// only services.
type MonitoredResource struct {
	BaseResource
	MonitoredInfo
	// Services state collection
	Services []MonitoredService `json:"services"`
}

// String implements Stringer interface
func (p MonitoredResource) String() string {
	return fmt.Sprintf("[%s, %s, %s, %s, %s, %s, %s, %s, %s]",
		p.Name, p.Type, p.Owner, p.Properties,
		p.Status, p.LastCheckTime, p.NextCheckTime, p.LastPluginOutput,
		p.Services,
	)
}

func (p *MonitoredResource) AddService(svc MonitoredService) {
	p.Services = append(p.Services, svc)
}

func (p MonitoredResource) ToInventoryResource() InventoryResource {
	var services []InventoryService
	for _, svc := range p.Services {
		services = append(services, svc.ToInventoryService())
	}
	return InventoryResource{
		BaseResource: p.BaseResource,
		Services:     services,
	}
}

// A MonitoredService represents a Groundwork Service creating during a metrics scan.
// In cloud systems, services are usually modeled as a complex metric definition, with each sampled
// metric variation represented as as single metric time series.
//
// A MonitoredService contains a collection of TimeSeries Metrics.
// MonitoredService collections are attached to a MonitoredResource during a metrics scan.
type MonitoredService struct {
	BaseInfo
	MonitoredInfo
	// metrics
	Metrics []TimeSeries `json:"metrics"`
}

func (p *MonitoredService) AddMetric(t TimeSeries) {
	p.Metrics = append(p.Metrics, t)
}

// String implements Stringer interface
func (p MonitoredService) String() string {
	return fmt.Sprintf("[%s, %s, %s, %s, %s, %s, %s, %s, %s]",
		p.Name, p.Type, p.Owner, p.Properties,
		p.Status, p.LastCheckTime, p.NextCheckTime, p.LastPluginOutput,
		p.Metrics,
	)
}

func (p MonitoredService) ToInventoryService() InventoryService {
	return InventoryService{
		BaseInfo: p.BaseInfo,
	}
}

// ResourcesWithServicesRequest defines SendResourcesWithMetrics payload
type ResourcesWithServicesRequest struct {
	Context   *TracerContext      `json:"context,omitempty"`
	Resources []MonitoredResource `json:"resources"`
	Groups    []ResourceGroup     `json:"groups,omitempty"`
}

func (p *ResourcesWithServicesRequest) AddResource(res MonitoredResource) {
	p.Resources = append(p.Resources, res)
}

func (p *ResourcesWithServicesRequest) AddResourceGroup(gr ResourceGroup) {
	p.Groups = append(p.Groups, gr)
}

func (p *ResourcesWithServicesRequest) SetContext(c TracerContext) {
	if p.Context == nil {
		p.Context = new(TracerContext)
	}
	p.Context.SetContext(c)
}

// String implements Stringer interface
func (p ResourcesWithServicesRequest) String() string {
	return fmt.Sprintf("[%s, %s]",
		p.Context.String(), p.Resources)
}
