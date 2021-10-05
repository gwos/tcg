package transit

import (
	"fmt"
	"reflect"

	"github.com/gwos/tcg/sdk/logper"
)

type BaseInfo struct {
	// The unique name of the resource
	Name string `json:"name"`
	// Type: Required. The resource type of the resource
	// General Nagios Types are hosts, whereas CloudHub can have richer complexity
	Type ResourceType `json:"type"`
	// Owner relationship for associations like hypervisor->virtual machine
	Owner string `json:"owner,omitempty"`
	// CloudHub Categorization of resources
	Category string `json:"category,omitempty"`
	// Optional description of this resource, such as Nagios notes
	Description string `json:"description,omitempty"`
	// Foundation Properties
	Properties map[string]TypedValue `json:"properties,omitempty"`
}

func (p *BaseInfo) SetName(s string) {
	p.Name = s
}

func (p *BaseInfo) SetType(s ResourceType) {
	p.Type = s
}

func (p *BaseInfo) SetOwner(s string) {
	p.Owner = s
}

func (p *BaseInfo) SetCategory(s string) {
	p.Category = s
}

func (p *BaseInfo) SetDescription(s string) {
	p.Description = s
}

func (p *BaseInfo) SetProperty(k string, v interface{}) {
	t := NewTypedValue(v)
	if t == nil {
		logper.Error(nil, "could not set property %s on %s: unsupported value type: %T",
			k, p.Name, reflect.TypeOf(v))
		return
	}
	if p.Properties == nil {
		p.Properties = make(map[string]TypedValue)
	}
	p.Properties[k] = *t
}

func (p *BaseInfo) CreateProperties(properties map[string]interface{}) {
	for k, v := range properties {
		p.SetProperty(k, v)
	}
}

type BaseResource struct {
	BaseInfo
	// Device (usually IP address), leave empty if not available, will default to name
	Device string `json:"device,omitempty"`
}

func (p *BaseResource) SetDevice(s string) {
	p.Device = s
}

func (p BaseResource) ToResourceRef() ResourceRef {
	return ResourceRef{Name: p.Name, Type: p.Type, Owner: p.Owner}
}

// InventoryResource represents a resource that is included in a inventory scan.
// Examples include:
//  * nagios host
//  * virtual machine instance
//  * RDS database
//  * storage devices such as disks
//  * cloud resources such as cloud apps, cloud functions(lambdas)
//
// An InventoryResource is the representation of a specific monitored resource during an inventory scan.
// Each InventoryResource contains list of services (InventoryService) (no metrics are sent).
type InventoryResource struct {
	BaseResource
	// Inventory Service collection
	Services []InventoryService `json:"services"`
}

func (p *InventoryResource) AddService(svc InventoryService) {
	p.Services = append(p.Services, svc)
}

// String implements Stringer interface
func (p InventoryResource) String() string {
	return fmt.Sprintf("[%s, %s, %s, %s, %s, %s, %s, %s]",
		p.Name, p.Type, p.Owner, p.Category, p.Description, p.Properties,
		p.Device, p.Services,
	)
}

// InventoryService represents a Groundwork Service that is included in a inventory scan.
// In cloud systems, services are usually modeled as a complex metric definition, with each sampled
// metric variation represented as as single metric time series. During inventory scans, TCG does not gather metric samples.
//
// InventoryService collections are attached to an InventoryResource during inventory scans.
type InventoryService struct {
	BaseInfo
}

// String implements Stringer interface
func (p InventoryService) String() string {
	return fmt.Sprintf("[%s, %s, %s, %s, %s, %s]",
		p.Name, p.Type, p.Owner, p.Category, p.Description, p.Properties)
}

// ResourceRef references an InventoryResource in a group collection
type ResourceRef struct {
	// The unique name of the resource
	Name string `json:"name"`
	// Type: Optional. The resource type uniquely defining the resource type
	// General Nagios Types are host and service, whereas CloudHub can have richer complexity
	Type ResourceType `json:"type,omitempty"`
	// Owner relationship for associations like host->service
	Owner string `json:"owner,omitempty"`
}

// String implements Stringer interface
func (p ResourceRef) String() string {
	return fmt.Sprintf("[%s, %s, %s]",
		p.Name, p.Type, p.Owner)
}

// ResourceGroup defines group entity
type ResourceGroup struct {
	GroupName   string        `json:"groupName"`
	Type        GroupType     `json:"type"`
	Description string        `json:"description,omitempty"`
	Resources   []ResourceRef `json:"resources"`
}

func (p *ResourceGroup) AddResource(res ResourceRef) {
	p.Resources = append(p.Resources, res)
}

func (p *ResourceGroup) SetName(s string) {
	p.GroupName = s
}

func (p *ResourceGroup) SetType(s GroupType) {
	p.Type = s
}

func (p *ResourceGroup) SetDescription(s string) {
	p.Description = s
}

// String implements Stringer interface
func (p ResourceGroup) String() string {
	return fmt.Sprintf("[%s, %s, %s, %s]",
		p.GroupName, p.Type, p.Description, p.Resources)
}

// InventoryRequest defines SynchronizeInventory payload
type InventoryRequest struct {
	Context       *TracerContext      `json:"context,omitempty"`
	OwnershipType HostOwnershipType   `json:"ownershipType,omitempty"`
	Resources     []InventoryResource `json:"resources"`
	Groups        []ResourceGroup     `json:"groups,omitempty"`
}

func (p *InventoryRequest) AddResource(res InventoryResource) {
	p.Resources = append(p.Resources, res)
}

func (p *InventoryRequest) AddResourceGroup(gr ResourceGroup) {
	p.Groups = append(p.Groups, gr)
}

func (p *InventoryRequest) SetContext(c TracerContext) {
	if p.Context == nil {
		p.Context = new(TracerContext)
	}
	p.Context.SetContext(c)
}

// String implements Stringer interface
func (p InventoryRequest) String() string {
	return fmt.Sprintf("[%s, %s, %s]",
		p.Context.String(), p.Resources, p.Groups)
}
