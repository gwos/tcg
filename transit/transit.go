package transit

import (
	"fmt"
	"time"
)

// MetricKind: The metric kind of the time series.
// If present, it must be the same as the metric kind
// of the associated metric. If the associated metric's descriptor must
// be auto-created, then this field specifies the metric kind of the new
// descriptor and must be either GAUGE (the default) or CUMULATIVE.
//
// Possible values:
//   "METRIC_KIND_UNSPECIFIED" - Do not use this default value.
//   "GAUGE" - An instantaneous measurement of a value.
//   "DELTA" - The change in a value during a time interval.
//   "CUMULATIVE" - A value accumulated over a time interval. Cumulative
// measurements in a time series should have the same start time and
// increasing end times, until an event resets the cumulative value to
// zero and sets a new start time for the following points.
type MetricKindEnum int

const (
	GAUGE MetricKindEnum = iota
	DELTA
	CUMULATIVE
	METRIC_KIND_UNSPECIFIED
)

func (metricKind MetricKindEnum) String() string {
	return [...]string{"GAUGE", "DELTA", "CUMULATIVE", "METRIC_KIND_UNSPECIFIED"}[metricKind]
}

// ValueType defines the data type of the value of a metric
type ValueTypeEnum int

const (
	BOOL ValueTypeEnum = iota
	INT8
	INT32
	INT64
	DOUBLE
	STRING
	VALUE_TYPE_UNSPECIFIED
)

func (valueType ValueTypeEnum) String() string {
	return [...]string{"BOOL", "INT8", "INT32", "INT64", "DOUBLE", "STRING", "VALUE_TYPE_UNSPECIFIED"}[valueType]
}

// Supported units are a subset of The Unified Code for Units of Measure
// (http://unitsofmeasure.org/ucum.html) standard:Basic units (UNIT)
type UnitEnum string
const (
	UnitCounter = "1"
)

// CloudHub Compute Types
type ComputeTypeEnum int
const (
	Query ComputeTypeEnum = iota
	Regex
	Synthetic
	Info
	Performance
	Health
)

func (computeType ComputeTypeEnum) String() string {
	return [...]string{"query", "regex", "synthetic", "info", "performance", "health"}[computeType]
}

// Groundwork service monitor status
type MonitorStatusEnum int

const (
	SERVICE_OK MonitorStatusEnum = iota
	SERVICE_UNSCHEDULED_CRITICAL
	SERVICE_WARNING
	SERVICE_PENDING
	SERVICE_SCHEDULED_CRITICAL
	SERVICE_UNKNOWN
	HOST_UP
	HOST_UNSCHEDULED_DOWN
	HOST_WARNING
	HOST_PENDING
	HOST_SCHEDULED_DOWN
	HOST_UNREACHABLE
)

func (status MonitorStatusEnum) String() string {
	return [...]string{
		"OK", "UNSCHEDULED CRITICAL", "WARNING", "PENDING", "SCHEDULED CRITICAL", "UNKNOWN",
		"UP", "UNSCHEDULED DOWN", "WARNING", "PENDING", "SCHEDULED DOWN", "UNREACHABLE",
	}[status]
}

// Groundwork Standard Monitored Resource Types
const (
	ServiceResource = "service"
	HostResource    = "host"
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
	EndTime time.Time `json:"endTime,omitempty"`

	// StartTime: Optional. The beginning of the time interval. The default
	// value for the start time is the end time. The start time must not be
	// later than the end time.
	StartTime time.Time `json:"startTime,omitempty"`
}

// TypedValue: A single strongly-typed value.
type TypedValue struct {
	// BoolValue: A Boolean value: true or false.
	BoolValue *bool `json:"boolValue,omitempty"`

	// DoubleValue: A 64-bit double-precision floating-point number. Its
	// magnitude is approximately &plusmn;10<sup>&plusmn;300</sup> and it
	// has 16 significant digits of precision.
	DoubleValue *float64 `json:"doubleValue,omitempty"`

	// Int8Value: A small integer
	Int8Value *int8 `json:"int8Value,omitempty,string"`
	Int32Value *int32 `json:"int32Value,omitempty,string"`

	// Int64Value: A 64-bit integer. Its range is approximately
	// &plusmn;9.2x10<sup>18</sup>.
	Int64Value *int64 `json:"int64Value,omitempty,string"`

	// StringValue: A variable-length string value.
	StringValue *string `json:"stringValue,omitempty"`
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
// time-varying values of a metric. A time series is identified by a
// combination of a fully-specified monitored resource and a
// fully-specified metric. This type is used for both listing and
// creating time series.
type TimeSeries struct {
	// Metric: The associated metric. A fully-specified metric used to
	// identify the time series.
	Metric *Metric `json:"metric,omitempty"`

	// Metric Kind defines how a metric is gathered, as a gauge, delta, or cumulative
	MetricKind MetricKindEnum `json:"metricKind,omitempty"`

	// Points: The data points of this time series. When listing time
	// series, points are returned in reverse time order.When creating a
	// time series, this field must contain exactly one point and the
	// point's type must be the same as the value type of the associated
	// metric. If the associated metric's descriptor must be auto-created,
	// then the value type of the descriptor is determined by the point's
	// type, which must be BOOL, INT64, DOUBLE, or DISTRIBUTION.
	Points []*Point `json:"points,omitempty"`

	// Resource: The associated monitored resource. Custom metrics can use
	// only certain monitored resource types in their time series data.
	Resource *MonitoredResource `json:"resource,omitempty"`

	// ValueType: The value type of the time series. When listing time
	// series, this value type might be different from the value type of the
	// associated metric if this time series is an alignment or reduction of
	// other time series.When creating a time series, this field is
	// optional. If present, it must be the same as the type of the data in
	// the points field.
	//
	// Possible values:
	//   "VALUE_TYPE_UNSPECIFIED" - Do not use this default value.
	//   "BOOL" - The value is a boolean. This value type can be used only
	// if the metric kind is GAUGE.
	//   "INT64" - The value is a signed 64-bit integer.
	//   "DOUBLE" - The value is a double precision floating point number.
	//   "STRING" - The value is a text string. This value type can be used
	// only if the metric kind is GAUGE.
	//   "DISTRIBUTION" - The value is a Distribution.
	//   "MONEY" - The value is money.
	ValueType ValueTypeEnum `json:"valueType,omitempty"`
}

// Metric: A specific metric, identified by specifying values for all of
// the labels of a MetricDescriptor.
type Metric struct {
	// Type: An existing metric type, MetricDescriptor. For
	// example, custom.googleapis.com/invoice/paid/amount.
	Type string `json:"type"`

	// Labels: The set of label values that uniquely identify this metric.
	// All labels listed in the MetricDescriptor must be assigned values.
	Labels map[string]TypedValue `json:"labels,omitempty"`
}

// MetricDescriptor: Defines a metric type and its schema
type MetricDescriptor struct {

	// Groundwork Compute Type such as Synthetic
	ComputeType ComputeTypeEnum `json:"name,omitempty"`

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
	MetricKind MetricKindEnum `json:"metricKind,omitempty"`

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
	Key string `json:"key"`
	Value int32 `json:"value"`
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
	// The unique name of the instance
	Name string `json:name,required`

	// Type: Required. The monitored resource type. This field must match
	// the type field of a MonitoredResourceDescriptor object. For example,
	// the type of a Compute Engine VM instance is gce_instance.
	Type string `json:"type"`

	// Groundwork Status
	Status MonitorStatusEnum `json:"status"`

	// Owner relationship for associations like host->service
	Owner *MonitoredResource `json:"owner,omitempty"`

	// Labels: Values for all of the labels listed in the
	// associated monitored resource descriptor. For example, Compute Engine
	// VM instances use the labels "project_id", "instance_id", and "zone".
	Labels map[string]string `json:"labels,omitempty"`

}

// MonitoredResourceDescriptor: An object that describes the schema of a
// MonitoredResource object using a type name and a set of labels. For
// example, the monitored resource descriptor for Google Compute Engine
// VM instances has a type of "gce_instance" and specifies the use of
// the labels "instance_id" and "zone" to identify particular VM
// instances.Different APIs can support different monitored resource
// types. APIs generally provide a list method that returns the
// monitored resource descriptors used by the API.
type MonitoredResourceDescriptor struct {
	// Type: Required. The monitored resource type. For example, the type
	// "cloudsql_database" represents databases in Google Cloud SQL. The
	// maximum length of this value is 256 characters.
	Type string `json:"type,omitempty"`

	// Description: Optional. A detailed description of the monitored
	// resource type that might be used in documentation.
	Description string `json:"description,omitempty"`

	// DisplayName: Optional. A concise name for the monitored resource type
	// that might be displayed in user interfaces. It should be a Title
	// Cased Noun Phrase, without any article or other determiners. For
	// example, "Google Cloud SQL Database".
	DisplayName string `json:"displayName,omitempty"`

	// Labels: A set of labels used to describe instances of this
	// monitored resource type. For example, an individual Google Cloud SQL
	// database is identified by values for the labels "database_id" and
	// "zone".
	Labels []*LabelDescriptor `json:"labels,omitempty"`
}

// Trace Context of a Transit call
type TracerContext struct {
	appType    string
	agentId    string
	traceToken string
	timeStamp  time.Time
}

type TransitSendInventoryRequest struct {
	context   *TracerContext
	inventory *[]MonitoredResource
	groups    *[]Group
}

type TransitSynchronizeResponse struct {
	ResourcesAdded   int
	ResourcesDeleted int
}

type Group struct {
	groupName string
	resources []MonitoredResource
}

// Transit interfaces / operations
type TransitServices interface {
	SendMetrics(metrics *[]TimeSeries) (string, error)
	ListMetrics() (*[]MetricDescriptor, error)
	SynchronizeInventory(inventory *[]MonitoredResource, groups *[]Group) (TransitSynchronizeResponse, error)
	// listInventory() (/*TODO*/ error)
}

// Groundwork Connection Configuration
type GroundworkConfig struct {
	HostName string
	Account  string
	Token    string
	SSL      bool
}

// Implementation of TransitServices
type Transit struct {
	config GroundworkConfig
}

// create and connect to a Transit instance from a Groundwork connection configuration
func Connect(config GroundworkConfig) Transit {
	transit := Transit{config: config}
	return transit;
}

func Disconnect(transit *Transit) {}

// TODO: implement
func (transit Transit) SendMetrics(metrics *[]TimeSeries) (string, error) {
	for _, ts := range *metrics {
		fmt.Printf("metric: %s, resourceType: %s, host:service: %s:%s\n",
			ts.Metric.Type,
			ts.Resource.Type,
			ts.Resource.Labels["host"],
			ts.Resource.Labels["name"])
		for _, point := range ts.Points {
			fmt.Printf("\t%f - %s\n", *point.Value.DoubleValue, point.Interval.EndTime.Format(time.RFC3339Nano))
		}
		fmt.Println()
	}
	return "success", nil
}

// TODO: implement
func (transit Transit) ListMetrics() (*[]MetricDescriptor, error) {
	// setup label descriptor samples
	cores := LabelDescriptor{
		Description: "Number of Cores",
		Key:         "cores",
		ValueType:   STRING,
	};
	sampleTime := LabelDescriptor{
		Description: "Sample Time",
		Key:         "sampleTime",
		ValueType:   INT64,
	}
	load1 := MetricDescriptor{
		Type:        "local_load_1",
		Description: "Local Load for 1 minute",
		DisplayName: "LocalLoad1",
		Labels:      []*LabelDescriptor{&cores, &sampleTime},
		MetricKind:  GAUGE,
		ComputeType: Query,
		CustomName:  "load-one-minute",
		Unit:        UnitCounter,
		ValueType:   DOUBLE,
		Thresholds:  []*ThresholdDescriptor{
			&ThresholdDescriptor{ Key: "critical", Value: 200 },
			&ThresholdDescriptor{ Key: "warning", Value: 100 },
		},
	}
	load5 := MetricDescriptor{
		Type:        "local_load_5",
		Description: "Local Load for 5 minute",
		DisplayName: "LocalLoad5",
		Labels:      []*LabelDescriptor{&cores, &sampleTime},
		MetricKind:  GAUGE,
		CustomName:  "load-five-minutes",
		Unit:        UnitCounter,
		ValueType:   DOUBLE,
		Thresholds:  []*ThresholdDescriptor{
			&ThresholdDescriptor{ Key: "critical", Value: 205 },
			&ThresholdDescriptor{ Key: "warning", Value: 105 },
		},

	}
	load15 := MetricDescriptor{
		Type:        "local_load_15",
		Description: "Local Load for 15 minute",
		DisplayName: "LocalLoad15",
		Labels:      []*LabelDescriptor{&cores, &sampleTime},
		MetricKind:  GAUGE,
		CustomName:  "load-fifteen-minutes",
		Unit:        UnitCounter,
		ValueType:   DOUBLE,
		Thresholds:  []*ThresholdDescriptor{
			&ThresholdDescriptor{ Key: "critical", Value: 215 },
			&ThresholdDescriptor{ Key: "warning", Value: 115 },
		},
	}
	arr := []MetricDescriptor{load1, load5, load15};
	return &arr, nil
}

// internal transit data
type transitSendMetricsRequest struct {
	trace   TracerContext
	metrics []*TimeSeries
}

type transitSendInventoryRequest struct {
	trace     TracerContext
	inventory []*MonitoredResourceDescriptor
}
