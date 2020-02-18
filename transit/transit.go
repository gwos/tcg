package transit

import (
	"fmt"
	"github.com/gwos/tng/milliseconds"
)

type VersionString string

const (
	TransitModelVersion VersionString = "1.0.0"
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
	Service ServiceType = "SERVICE"
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

// TypedValue defines a single strongly-typed value.
type TypedValue struct {
	ValueType ValueType `json:"valueType"`

	// BoolValue: A Boolean value: true or false.
	BoolValue bool `json:"boolValue,omitempty"`

	// DoubleValue: A 64-bit double-precision floating-point number. Its
	// magnitude is approximately &plusmn;10<sup>&plusmn;300</sup> and it
	// has 16 significant digits of precision.
	DoubleValue float64 `json:"doubleValue,omitempty"`

	// Int64Value: A 64-bit integer. Its range is approximately
	// &plusmn;9.2x10<sup>18</sup>.
	IntegerValue int64 `json:"integerValue,omitempty"`

	// StringValue: A variable-length string value.
	StringValue string `json:"stringValue,omitempty"`

	// a time stored as full timestamp
	TimeValue *milliseconds.MillisecondTimestamp `json:"timeValue,omitempty"`
}

type ThresholdValue struct {
	SampleType MetricSampleType `json:"sampleType"`
	Label      string           `json:"label"`
	Value      *TypedValue      `json:"value"`
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
	Interval   *TimeInterval     `json:"interval"`
	Value      *TypedValue       `json:"value"`
	Tags       map[string]string `json:"tags,omitempty"`
	Unit       UnitType          `json:"unit,omitempty"`
	Thresholds *[]ThresholdValue `json:"thresholds,omitempty"`
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

func (md MetricDescriptor) String() string {
	return fmt.Sprintf("%s - %s", md.Type, md.CustomName)
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

// ThresholdDescriptor defines a Threshold
type ThresholdDescriptor struct {
	// Key: The threshold key.
	Key   string `json:"key"`
	Value int32  `json:"value"`
}

// InventoryResource represents a resource that is included in a inventory scan.
// Examples include:
//  * nagios host
//  * virtual machine instance
//  * RDS database
//  * storage devices such as disks
//  * cloud resources such as cloud apps, cloud functions(lambdas)
//
// An InventoryResource is the representation of a specific monitored resource during an nventory scan.
// Each InventoryResource contains list of services (InventoryService) (no metrics are sent).
type InventoryResource struct {
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
	// Device (usually IP address), leave empty if not available, will default to name
	Device string `json:"device,omitempty"`
	// Foundation Properties
	Properties map[string]TypedValue `json:"properties,omitempty"`
	// Inventory Service collection
	Services []InventoryService `json:"services"`
}

// InventoryService represents a Groundwork Service that is included in a inventory scan.
// In cloud systems, services are usually modeled as a complex metric definition, with each sampled
// metric variation represented as as single metric time series. During inventory scans, TNG does not gather metric samples.
//
// InventoryService collections are attached to an InventoryResource during inventory scans.
type InventoryService struct {
	// The unique name of the service
	Name string `json:"name,required"`
	// Type: Required. The service type
	Type ServiceType `json:"type"`
	// Owner relationship for associations like host->service
	Owner string `json:"owner,omitempty"`
	// CloudHub Categorization of services
	Category string `json:"category,omitempty"`
	// Optional description of this service
	Description string `json:"description,omitempty"`
	// Foundation Properties
	Properties map[string]TypedValue `json:"properties,omitempty"`
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
	// The unique name of the resource
	Name string `json:"name,required"`
	// Type: Required. The resource type
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
	// Services state collection
	Services []MonitoredService `json:"services"`
}

// A MonitoredService represents a Groundwork Service creating during a metrics scan.
// In cloud systems, services are usually modeled as a complex metric definition, with each sampled
// metric variation represented as as single metric time series.
//
// A MonitoredService contains a collection of TimeSeries Metrics.
// MonitoredService collections are attached to a MonitoredResource during a metrics scan.
type MonitoredService struct {
	// The unique name of the service
	Name string `json:"name,required"`
	// Type: Required. The service type uniquely defining the service type
	// General Nagios Types are host and service, whereas CloudHub can have richer complexity
	Type ServiceType `json:"type,required"`
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
	// metrics
	Metrics []TimeSeries `json:"metrics"`
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

// TracerContext describes a Transit call
type TracerContext struct {
	AppType    string                            `json:"appType"`
	AgentID    string                            `json:"agentId"`
	TraceToken string                            `json:"traceToken"`
	TimeStamp  milliseconds.MillisecondTimestamp `json:"timeStamp"`
	Version    VersionString                     `json:"version"`
}

// OperationResult defines API answer
type OperationResult struct {
	Entity   string `json:"entity"`
	Status   string `json:"status"`
	Message  string `json:"message"`
	Location string `json:"location"`
	EntityID int    `json:"entityID"`
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

// ResourceGroup defines group entity
type ResourceGroup struct {
	GroupName   string                 `json:"groupName,required"`
	Type        GroupType              `json:"type,required"`
	Description string                 `json:"description,omitempty"`
	Resources   []MonitoredResourceRef `json:"resources,required"`
}

// ResourcesWithServicesRequest defines SendResourcesWithMetrics payload
type ResourcesWithServicesRequest struct {
	Context   *TracerContext      `json:"context,omitempty"`
	Resources []MonitoredResource `json:"resources"`
}

// InventoryRequest defines SynchronizeInventory payload
type InventoryRequest struct {
	Context   *TracerContext      `json:"context,omitempty"`
	Resources []InventoryResource `json:"resources"`
	Groups    []ResourceGroup     `json:"groups,omitempty"`
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

type GroundworkEventsRequest struct {
	Events				[]GroundworkEvent  				`json:"events"`
}

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
	MonitorServer     string                              `json:"monitorServer,omitempty"`
	ConsolidationName string                              `json:"consolidationName,omitempty"`
	LogType           string                              `json:"logType,omitempty"`
	ErrorType         string                              `json:"errorType,omitempty"`
	LoggerName        string                              `json:"loggerName,omitempty"`
	ApplicationName   string                              `json:"applicationName,omitempty"`
}

type MonitorConnection struct {
	Id          int                    `json:"id"`
	Server      string                 `json:"server"`
	UserName    string                 `json:"userName"`
	Password    string                 `json:"password"`
	SslEnabled  bool                   `json:"sslEnabled"`
	Url         string                 `json:"url"`
	Extensions  map[string]interface{} `json:"extensions"`
	ConnectorId int                    `json:"connectorId"`
}

type GroundworkEventsAckRequest struct {
	Acks			[]GroundworkEventAck					`json:"acks"`
}

type GroundworkEventAck struct {
	AppType            string `json:"appType,required"`
	Host               string `json:"host,required"`
	Service            string `json:"service,omitempty"`
	AcknowledgedBy     string `json:"acknowledgedBy,omitempty"`
	AcknowledgeComment string `json:"acknowledgeComment,omitempty"`
}
