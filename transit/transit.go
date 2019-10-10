package transit

import "C"
import (
	"fmt"
	"strconv"
	"time"
)

// MetricKind: The metric kind of the time series.
//   "METRIC_KIND_UNSPECIFIED" - Do not use this default value.
//   "GAUGE" - An instantaneous measurement of a value.
//   "DELTA" - The change in a value during a time interval.
//   "CUMULATIVE" - A value accumulated over a time interval. Cumulative
type MetricKindEnum string

const (
	GAUGE                   MetricKindEnum = "GAUGE"
	DELTA                                  = "DELTA"
	CUMULATIVE                             = "CUMULATIVE"
	METRIC_KIND_UNSPECIFIED                = "METRIC_KIND_UNSPECIFIED"
)

// ValueType defines the data type of the value of a metric
type ValueTypeEnum string

const (
	IntegerType     ValueTypeEnum = "IntegerType"
	DoubleType                    = "DoubleType"
	StringType                    = "StringType"
	BooleanType                   = "BooleanType"
	DateType                      = "DateType"
	UnspecifiedType               = "UnspecifiedType"
)

// Supported units are a subset of The Unified Code for Units of Measure
// (http://unitsofmeasure.org/ucum.html) standard:Basic units (UNIT)
type UnitEnum string

const (
	UnitCounter = "1"
)

// CloudHub Compute Types
type ComputeTypeEnum string

const (
	Query       ComputeTypeEnum = "Query"
	Regex                       = "Regex"
	Synthetic                   = "Synthetic"
	Info                        = "Info"
	Performance                 = "Performance"
	Health                      = "Health"
)

// MonitorStatusEnum represents Groundwork service monitor status
type MonitorStatusEnum string

const (
	SERVICE_OK                   MonitorStatusEnum = "SERVICE_OK"
	SERVICE_UNSCHEDULED_CRITICAL                   = "SERVICE_UNSCHEDULED_CRITICAL"
	SERVICE_WARNING                                = "SERVICE_WARNING"
	SERVICE_PENDING                                = "SERVICE_PENDING"
	SERVICE_SCHEDULED_CRITICAL                     = "SERVICE_SCHEDULED_CRITICAL"
	SERVICE_UNKNOWN                                = "SERVICE_UNKNOWN"
	HOST_UP                                        = "HOST_UP"
	HOST_UNSCHEDULED_DOWN                          = "HOST_UNSCHEDULED_DOWN"
	HOST_WARNING                                   = "HOST_WARNING"
	HOST_PENDING                                   = "HOST_PENDING"
	HOST_SCHEDULED_DOWN                            = "HOST_SCHEDULED_DOWN"
	HOST_UNREACHABLE                               = "HOST_UNREACHABLE"
)

// Groundwork Standard Monitored Resource Types
const (
	ServiceResource = "service"
	HostResource    = "host"
)

// TimeSeries Metric Sample Possible Types
type MetricSampleType string

const (
	Value    MetricSampleType = "Value"
	Critical                  = "Critical"
	Warning                   = "Warning"
	Min                       = "Min"
	Max                       = "Max"
)

// TimeInterval: A closed time interval. It extends from the start time
// to the end time, and includes both: [startTime, endTime]. Valid time
// intervals depend on the MetricKind of the metric value. In no case
// can the end time be earlier than the start time.
// For a GAUGE metric, the startTime value is technically optional; if
// no value is specified, the start time defaults to the value of the
// end time, and the interval represents a single point in time. Such an
//  interval is valid only for GAUGE metrics, which are point-in-time
// measurements.
// For DELTA and CUMULATIVE metrics, the start time must be earlier
// than the end time.
// In all cases, the start time of the next interval must be  at least a
// microsecond after the end time of the previous interval.  Because the
// interval is closed, if the start time of a new interval  is the same
// as the end time of the previous interval, data written  at the new
// start time could overwrite data written at the previous  end time.
type TimeInterval struct {
	// EndTime: Required. The end of the time interval.
	EndTime SpecialDate `json:"endTime,omitempty"`

	// StartTime: Optional. The beginning of the time interval. The default
	// value for the start time is the end time. The start time must not be
	// later than the end time.
	StartTime SpecialDate `json:"startTime,omitempty"`
}

// TypedValue: A single strongly-typed value.
type TypedValue struct {
	ValueType ValueTypeEnum `json:"valueType"`

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

	// a date stored as full timestamp
	DateValue SpecialDate `json:"dateValue,omitempty"`
}

// Point: A single data point in a time series.
type Point struct {
	// Interval: The time interval to which the data point applies. For
	// GAUGE metrics, only the end time of the interval is used. For DELTA
	// metrics, the start and end time should specify a non-zero interval,
	// with subsequent points specifying contiguous and non-overlapping
	// intervals. For CUMULATIVE metrics, the start and end time should
	// specify a non-zero interval, with subsequent points specifying the
	// same start time and increasing end times, until an event resets the
	// cumulative value to zero and sets a new start time for the following
	// points.
	Interval *TimeInterval `json:"interval,omitempty"`

	// Value: The value of the data point.
	Value *TypedValue `json:"value,omitempty"`
}

// TimeSeries: A collection of data points that describes the
// time-varying values of a metric.
type TimeSeries struct {
	MetricName string            `json:"metricName"`
	SampleType MetricSampleType  `json:"sampleType"`
	Tags       map[string]string `json:"tags,omitempty"`
	Interval   *TimeInterval     `json:"interval,omitempty"`
	Value      *TypedValue       `json:"value,omitempty"`
	Unit       string            `json:"unit,omitempty"`
}

// MetricDescriptor: Defines a metric type and its schema
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
	// (http://unitsofmeasure.org/ucum.html) standard:Basic units (UNIT)
	Unit UnitEnum `json:"unit,omitempty"`

	// ValueType: Whether the measurement is an integer, a floating-point
	// number, etc. Some combinations of metric_kind and value_type might
	// not be supported.
	ValueType ValueTypeEnum `json:"valueType,omitempty"`

	// Groundwork Compute Type such as Synthetic
	ComputeType ComputeTypeEnum `json:"computeType,omitempty"`

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
	// zero and sets a new start time for the following points.
	MetricKind MetricKindEnum `json:"metricKind"`
}

func (md MetricDescriptor) String() string {
	return fmt.Sprintf("%s - %s", md.Type, md.CustomName)
}

// LabelDescriptor: A description of a label.
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
	ValueType ValueTypeEnum `json:"valueType,omitempty"`
}

// A definition of a Threshold
type ThresholdDescriptor struct {
	// Key: The threshold key.
	Key   string `json:"key"`
	Value int32  `json:"value"`
}

// MonitoredResource: An object representing a live resource instance that
// can be used  for monitoring. Examples include for example:
// * virtual machine instances
// * databases
// * storage devices such as disks
// * webapps, serverless functions(lambdas)
// * real hosts and services on those hosts
// The type field identifies a MonitoredResourceDescriptor object
// that describes the resource's schema. Information in the labels field
// identifies the actual resource and its attributes according to the
// schema. For example, a particular Compute Engine VM instance could be
// represented by the following object, because the
// MonitoredResourceDescriptor for "gce_instance" has labels
// "instance_id" and "zone":
// { "type": "gce_instance",
//   "labels": { "instance_id": "12345678901234",
//               "zone": "us-central1-a" }}
//
type MonitoredResource struct {
	// The unique name of the resource
	Name string `json:"name,required"`
	// Type: Required. The monitored resource type uniquely defining the resource type
	// General Nagios Types are host and service, where as CloudHub
	Type string `json:"type,required"`
	// Restrict to a Groundwork Monitor Status
	Status MonitorStatusEnum `json:"status,required"`
	//  Owner relationship for associations like host->service
	Owner string `json:"owner,omitempty"`
	// The last status check time on this resource
	LastCheckTime SpecialDate `json:"lastCheckTime,omitempty"`
	// The last status check time on this resource
	NextCheckTime SpecialDate `json:"nextCheckTime,omitempty"`
	// Nagios plugin output string
	LastPlugInOutput string `json:"lastPlugInOutput,omitempty"`
	// CloudHub Categorization of resources, translate to Foundation Metric Type
	Category string `json:"category,omitempty"`
	// CloudHub Categorization of resources, translate to Foundation Metric Type
	Description string `json:"description,omitempty"`
	// General Foundation Properties
	Properties map[string]TypedValue `json:"properties,omitempty"`
}

// Trace Context of a Transit call
type TracerContext struct {
	AppType    string      `json:"appType"`
	AgentId    string      `json:"agentId"`
	TraceToken string      `json:"traceToken"`
	TimeStamp  SpecialDate `json:"timeStamp"`
}

type SendInventoryRequest struct {
	Context   *TracerContext       `json:"context"`
	Inventory *[]MonitoredResource `json:"resources"`
	Groups    *[]Group             `json:"groups"`
}

type OperationResults struct {
	ResourcesAdded   int                `json:"successful"`
	ResourcesDeleted int                `json:"failed"`
	EntityType       string             `json:"entityType"`
	Operation        string             `json:"operation"`
	Warning          int                `json:"warning"`
	Count            int                `json:"count"`
	Results          *[]OperationResult `json:"results"`
}

type OperationResult struct {
	Entity   string `json:"entity"`
	Status   string `json:"status"`
	Message  string `json:"message"`
	Location string `json:"location"`
	EntityId int    `json:"entityId"`
}

type Group struct {
	GroupName string              `json:"groupName"`
	Resources []MonitoredResource `json:"resources"`
}

type ResourceWithMetrics struct {
	Resource MonitoredResource `json:"resource"`
	Metrics  []TimeSeries      `json:"metrics"`
}

type ResourceWithMetricsRequest struct {
	Context   TracerContext
	Resources []ResourceWithMetrics
}

// internal transit dta
type SendMetricsRequest struct {
	Trace   TracerContext          `json:"context"`
	Metrics *[]ResourceWithMetrics `json:"resources"`
}

// Transit interfaces / operations
type Services interface {
	sendResourcesWithMetrics(resources []byte) (*OperationResults, error)
	listMetrics() (*[]MetricDescriptor, error)
	synchronizeInventory(inventory []byte) (*OperationResults, error)
}

// Groundwork Connection Configuration
type GroundworkConfig struct {
	HostName string `yaml:"host",envconfig:"HOST"`
	Account  string `yaml:"account",envconfig:"ACCOUNT"`
	Password string `yaml:"password",envconfig:"PASSWORD"`
	Token    string
	SSL      bool `yaml:"ssl",envconfig:"SSL"`
}

// Implementation of Services
type Transit struct {
	Config GroundworkConfig `yaml:"config",envconfig:"CONFIG"`
}

type Actions struct {
	SynchronizeInventory    SynchronizeInventoryAction    `yaml:"synchronizeInventoryAction",envconfig:"SYNCHRONIZE_INVENTORY_ACTION"`
	SendResourceWithMetrics SendResourceWithMetricsAction `yaml:"sendResourceWithMetricsAction",envconfig:"SEND_RESOURCE_WITH_METRICS_ACTION"`
}

type SynchronizeInventoryAction struct {
	EntryPoint string `yaml:"entrypoint",envconfig:"ENTRYPOINT"`
}

type SendResourceWithMetricsAction struct {
	EntryPoint string `yaml:"entrypoint",envconfig:"ENTRYPOINT"`
}

type SpecialDate struct {
	time.Time
}

func (sd *SpecialDate) UnmarshalJSON(input []byte) error {
	strInput := string(input)

	i, err := strconv.ParseInt(strInput, 10, 64)
	if err != nil {
		return err
	}

	i *= int64(time.Millisecond)

	*sd = SpecialDate{time.Unix(0, i)}

	return nil
}

func (sd SpecialDate) MarshalJSON() ([]byte, error) {
	return []byte(fmt.Sprintf("%d", sd.UnixNano()/int64(time.Millisecond))), nil
}
