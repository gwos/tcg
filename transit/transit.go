package transit

import (
	"errors"
	"fmt"
	"strconv"

	"github.com/gwos/tcg/milliseconds"
	"github.com/rs/zerolog/log"
)

// VersionString defines type of constant
type VersionString string

// ModelVersion defines versioning
const (
	ModelVersion VersionString = "1.0.0"
)

// HostOwnershipType defines the host ownership type of inventory.
type HostOwnershipType string

// Take - Always take ownership, can overwrite ownership, aggressive take everything
// Creator - Leave ownership if already owned (owns things it creates, if I didn't create it I don't own it)
// Yield - Always defer ownership - don't want to own it, if someone else comes along, let them own it
const (
	Creator HostOwnershipType = "Creator"
	Take    HostOwnershipType = "Take"
	Yield   HostOwnershipType = "Yield"
)

// MetricKind defines the metric kind of the time series.
type MetricKind string

// MetricKindUnspecified - Do not use this default value.
// Gauge - An instantaneous measurement of a value.
// Delta - The change in a value during a time interval.
// Cumulative - A value accumulated over a time interval. Cumulative
const (
	MetricKindUnspecified MetricKind = "METRIC_KIND_UNSPECIFIED"
	Gauge                            = "GAUGE"
	Delta                            = "DELTA"
	Cumulative                       = "CUMULATIVE"
)

// ValueType defines the data type of the value of a metric
type ValueType string

// Data type of the value of a metric
const (
	IntegerType     ValueType = "IntegerType"
	DoubleType                = "DoubleType"
	StringType                = "StringType"
	BooleanType               = "BooleanType"
	TimeType                  = "TimeType"
	UnspecifiedType           = "UnspecifiedType"
)

// UnitType - Supported units are a subset of The Unified Code for Units of Measure
// (http://unitsofmeasure.org/ucum.html) standard, added as we encounter
// the need for them in monitoring contexts.
type UnitType string

// Supported units
const (
	UnitCounter UnitType = "1"
	PercentCPU           = "%{cpu}"
	KB                   = "KB"
	MB                   = "MB"
	GB                   = "GB"
)

// ComputeType defines CloudHub Compute Types
type ComputeType string

// CloudHub Compute Types
const (
	Query         ComputeType = "Query"
	Regex                     = "Regex"
	Synthetic                 = "Synthetic"
	Informational             = "Informational"
	Performance               = "Performance"
	Health                    = "Health"
)

// MonitorStatus represents Groundwork service monitor status
type MonitorStatus string

// Groundwork Standard Monitored Resource Statuses
const (
	ServiceOk                  MonitorStatus = "SERVICE_OK"
	ServiceWarning                           = "SERVICE_WARNING"
	ServiceUnscheduledCritical               = "SERVICE_UNSCHEDULED_CRITICAL"
	ServicePending                           = "SERVICE_PENDING"
	ServiceScheduledCritical                 = "SERVICE_SCHEDULED_CRITICAL"
	ServiceUnknown                           = "SERVICE_UNKNOWN"
	HostUp                                   = "HOST_UP"
	HostUnscheduledDown                      = "HOST_UNSCHEDULED_DOWN"
	HostPending                              = "HOST_PENDING"
	HostScheduledDown                        = "HOST_SCHEDULED_DOWN"
	HostUnreachable                          = "HOST_UNREACHABLE"
	HostUnchanged                            = "HOST_UNCHANGED"
)

// ResourceType defines the resource type
type ResourceType string

// The resource type uniquely defining the resource type
// General Nagios Types are host and service, whereas CloudHub can have richer complexity
const (
	Host           ResourceType = "host"
	Hypervisor                  = "hypervisor"
	Instance                    = "instance"
	VirtualMachine              = "virtual-machine"
	CloudApp                    = "cloud-app"
	CloudFunction               = "cloud-function"
	LoadBalancer                = "load-balancer"
	Container                   = "container"
	Storage                     = "storage"
	Network                     = "network"
	NetworkSwitch               = "network-switch"
	NetworkDevice               = "network-device"
)

// ServiceType defines the service type
type ServiceType string

// Possible Types
const (
	Process ServiceType = "Process"
	Service             = "Service"
)

// GroupType defines the foundation group type
type GroupType string

// The group type uniquely defining corresponding foundation group type
const (
	HostGroup    GroupType = "HostGroup"
	ServiceGroup           = "ServiceGroup"
	CustomGroup            = "CustomGroup"
)

// MetricSampleType defines TimeSeries Metric Sample Possible Types
type MetricSampleType string

// TimeSeries Metric Sample Possible Types
const (
	Value    MetricSampleType = "Value"
	Warning                   = "Warning"
	Critical                  = "Critical"
	Min                       = "Min"
	Max                       = "Max"
)

// TimeInterval defines a closed time interval. It extends from the start time
// to the end time, and includes both: [startTime, endTime]. Valid time
// intervals depend on the MetricKind of the metric value. In no case
// can the end time be earlier than the start time.
// For a GAUGE metric, the StartTime value is technically optional; if
// no value is specified, the start time defaults to the value of the
// end time, and the interval represents a single point in time. Such an
// interval is valid only for GAUGE metrics, which are point-in-time
// measurements.
// For DELTA and CUMULATIVE metrics, the start time must be earlier
// than the end time.
// In all cases, the start time of the next interval must be at least a
// microsecond after the end time of the previous interval.  Because the
// interval is closed, if the start time of a new interval is the same
// as the end time of the previous interval, data written at the new
// start time could overwrite data written at the previous end time.
type TimeInterval struct {
	// EndTime: Required. The end of the time interval.
	EndTime milliseconds.MillisecondTimestamp `json:"endTime,omitempty"`

	// StartTime: Optional. The beginning of the time interval. The default
	// value for the start time is the end time. The start time must not be
	// later than the end time.
	StartTime milliseconds.MillisecondTimestamp `json:"startTime,omitempty"`
}

func (value TimeInterval) String() string {
	return fmt.Sprintf("[%s, %s]",
		value.EndTime.String(),
		value.StartTime.String(),
	)
}

// TypedValue defines a single strongly-typed value.
type TypedValue struct {
	ValueType ValueType `json:"valueType"`

	// BoolValue: A Boolean value: true or false.
	BoolValue bool `json:"boolValue,omitempty"`

	// DoubleValue: A 64-bit double-precision floating-point number. Its
	// magnitude is approximately &plusmn;10<sup>&plusmn;300</sup> and it
	// has 16 significant digits of precision.
	DoubleValue float64 `json:"doubleValue"`

	// Int64Value: A 64-bit integer. Its range is approximately
	// &plusmn;9.2x10<sup>18</sup>.
	IntegerValue int64 `json:"integerValue"`

	// StringValue: A variable-length string value.
	StringValue string `json:"stringValue,omitempty"`

	// a time stored as full timestamp
	TimeValue *milliseconds.MillisecondTimestamp `json:"timeValue,omitempty"`
}

func (value TypedValue) String() string {
	switch value.ValueType {
	case IntegerType:
		return strconv.FormatInt(value.IntegerValue, 10)
	case StringType:
		return value.StringValue
	case DoubleType:
		return fmt.Sprintf("%f", value.DoubleValue)
	case BooleanType:
		return strconv.FormatBool(value.BoolValue)
	case TimeType:
		return value.TimeValue.String()
	}
	return ""
}

// ThresholdValue describes threshold
type ThresholdValue struct {
	SampleType MetricSampleType `json:"sampleType"`
	Label      string           `json:"label"`
	Value      *TypedValue      `json:"value"`
}

func (thresholdValue ThresholdValue) String() string {
	return fmt.Sprintf("[%s, %s, %s]",
		thresholdValue.SampleType, thresholdValue.Label, thresholdValue.Value.String())
}

// TimeSeries defines a single Metric Sample, its time interval, and 0 or more thresholds
type TimeSeries struct {
	MetricName string           `json:"metricName"`
	SampleType MetricSampleType `json:"sampleType,omitEmpty"`
	// Interval: The time interval to which the data sample applies. For
	// GAUGE metrics, only the end time of the interval is used. For DELTA
	// metrics, the start and end time should specify a non-zero interval,
	// with subsequent samples specifying contiguous and non-overlapping
	// intervals. For CUMULATIVE metrics, the start and end time should
	// specify a non-zero interval, with subsequent samples specifying the
	// same start time and increasing end times, until an event resets the
	// cumulative value to zero and sets a new start time for the following
	// samples.
	Interval          *TimeInterval     `json:"interval"`
	Value             *TypedValue       `json:"value"`
	Tags              map[string]string `json:"tags,omitempty"`
	Unit              UnitType          `json:"unit,omitempty"`
	Thresholds        *[]ThresholdValue `json:"thresholds,omitempty"`
	MetricComputeType ComputeType       `json:"-"`
	MetricExpression  string            `json:"-"`
}

func (timeSeries TimeSeries) String() string {
	return fmt.Sprintf("[%s, %s, %s, %s, %s, %s, %s]",
		timeSeries.MetricName, timeSeries.SampleType, timeSeries.Interval.String(), timeSeries.Value.String(),
		timeSeries.Tags, timeSeries.Unit, *timeSeries.Thresholds)
}

func (timeSeries *TimeSeries) CreateTag(name string, value string) {
	if timeSeries.Tags == nil {
		timeSeries.Tags = make(map[string]string)
	}
	timeSeries.Tags[name] = value
}

// MetricDescriptor defines a metric type and its schema
type MetricDescriptor struct {
	// Custom Name: Override the resource type with a custom name of the metric descriptor.
	CustomName string `json:"name,omitempty"`

	// Description: A detailed description of the metric, which can be used
	// in documentation.
	Description string `json:"description,omitempty"`

	// DisplayName: A concise name for the metric, which can be displayed in
	// user interfaces. Use sentence case without an ending period, for
	// example "Request count". This field is optional but it is recommended
	// to be set for any metrics associated with user-visible concepts, such
	// as Quota.
	DisplayName string `json:"displayName,omitempty"`

	// Labels: The set of labels that can be used to describe a specific
	// instance of this metric type. For example, the
	// appengine.googleapis.com/http/server/response_latencies metric type
	// has a label for the HTTP response code, response_code, so you can
	// look at latencies for successful responses or just for responses that
	// failed.
	Labels []*LabelDescriptor `json:"labels,omitempty"`

	Thresholds []*ThresholdDescriptor `json:"thresholds,omitempty"`

	// Type: The metric type, including its DNS name prefix. The type is not
	// URL-encoded. All user-defined metric types have the DNS name
	// custom.googleapis.com or external.googleapis.com. Metric types should
	// use a natural hierarchical grouping. For
	// example:
	// "custom.googleapis.com/invoice/paid/amount"
	// "external.googlea
	// pis.com/prometheus/up"
	// "appengine.googleapis.com/http/server/response_
	// latencies"
	//
	Type string `json:"type,omitempty"`

	// Unit: The unit in which the metric value is reported. It is only
	// applicable if the value_type is INT64, DOUBLE, or DISTRIBUTION. The
	// supported units are a subset of The Unified Code for Units of Measure
	// (http://unitsofmeasure.org/ucum.html) standard, added as we encounter
	// the need for them in monitoring contexts.
	Unit UnitType `json:"unit,omitempty"`

	// ValueType: Whether the measurement is an integer, a floating-point
	// number, etc. Some combinations of metric_kind and value_type might
	// not be supported.
	ValueType ValueType `json:"valueType,omitempty"`

	// Groundwork Compute Type such as Synthetic
	ComputeType ComputeType `json:"computeType,omitempty"`

	// Metadata: Optional. Metadata which can be used to guide usage of the
	// metric.
	// Metadata *MetricDescriptorMetadata `json:"metadata,omitempty"`

	// MetricKind: Whether the metric records instantaneous values, changes
	// to a value, etc. Some combinations of metric_kind and value_type
	// might not be supported.
	//
	// Possible values:
	//   "METRIC_KIND_UNSPECIFIED" - Do not use this default value.
	//   "GAUGE" - An instantaneous measurement of a value.
	//   "DELTA" - The change in a value during a time interval.
	//   "CUMULATIVE" - A value accumulated over a time interval. Cumulative
	// measurements in a time series should have the same start time and
	// increasing end times, until an event resets the cumulative value to
	// zero and sets a new start time for the following samples.
	MetricKind MetricKind `json:"metricKind"`
}

func (metricDescriptor MetricDescriptor) String() string {
	return fmt.Sprintf("%s - %s", metricDescriptor.Type, metricDescriptor.CustomName)
}

// LabelDescriptor defines a Label.
type LabelDescriptor struct {
	// Description: A human-readable description for the label.
	Description string `json:"description,omitempty"`

	// Key: The label key.
	Key string `json:"key,omitempty"`

	// ValueType: The type of data that can be assigned to the label.
	//
	// Possible values:
	//   "STRING" - A variable-length string. This is the default.
	//   "BOOL" - Boolean; true or false.
	//   "INT64" - A 64-bit signed integer.
	ValueType ValueType `json:"valueType,omitempty"`
}

func (labelDescriptor LabelDescriptor) String() string {
	return fmt.Sprintf("[%s, %s, %s]", labelDescriptor.Description, labelDescriptor.Key, labelDescriptor.ValueType)
}

// ThresholdDescriptor defines a Threshold
type ThresholdDescriptor struct {
	// Key: The threshold key.
	Key   string `json:"key"`
	Value int32  `json:"value"`
}

func (thresholdDescriptor ThresholdDescriptor) String() string {
	return fmt.Sprintf("[%s, %d]", thresholdDescriptor.Key, thresholdDescriptor.Value)
}

type BaseTransitData struct {
	// The unique name of the resource
	Name string `json:"name,required"`
	// Type: Required. The resource type of the resource
	// General Nagios Types are hosts, whereas CloudHub can have richer complexity
	Type ResourceType `json:"type,required"`
	// Owner relationship for associations like hypervisor->virtual machine
	Owner string `json:"owner,omitempty"`
	// CloudHub Categorization of resources
	Category string `json:"category,omitempty"`
	// Optional description of this resource, such as Nagios notes
	Description string `json:"description,omitempty"`
	// Foundation Properties
	Properties map[string]TypedValue `json:"properties,omitempty"`
}

type BaseResource struct {
	BaseTransitData
	// Device (usually IP address), leave empty if not available, will default to name
	Device string `json:"device,omitempty"`
}

// DynamicInventoryResource represents a resource that is included in a inventory scan.
// Examples include:
//  * nagios host
//  * virtual machine instance
//  * RDS database
//  * storage devices such as disks
//  * cloud resources such as cloud apps, cloud functions(lambdas)
//
// An DynamicInventoryResource is the representation of a specific monitored resource during an inventory scan.
// Each DynamicInventoryResource contains list of services (InventoryService) (no metrics are sent).
type DynamicInventoryResource struct {
	BaseResource
	// Inventory Service collection
	Services []DynamicInventoryService `json:"services"`
}

type InventoryResource struct {
	// The unique Name of the resource
	Name string `json:"name,required"`
	// Type: Required. The resource type of the resource
	// General Nagios Types are hosts, whereas CloudHub can have richer complexity
	Type ResourceType `json:"type,required"`
	// Owner relationship for associations like hypervisor->virtual machine
	Owner string `json:"owner,omitempty"`
	// CloudHub Categorization of resources
	Category string `json:"category,omitempty"`
	// Optional Description of this resource, such as Nagios notes
	Description string `json:"description,omitempty"`
	// Device is needed for detection on Foundation side
	Device string `json:"device,omitempty"`
	// Foundation Properties
	Properties map[string]TypedValue `json:"properties,omitempty"`
	Services   []InventoryService    `json:"services"`
}

// DynamicInventoryService represents a Groundwork Service that is included in a inventory scan.
// In cloud systems, services are usually modeled as a complex metric definition, with each sampled
// metric variation represented as as single metric time series. During inventory scans, TCG does not gather metric samples.
//
// DynamicInventoryService collections are attached to an DynamicInventoryResource during inventory scans.
type DynamicInventoryService struct {
	BaseTransitData
}

type InventoryService struct {
	// The unique name of the resource
	Name string `json:"name,required"`
	// Type: Required. The resource type of the resource
	// General Nagios Types are hosts, whereas CloudHub can have richer complexity
	Type ResourceType `json:"type,required"`
	// Owner relationship for associations like hypervisor->virtual machine
	Owner string `json:"owner,omitempty"`
	// CloudHub Categorization of resources
	Category string `json:"category,omitempty"`
	// Optional description of this resource, such as Nagios notes
	Description string `json:"description,omitempty"`
	// Foundation Properties
	Properties map[string]TypedValue `json:"properties,omitempty"`
}

// A DynamicMonitoredResource defines the current status and services of a resource during a metrics scan.
// Examples include:
//  * nagios host
//  * virtual machine instance
//  * RDS database
//  * storage devices such as disks
//  * cloud resources such as cloud apps, cloud functions(lambdas)
//
// A DynamicMonitoredResource is the representation of a specific monitored resource during a metric scan.
// Each DynamicMonitoredResource contains list of services (MonitoredService). A DynamicMonitoredResource does not have metrics,
// only services.
type DynamicMonitoredResource struct {
	BaseResource
	// Restrict to a Groundwork Monitor Status
	Status MonitorStatus `json:"status,required"`
	// The last status check time on this resource
	LastCheckTime milliseconds.MillisecondTimestamp `json:"lastCheckTime,omitempty"`
	// The next status check time on this resource
	NextCheckTime milliseconds.MillisecondTimestamp `json:"nextCheckTime,omitempty"`
	// Nagios plugin output string
	LastPlugInOutput string `json:"lastPluginOutput,omitempty"`
	// Services state collection
	Services []DynamicMonitoredService `json:"services"`
}

type MonitoredResource struct {
	// The unique name of the resource
	Name string `json:"name,required"`
	// Type: Required. The resource type of the resource
	// General Nagios Types are hosts, whereas CloudHub can have richer complexity
	Type ResourceType `json:"type,required"`
	// Owner relationship for associations like hypervisor->virtual machine
	Owner string `json:"owner,omitempty"`
	// Restrict to a Groundwork Monitor Status
	Status MonitorStatus `json:"status,required"`
	// The last status check time on this resource
	LastCheckTime milliseconds.MillisecondTimestamp `json:"lastCheckTime,omitempty"`
	// The next status check time on this resource
	NextCheckTime milliseconds.MillisecondTimestamp `json:"nextCheckTime,omitempty"`
	// Nagios plugin output string
	LastPlugInOutput string `json:"lastPluginOutput,omitempty"`
	// Foundation Properties
	Properties map[string]TypedValue `json:"properties,omitempty"`
	Services   []MonitoredService    `json:"services"`
}

// A DynamicMonitoredService represents a Groundwork Service creating during a metrics scan.
// In cloud systems, services are usually modeled as a complex metric definition, with each sampled
// metric variation represented as as single metric time series.
//
// A DynamicMonitoredService contains a collection of TimeSeries Metrics.
// MonitoredService collections are attached to a DynamicMonitoredResource during a metrics scan.
type DynamicMonitoredService struct {
	BaseTransitData
	// Restrict to a Groundwork Monitor Status
	Status MonitorStatus `json:"status,required"`
	// The last status check time on this resource
	LastCheckTime milliseconds.MillisecondTimestamp `json:"lastCheckTime,omitempty"`
	// The next status check time on this resource
	NextCheckTime milliseconds.MillisecondTimestamp `json:"nextCheckTime,omitempty"`
	// Nagios plugin output string
	LastPlugInOutput string `json:"lastPluginOutput,omitempty"`
	// metrics
	Metrics []TimeSeries `json:"metrics"`
}

type MonitoredService struct {
	// The unique name of the resource
	Name string `json:"name,required"`
	// Type: Required. The resource type of the resource
	// General Nagios Types are hosts, whereas CloudHub can have richer complexity
	Type ResourceType `json:"type,required"`
	// Owner relationship for associations like hypervisor->virtual machine
	Owner string `json:"owner,omitempty"`
	// CloudHub Categorization of resources
	Category string `json:"category,omitempty"`
	// Optional description of this resource, such as Nagios notes
	Description string `json:"description,omitempty"`
	// Foundation Properties
	Properties map[string]TypedValue `json:"properties,omitempty"`
	// Restrict to a Groundwork Monitor Status
	Status MonitorStatus `json:"status,required"`
	// The last status check time on this resource
	LastCheckTime milliseconds.MillisecondTimestamp `json:"lastCheckTime,omitempty"`
	// The next status check time on this resource
	NextCheckTime milliseconds.MillisecondTimestamp `json:"nextCheckTime,omitempty"`
	// Nagios plugin output string
	LastPlugInOutput string `json:"lastPluginOutput,omitempty"`
	// metrics
	Metrics []TimeSeries `json:"metrics"`
}

func (inventoryResource DynamicInventoryResource) String() string {
	return fmt.Sprintf("[%s, %s, %s, %s, %s, %s, %s, %s]",
		inventoryResource.BaseResource.Name, inventoryResource.BaseResource.Type,
		inventoryResource.BaseResource.Owner, inventoryResource.BaseResource.Category,
		inventoryResource.BaseResource.Description, inventoryResource.BaseResource.Device,
		inventoryResource.BaseResource.Properties, inventoryResource.Services,
	)
}

func (inventoryResource *DynamicInventoryResource) CreateProperty(name string, value TypedValue) {
	if inventoryResource.BaseResource.Properties == nil {
		inventoryResource.BaseResource.Properties = make(map[string]TypedValue)
	}
	inventoryResource.BaseResource.Properties[name] = value
}

func (monitoredService *DynamicMonitoredService) CreateProperties(properties map[string]interface{}) {
	for k, v := range properties {
		var typedValue TypedValue
		err := typedValue.toTypedValue(v)
		if err != nil {
			log.Err(err).Msgf("could not create property %s of service %s", k, monitoredService.Name)
		}
		monitoredService.CreateProperty(k, typedValue)
	}
}

func (value *TypedValue) toTypedValue(v interface{}) error {
	switch v.(type) {
	case bool:
		value.ValueType = BooleanType
		value.BoolValue = v.(bool)
	case float32:
		value.ValueType = DoubleType
		value.DoubleValue = float64(v.(float32))
	case float64:
		value.ValueType = DoubleType
		value.DoubleValue = v.(float64)
	case int:
		value.ValueType = IntegerType
		value.IntegerValue = int64(v.(int))
	case int8:
		value.ValueType = IntegerType
		value.IntegerValue = int64(v.(int8))
	case int16:
		value.ValueType = IntegerType
		value.IntegerValue = int64(v.(int16))
	case int32:
		value.ValueType = IntegerType
		value.IntegerValue = int64(v.(int32))
	case int64:
		value.ValueType = IntegerType
		value.IntegerValue = v.(int64)
	case string:
		value.ValueType = StringType
		value.StringValue = v.(string)
	case milliseconds.MillisecondTimestamp:
		value.ValueType = TimeType
		value.TimeValue = v.(*milliseconds.MillisecondTimestamp)
	default:
		return errors.New("unable to convert to typed value: unsupported type")
	}
	return nil
}

func (inventoryService DynamicInventoryService) String() string {
	return fmt.Sprintf("[%s, %s, %s, %s, %s, %s]",
		inventoryService.Name, inventoryService.Type, inventoryService.Owner,
		inventoryService.Category, inventoryService.Description, inventoryService.Properties,
	)
}

func (inventoryService *DynamicInventoryService) CreateProperty(name string, value TypedValue) {
	if inventoryService.Properties == nil {
		inventoryService.Properties = make(map[string]TypedValue)
	}
	inventoryService.Properties[name] = value
}

func (monitoredResource DynamicMonitoredResource) String() string {
	return fmt.Sprintf("[%s, %s, %s, %s, %s, %s, %s, %s, %s]",
		monitoredResource.BaseResource.Name,
		monitoredResource.BaseResource.Type,
		monitoredResource.BaseResource.Owner,
		monitoredResource.Status,
		monitoredResource.LastCheckTime.String(),
		monitoredResource.NextCheckTime.String(),
		monitoredResource.LastPlugInOutput,
		monitoredResource.BaseResource.Properties,
		monitoredResource.Services,
	)
}

func (monitoredResource *DynamicMonitoredResource) CreateProperty(name string, value TypedValue) {
	if monitoredResource.BaseResource.Properties == nil {
		monitoredResource.BaseResource.Properties = make(map[string]TypedValue)
	}
	monitoredResource.BaseResource.Properties[name] = value
}

func (monitoredService DynamicMonitoredService) String() string {
	return fmt.Sprintf("[%s, %s, %s, %s, %s, %s, %s, %s, %s]",
		monitoredService.Name, monitoredService.Type, monitoredService.Owner, monitoredService.Status,
		monitoredService.LastCheckTime.String(), monitoredService.NextCheckTime.String(),
		monitoredService.LastPlugInOutput, monitoredService.Properties, monitoredService.Metrics,
	)
}

func (monitoredService *DynamicMonitoredService) CreateProperty(name string, value TypedValue) {
	if monitoredService.Properties == nil {
		monitoredService.Properties = make(map[string]TypedValue)
	}
	monitoredService.Properties[name] = value
}

// MonitoredResourceRef references a MonitoredResource in a group collection
type MonitoredResourceRef struct {
	// The unique name of the resource
	Name string `json:"name,required"`
	// Type: Optional. The resource type uniquely defining the resource type
	// General Nagios Types are host and service, whereas CloudHub can have richer complexity
	Type ResourceType `json:"type,omitempty"`
	// Owner relationship for associations like host->service
	Owner string `json:"owner,omitempty"`
}

func (monitoredResourceRef MonitoredResourceRef) String() string {
	return fmt.Sprintf("[%s, %s, %s]",
		monitoredResourceRef.Name,
		monitoredResourceRef.Type,
		monitoredResourceRef.Owner,
	)
}

// TracerContext describes a Transit call
type TracerContext struct {
	AppType    string                            `json:"appType"`
	AgentID    string                            `json:"agentId"`
	TraceToken string                            `json:"traceToken"`
	TimeStamp  milliseconds.MillisecondTimestamp `json:"timeStamp"`
	Version    VersionString                     `json:"version"`
}

func (tracerContext TracerContext) String() string {
	return fmt.Sprintf("[%s, %s, %s, %s, %s]",
		tracerContext.AppType, tracerContext.AgentID, tracerContext.TraceToken,
		tracerContext.TimeStamp, tracerContext.Version,
	)
}

// OperationResult defines API answer
type OperationResult struct {
	Entity   string `json:"entity"`
	Status   string `json:"status"`
	Message  string `json:"message"`
	Location string `json:"location"`
	EntityID int    `json:"entityID"`
}

func (operationResult OperationResult) String() string {
	return fmt.Sprintf("[%s, %s, %s, %s, %d]",
		operationResult.Entity, operationResult.Status, operationResult.Message,
		operationResult.Location, operationResult.EntityID,
	)
}

// OperationResults defines API answer
type OperationResults struct {
	ResourcesAdded   int                `json:"successful"`
	ResourcesDeleted int                `json:"failed"`
	EntityType       string             `json:"entityType"`
	Operation        string             `json:"operation"`
	Warning          int                `json:"warning"`
	Count            int                `json:"count"`
	Results          *[]OperationResult `json:"results"`
}

func (operationResults OperationResults) String() string {
	return fmt.Sprintf("[%d, %d, %s, %s, %d, %d, %s]",
		operationResults.ResourcesAdded, operationResults.ResourcesDeleted, operationResults.EntityType,
		operationResults.Operation, operationResults.Warning, operationResults.Count, *operationResults.Results,
	)
}

// ResourceGroup defines group entity
type ResourceGroup struct {
	GroupName   string                 `json:"groupName,required"`
	Type        GroupType              `json:"type,required"`
	Description string                 `json:"description,omitempty"`
	Resources   []MonitoredResourceRef `json:"resources,required"`
}

func (resourceGroup ResourceGroup) String() string {
	return fmt.Sprintf("[%s, %s, %s, %s]",
		resourceGroup.GroupName, resourceGroup.Type, resourceGroup.Description, resourceGroup.Resources,
	)
}

// DynamicResourcesWithServicesRequest defines SendResourcesWithMetrics payload
type DynamicResourcesWithServicesRequest struct {
	Context   *TracerContext             `json:"context,omitempty"`
	Resources []DynamicMonitoredResource `json:"resources"`
	Groups    []ResourceGroup            `json:"groups,omitempty"`
}

type ResourcesWithServicesRequest struct {
	Context   *TracerContext      `json:"context,omitempty"`
	Resources []MonitoredResource `json:"resources"`
	Groups    []ResourceGroup     `json:"groups,omitempty"`
}

func (resourcesWithServicesRequest DynamicResourcesWithServicesRequest) String() string {
	return fmt.Sprintf("[%s, %s]",
		resourcesWithServicesRequest.Context.String(),
		resourcesWithServicesRequest.Resources,
	)
}

// DynamicInventoryRequest defines SynchronizeInventory payload
type DynamicInventoryRequest struct {
	Context       *TracerContext             `json:"context,omitempty"`
	OwnershipType HostOwnershipType          `json:"ownershipType,omitempty"`
	Resources     []DynamicInventoryResource `json:"resources"`
	Groups        []ResourceGroup            `json:"groups,omitempty"`
}

// InventoryRequest defines SynchronizeInventory payload
type InventoryRequest struct {
	Context       *TracerContext      `json:"context,omitempty"`
	OwnershipType HostOwnershipType   `json:"ownershipType,omitempty"`
	Resources     []InventoryResource `json:"resources"`
	Groups        []ResourceGroup     `json:"groups,omitempty"`
}

func (inventoryRequest DynamicInventoryRequest) String() string {
	return fmt.Sprintf("[%s, %s, %s]",
		inventoryRequest.Context.String(), inventoryRequest.Resources, inventoryRequest.Groups,
	)
}

// IncidentAlert describes alerts received from cloud services
type IncidentAlert struct {
	IncidentID    string                            `json:"incidentId"`
	ResourceName  string                            `json:"resourceName,required"`
	Status        string                            `json:"status"`
	StartedAt     milliseconds.MillisecondTimestamp `json:"startedAt"`
	EndedAt       milliseconds.MillisecondTimestamp `json:"endedAt,omitempty"`
	ConditionName string                            `json:"conditionName"`
	URL           string                            `json:"url,omitempty"`
	Summary       string                            `json:"summary,omitempty"`
}

func (incidentAlert IncidentAlert) String() string {
	return fmt.Sprintf("[%s, %s, %s, %s, %s, %s, %s, %s]",
		incidentAlert.IncidentID, incidentAlert.ResourceName, incidentAlert.Status, incidentAlert.StartedAt.String(),
		incidentAlert.EndedAt.String(), incidentAlert.ConditionName, incidentAlert.URL, incidentAlert.Summary,
	)
}

// GroundworkEventsRequest describes request payload
type GroundworkEventsRequest struct {
	Events []GroundworkEvent `json:"events"`
}

func (groundworkEventsRequest GroundworkEventsRequest) String() string {
	return fmt.Sprintf("[%s]", groundworkEventsRequest.Events)
}

// GroundworkEvent describes event
type GroundworkEvent struct {
	Device              string                            `json:"device,omitempty"`
	Host                string                            `json:"host,required"`
	Service             string                            `json:"service,omitempty"`
	OperationStatus     string                            `json:"operationStatus,omitempty"`
	MonitorStatus       string                            `json:"monitorStatus,required"`
	Severity            string                            `json:"severity,omitempty"`
	ApplicationSeverity string                            `json:"applicationSeverity,omitempty"`
	Component           string                            `json:"component,omitempty"`
	SubComponent        string                            `json:"subComponent,omitempty"`
	Priority            string                            `json:"priority,omitempty"`
	TypeRule            string                            `json:"typeRule,omitempty"`
	TextMessage         string                            `json:"textMessage,omitempty"`
	LastInsertDate      milliseconds.MillisecondTimestamp `json:"lastInsertDate,omitempty"`
	ReportDate          milliseconds.MillisecondTimestamp `json:"reportDate,required"`
	AppType             string                            `json:"appType,required"`
	// Update level attributes (update only)
	MonitorServer     string `json:"monitorServer,omitempty"`
	ConsolidationName string `json:"consolidationName,omitempty"`
	LogType           string `json:"logType,omitempty"`
	ErrorType         string `json:"errorType,omitempty"`
	LoggerName        string `json:"loggerName,omitempty"`
	ApplicationName   string `json:"applicationName,omitempty"`
}

func (groundworkEvent GroundworkEvent) String() string {
	return fmt.Sprintf("[%s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s]",
		groundworkEvent.Device, groundworkEvent.Host, groundworkEvent.Service, groundworkEvent.OperationStatus,
		groundworkEvent.MonitorStatus, groundworkEvent.Severity, groundworkEvent.ApplicationSeverity,
		groundworkEvent.Component, groundworkEvent.SubComponent, groundworkEvent.Priority, groundworkEvent.TypeRule,
		groundworkEvent.TextMessage, groundworkEvent.LastInsertDate.String(), groundworkEvent.ReportDate.String(),
		groundworkEvent.AppType, groundworkEvent.MonitorServer, groundworkEvent.ConsolidationName,
		groundworkEvent.LogType, groundworkEvent.ErrorType, groundworkEvent.LoggerName, groundworkEvent.ApplicationName,
	)
}

// MonitorConnection describes the connection to the monitored system
type MonitorConnection struct {
	ID          int         `json:"id"`
	Server      string      `json:"server"`
	UserName    string      `json:"userName"`
	Password    string      `json:"password"`
	SslEnabled  bool        `json:"sslEnabled"`
	URL         string      `json:"url"`
	Views       []View      `json:"views,omitempty"`
	Extensions  interface{} `json:"extensions"`
	ConnectorID int         `json:"connectorId"`
}

type View struct {
	Name        string                 `json:"name"`
	DisplayName string                 `json:"displayName"`
	Enabled     bool                   `json:"enabled"`
	Extensions  map[string]interface{} `json:"extensions,omitempty"`
}

func (monitorConnection MonitorConnection) String() string {
	return fmt.Sprintf("[%d, %s, %s, %s, %t, %s, %s, %d]",
		monitorConnection.ID, monitorConnection.Server, monitorConnection.UserName, monitorConnection.Password,
		monitorConnection.SslEnabled, monitorConnection.URL, monitorConnection.Extensions, monitorConnection.ConnectorID,
	)
}

// GroundworkEventsAckRequest describes request payload
type GroundworkEventsAckRequest struct {
	Acks []GroundworkEventAck `json:"acks"`
}

// GroundworkEventAck describes event ack
type GroundworkEventAck struct {
	AppType            string `json:"appType,required"`
	Host               string `json:"host,required"`
	Service            string `json:"service,omitempty"`
	AcknowledgedBy     string `json:"acknowledgedBy,omitempty"`
	AcknowledgeComment string `json:"acknowledgeComment,omitempty"`
}

func (groundworkEventAck GroundworkEventAck) String() string {
	return fmt.Sprintf("[%s, %s, %s, %s, %s]",
		groundworkEventAck.AppType, groundworkEventAck.Host, groundworkEventAck.Service,
		groundworkEventAck.AcknowledgedBy, groundworkEventAck.AcknowledgeComment,
	)
}

// GroundworkEventsUnackRequest describes request payload
type GroundworkEventsUnackRequest struct {
	Unacks []GroundworkEventUnack `json:"unacks"`
}

// GroundworkEventUnack describes event ack
type GroundworkEventUnack struct {
	AppType string `json:"appType,required"`
	Host    string `json:"host,required"`
	Service string `json:"service,omitempty"`
}

func (groundworkEventUnack GroundworkEventUnack) String() string {
	return fmt.Sprintf("[%s, %s, %s]",
		groundworkEventUnack.AppType, groundworkEventUnack.Host, groundworkEventUnack.Service)
}

type MetricsProfile struct {
	Name        string             `json:"name"`
	ProfileType string             `json:"profileType"`
	IsTemplate  bool               `json:"isTemplate"`
	Metrics     []MetricDefinition `json:"metrics"`
}

func (metricsProfile MetricsProfile) String() string {
	return fmt.Sprintf("[%s, %s, %t, %s]",
		metricsProfile.Name, metricsProfile.ProfileType,
		metricsProfile.IsTemplate, metricsProfile.Metrics,
	)
}

type MetricDefinition struct {
	Name              string      `json:"name"`
	CustomName        string      `json:"customName,omitempty"`
	Description       string      `json:"description,omitempty"`
	Monitored         bool        `json:"monitored,omitempty"`
	Graphed           bool        `json:"graphed,omitempty"`
	MetricType        MetricKind  `json:"metricType"`
	ComputeType       ComputeType `json:"computeType"`
	ServiceType       string      `json:"serviceType"`
	SourceType        string      `json:"sourceType,omitempty"`
	AggregateType     string      `json:"aggregateType,omitempty"`
	WarningThreshold  int         `json:"warningThreshold"`
	CriticalThreshold int         `json:"criticalThreshold"`
	Expression        string      `json:"expression,omitempty"`
	Format            string      `json:"format,omitempty"`
}

func (metricDefinition MetricDefinition) String() string {
	return fmt.Sprintf("[%s, %s, %s, %t, %t, %s, %s, %s, %s, %s, %d, %d, %s, %s]",
		metricDefinition.Name, metricDefinition.CustomName, metricDefinition.Description, metricDefinition.Monitored,
		metricDefinition.Graphed, metricDefinition.MetricType, metricDefinition.ComputeType, metricDefinition.ServiceType,
		metricDefinition.SourceType, metricDefinition.AggregateType, metricDefinition.WarningThreshold,
		metricDefinition.CriticalThreshold, metricDefinition.Expression, metricDefinition.Format,
	)
}

// HostServiceInDowntime describes downtime schedule
type HostServiceInDowntime struct {
	HostName               string `json:"hostName"`
	ServiceDescription     string `json:"serviceDescription,omitempty"`
	ScheduledDowntimeDepth int    `json:"scheduledDowntimeDepth"`
	EntityType             string `json:"entityType"`
	EntityName             string `json:"entityName"`
}

// HostServicesInDowntime defines type used for ClearInDowntime API payload
type HostServicesInDowntime struct {
	BizHostServiceInDowntimes []HostServiceInDowntime `json:"bizHostServiceInDowntimes"`
}

// HostsAndServices defines type used for SetInDowntime API payload
type HostsAndServices struct {
	HostNames                 []string `json:"hostNames"`
	ServiceDescriptions       []string `json:"serviceDescriptions"`
	HostGroupNames            []string `json:"hostGroupNames"`
	ServiceGroupCategoryNames []string `json:"serviceGroupCategoryNames"`
	SetHosts                  bool     `json:"setHosts"`
	SetServices               bool     `json:"setServices"`
}

func (mr *DynamicMonitoredResource) ToMonitoredResourceRef() MonitoredResourceRef {
	return MonitoredResourceRef{Name: mr.Name, Type: Host, Owner: mr.Owner}
}

func (mr *DynamicMonitoredResource) ToInventoryResource() DynamicInventoryResource {
	var services []DynamicInventoryService
	for _, ms := range mr.Services {
		services = append(services, ms.ToInventoryService())
	}
	return DynamicInventoryResource{
		BaseResource: mr.BaseResource,
		Services:     services,
	}
}

func (ms *DynamicMonitoredService) ToInventoryService() DynamicInventoryService {
	return DynamicInventoryService{
		BaseTransitData: ms.BaseTransitData,
	}
}

// AgentIdentity defines TCG Agent Identity
type AgentIdentity struct {
	AgentID string `json:"agentID" yaml:"agentID"`
	AppName string `json:"appName" yaml:"appName"`
	AppType string `json:"appType" yaml:"appType"`
}
