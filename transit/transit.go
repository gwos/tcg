package transit

import "C"
import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
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
	TimeType                      = "TimeType"
	UnspecifiedType               = "UnspecifiedType"
)

// Supported units are a subset of The Unified Code for Units of Measure
// (http://unitsofmeasure.org/ucum.html) standard, added as we encounter
// the need for them in monitoring contexts.
type UnitEnum string

const (
	UnitCounter = "1"
	PercentCPU  = "%{cpu}"
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
	SERVICE_WARNING                                = "SERVICE_WARNING"
	SERVICE_UNSCHEDULED_CRITICAL                   = "SERVICE_UNSCHEDULED_CRITICAL"
	SERVICE_PENDING                                = "SERVICE_PENDING"
	SERVICE_SCHEDULED_CRITICAL                     = "SERVICE_SCHEDULED_CRITICAL"
	SERVICE_UNKNOWN                                = "SERVICE_UNKNOWN"
	HOST_UP                                        = "HOST_UP"
	HOST_UNSCHEDULED_DOWN                          = "HOST_UNSCHEDULED_DOWN"
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
	EndTime MillisecondTimestamp `json:"endTime,omitempty"`

	// StartTime: Optional. The beginning of the time interval. The default
	// value for the start time is the end time. The start time must not be
	// later than the end time.
	StartTime MillisecondTimestamp `json:"startTime,omitempty"`
}

// TypedValue defines a single strongly-typed value.
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

	// a time stored as full timestamp
	TimeValue MillisecondTimestamp `json:"timeValue,omitempty"`
}

// MetricSample defines a single data sample in a time series, which may represent
// either a measurement at a single point in time or data aggregated over a
// time duration.
type MetricSample struct {
	// SampleType: The kind of value this particular metric represents.
	SampleType MetricSampleType `json:"sampleType"`

	// Interval: The time interval to which the data sample applies. For
	// GAUGE metrics, only the end time of the interval is used. For DELTA
	// metrics, the start and end time should specify a non-zero interval,
	// with subsequent samples specifying contiguous and non-overlapping
	// intervals. For CUMULATIVE metrics, the start and end time should
	// specify a non-zero interval, with subsequent samples specifying the
	// same start time and increasing end times, until an event resets the
	// cumulative value to zero and sets a new start time for the following
	// samples.
	Interval *TimeInterval `json:"interval,omitempty"`

	// Value: The value of the metric sample.
	Value *TypedValue `json:"value,omitempty"`
}

// TimeSeries defines a collection of data samples that describes the
// time-varying values of a metric.
type TimeSeries struct {
	MetricName    string            `json:"metricName"`
	MetricSamples []*MetricSample   `json:"metricSamples"`
	Tags          map[string]string `json:"tags,omitempty"`
	Unit          UnitEnum          `json:"unit,omitempty"`
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
	// zero and sets a new start time for the following samples.
	MetricKind MetricKindEnum `json:"metricKind"`
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
	ValueType ValueTypeEnum `json:"valueType,omitempty"`
}

// ThresholdDescriptor defines a Threshold
type ThresholdDescriptor struct {
	// Key: The threshold key.
	Key   string `json:"key"`
	Value int32  `json:"value"`
}
// InventoryResource is an object representing a live resource instance that
// can be included in a monitoring inventory. Examples include for example:
// 	* virtual machine instances
// 	* databases
// 	* storage devices such as disks
// 	* webapps, serverless functions(lambdas)
// 	* real hosts and services on those hosts
type InventoryResource struct {
	// The unique name of the resource
	Name string `json:"name,required"`
	// Type: Required. The resource type uniquely defining the resource type
	// General Nagios Types are host and service, where as CloudHub can be more rich
	Type string `json:"type,required"`
	//  Owner relationship for associations like host->service
	Owner string `json:"owner,omitempty"`
	// CloudHub Categorization of resources, translate to Foundation Metric Type
	Category string `json:"category,omitempty"`
	// Optional description of this resource, such as Nagios notes
	Description string `json:"description,omitempty"`
	// Device (usually IP address), leave empty if not available, will default to name
	Device string `json:"device,omitempty"`
	// Foundation Properties
	Properties map[string]TypedValue `json:"properties,omitempty"`
}

// The current status of a monitored resource
type ResourceStatus struct {
	// The unique name of the resource
	Name string `json:"name,required"`
	// Type: Required. The resource type uniquely defining the resource type
	// General Nagios Types are host and service, where as CloudHub can be more rich
	Type string `json:"type,required"`
	// Restrict to a Groundwork Monitor Status
	Status MonitorStatusEnum `json:"status,required"`
	// The last status check time on this resource
	LastCheckTime MillisecondTimestamp `json:"lastCheckTime,omitempty"`
	// The next status check time on this resource
	NextCheckTime MillisecondTimestamp `json:"nextCheckTime,omitempty"`
	// Nagios plugin output string
	LastPlugInOutput string `json:"lastPlugInOutput,omitempty"`
	// Foundation Properties
	Properties map[string]TypedValue `json:"properties,omitempty"`
}

type MonitoredResource struct {
	// The unique name of the resource
	Name string `json:"name,required"`
	// Type: Required. The resource type uniquely defining the resource type
	// General Nagios Types are host and service, where as CloudHub can be more rich
	Type string `json:"type,required"`
}

// TracerContext describes a Transit call
type TracerContext struct {
	AppType    string               `json:"appType"`
	AgentId    string               `json:"agentId"`
	TraceToken string               `json:"traceToken"`
	TimeStamp  MillisecondTimestamp `json:"timeStamp"`
}

	type SendInventoryRequest struct {
	// Context   *TracerContext       `json:"context"`
	Inventory *[]InventoryResource `json:"resources"`
	Groups    *[]ResourceGroup     `json:"groups"`
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

type ResourceGroup struct {
	GroupName string               `json:"groupName"`
	Resources []MonitoredResource  `json:"resources"`
}

type ResourceWithMetrics struct {
	Resource ResourceStatus `json:"resource"`
	Metrics  []TimeSeries      `json:"metrics"`
}

// ResourceWithMetricsRequest defines SendResourcesWithMetrics payload
type ResourceWithMetricsRequest struct {
	Context   TracerContext         `json:"context"`
	Resources []ResourceWithMetrics `json:"resources"`
}

// Services defines operations
// TODO: clarify args
type Services interface {
	SendResourcesWithMetrics(request []byte) (*OperationResults, error)
	ListMetrics() (*[]MetricDescriptor, error)
	SynchronizeInventory(request []byte) (*OperationResults, error)
}

// GroundworkAction defines configurable options for an action
type GroundworkAction struct {
	Entrypoint string `yaml:"entrypoint"`
}

// GroundworkActions configures Groundwork actions
type GroundworkActions struct {
	Connect                 GroundworkAction `yaml:"connect"`
	Disconnect              GroundworkAction `yaml:"disconnect"`
	SynchronizeInventory    GroundworkAction `yaml:"synchronizeInventory"`
	SendResourceWithMetrics GroundworkAction `yaml:"sendResourceWithMetrics"`
}

// GroundworkConfig defines Groundwork Connection configuration
type GroundworkConfig struct {
	Host     string `yaml:"host",envconfig:"GW_HOST"`
	Account  string `yaml:"account",envconfig:"GW_ACCOUNT"`
	Password string `yaml:"password",envconfig:"GW_PASSWORD"`
	Token    string
}

// AgentConfig defines TNG Transit Agent configuration
type AgentConfig struct {
	Port int  `yaml:"port",envconfig:"AGENT_PORT"`
	SSL  bool `yaml:"ssl",envconfig:"AGENT_SSL"`
}

var Config Transit

// Implementation of Services
type Transit struct {
	AgentConfig       `yaml:"agentConfig"`
	GroundworkConfig  `yaml:"groundworkConfig"`
	GroundworkActions `yaml:"groundworkActions"`
}

func (transit *Transit) Connect() error {
	formValues := map[string]string{
		"gwos-app-name": "gw8",
		"user":          transit.GroundworkConfig.Account,
		"password":      transit.GroundworkConfig.Password,
	}

	headers := map[string]string{
		"Accept":       "text/plain",
		"Content-Type": "application/x-www-form-urlencoded",
	}

	entrypoint := url.URL{
		Scheme: "http",
		Host:   Config.GroundworkConfig.Host,
		Path:   Config.GroundworkActions.Connect.Entrypoint,
	}
	statusCode, byteResponse, err := SendRequest(http.MethodPost, entrypoint.String(), headers, formValues, nil)
	if err != nil {
		return err
	}
	if statusCode == 200 {
		transit.GroundworkConfig.Token = string(byteResponse)
		return nil
	}

	return errors.New(string(byteResponse))
}

func (transit Transit) Disconnect() error {
	formValues := map[string]string{
		"gwos-app-name":  "gw8",
		"gwos-api-token": transit.GroundworkConfig.Token,
	}

	headers := map[string]string{
		"Accept":       "text/plain",
		"Content-Type": "application/x-www-form-urlencoded",
	}

	entrypoint := url.URL{
		Scheme: "http",
		Host:   Config.GroundworkConfig.Host,
		Path:   Config.GroundworkActions.Disconnect.Entrypoint,
	}
	statusCode, byteResponse, err := SendRequest(http.MethodPost, entrypoint.String(), headers, formValues, nil)
	if err != nil {
		return err
	}

	if statusCode == 200 {
		return nil
	}
	return errors.New(string(byteResponse))
}

func (transit Transit) SynchronizeInventory(inventory []byte) (*OperationResults, error) {
	headers := map[string]string{
		"Accept":         "application/json",
		"Content-Type":   "application/json",
		"GWOS-API-TOKEN": transit.GroundworkConfig.Token,
		"GWOS-APP-NAME":  "gw8",
	}

	entrypoint := url.URL{
		Scheme: "http",
		Host:   Config.GroundworkConfig.Host,
		Path:   Config.GroundworkActions.SynchronizeInventory.Entrypoint,
	}
	statusCode, byteResponse, err := SendRequest(http.MethodPost, entrypoint.String(), headers, nil, inventory)
	if err != nil {
		return nil, err
	}
	if statusCode == 401 {
		err = transit.Connect()
		if err != nil {
			return nil, err
		}
	}
	if statusCode != 200 {
		return nil, errors.New(string(byteResponse))
	}

	var operationResults OperationResults

	err = json.Unmarshal(byteResponse, &operationResults)
	if err != nil {
		return nil, err
	}

	return &operationResults, nil
}

func (transit Transit) SendResourcesWithMetrics(resources []byte) (*OperationResults, error) {
	headers := map[string]string{
		"Accept":         "application/json",
		"Content-Type":   "application/json",
		"GWOS-API-TOKEN": transit.GroundworkConfig.Token,
		"GWOS-APP-NAME":  "gw8",
	}

	entrypoint := url.URL{
		Scheme: "http",
		Host:   Config.GroundworkConfig.Host,
		Path:   Config.GroundworkActions.SendResourceWithMetrics.Entrypoint,
	}
	statusCode, byteResponse, err := SendRequest(http.MethodPost, entrypoint.String(), headers, nil, resources)
	if err != nil {
		return nil, err
	}
	if statusCode == 401 {
		err = transit.Connect()
		if err != nil {
			return nil, err
		}
	}
	if statusCode != 200 {
		return nil, errors.New(string(byteResponse))
	}

	var operationResults OperationResults

	err = json.Unmarshal(byteResponse, &operationResults)
	if err != nil {
		return nil, err
	}

	return &operationResults, nil
}

// TODO: implement
func (transit Transit) ListMetrics() (*[]MetricDescriptor, error) {
	// setup label descriptor samples
	cores := LabelDescriptor{
		Description: "Number of Cores",
		Key:         "cores",
		ValueType:   StringType,
	}
	sampleTime := LabelDescriptor{
		Description: "Sample Time",
		Key:         "sampleTime",
		ValueType:   IntegerType,
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
		ValueType:   DoubleType,
		Thresholds: []*ThresholdDescriptor{
			&ThresholdDescriptor{Key: "critical", Value: 200},
			&ThresholdDescriptor{Key: "warning", Value: 100},
		},
	}
	load5 := MetricDescriptor{
		Type:        "local_load_5",
		Description: "Local Load for 5 minute",
		DisplayName: "LocalLoad5",
		Labels:      []*LabelDescriptor{&cores, &sampleTime},
		MetricKind:  GAUGE,
		ComputeType: Query,
		CustomName:  "load-five-minutes",
		Unit:        UnitCounter,
		ValueType:   DoubleType,
		Thresholds: []*ThresholdDescriptor{
			&ThresholdDescriptor{Key: "critical", Value: 205},
			&ThresholdDescriptor{Key: "warning", Value: 105},
		},
	}
	load15 := MetricDescriptor{
		Type:        "local_load_15",
		Description: "Local Load for 15 minute",
		DisplayName: "LocalLoad15",
		Labels:      []*LabelDescriptor{&cores, &sampleTime},
		MetricKind:  GAUGE,
		ComputeType: Query,
		CustomName:  "load-fifteen-minutes",
		Unit:        UnitCounter,
		ValueType:   DoubleType,
		Thresholds: []*ThresholdDescriptor{
			&ThresholdDescriptor{Key: "critical", Value: 215},
			&ThresholdDescriptor{Key: "warning", Value: 115},
		},
	}
	arr := []MetricDescriptor{load1, load5, load15}
	return &arr, nil
}

var AgentStatistics AgentStats

type AgentStats struct {
	AgentId                string
	AppType                string
	BytesSent              int
	MetricsSent            int
	MessagesSent           int
	LastInventoryRun       MillisecondTimestamp
	LastMetricsRun         MillisecondTimestamp
	ExecutionTimeInventory time.Duration
	ExecutionTimeMetrics   time.Duration
	UpSince                MillisecondTimestamp
	LastError              string
}

// MillisecondTimestamp refers to the JSON representation of timestamps, for
// time-data interchange, as a single integer representing a modified version of
// whole milliseconds since the UNIX epoch (00:00:00 UTC on January 1, 1970).
// Individual languages (Go, C, Java) will typically implement this structure
// using a more-complex contruction in their repective contexts, containing even
// finer granularity for local data storage, typically at the nanosecond level.
//
// The "modified version" comment reflects the following simplification.
// Despite the already fine-grained representation as milliseconds, this data
// value takes no account of leap seconds; for all of our calculations, we
// simply pretend they don't exist.  Individual feeders will typically map a
// 00:00:60 value for a leap second, obtained as a string so the presence of the
// leap second is obvious, as 00:01:00, and the fact that 00:01:00 will occur
// again in the following second will be silently ignored.  This means that any
// monitoring which really wants to accurately reflect International Atomic Time
// (TAI), UT1, or similar time coordinates will be subject to some disruption.
// It also means that even in ordinary circumstances, any calculations of
// sub-second time differences might run into surprises, since the following
// timestamps could appear in temporal order:
//
//         actual time   relative reported time in milliseconds
//     A:  00:00:59.000  59000
//     B:  00:00:60.000  60000
//     C:  00:00:60.700  60700
//     D:  00:01:00.000  60000
//     E:  00:01:00.300  60300
//     F:  00:01:01.000  61000
//
// In such a situation, (D - C) and (E - C) would be negative numbers.
//
// In other situations, a feeder might obtain a timestamp from a system hardware
// clock which, say, counts local nanoseconds and has no notion of any leap
// seconds having been inserted into human-readable string-time representations.
// So there could be some amount of offset if such values are compared across
// such a boundary.
//
// Beyond that, there is always the issue of computer clocks not being directly
// tied to atomic clocks, using inexpensive non-temperature-compensated crystals
// for timekeeping.  Such hardware can easily drift dramatically off course, and
// the local timekeeping may or may not be subject to course correction using
// HTP, chrony, or similar software that periodically adjusts the system time
// to keep it synchronized with the Internet.  Also, there may be large jumps
// in either a positive or negative direction when a drifted clock is suddenly
// brought back into synchronization with the rest of the world.
//
// In addition, we ignore here all temporal effects of Special Relativity, not
// to mention further adjustments needed to account for General Relativity.
// This is not a theoretical joke; those who monitor GPS satellites should take
// note of the limitations of this data type, and use some other data type for
// time-critical data exchange and calculations.
//
// The point of all this being, fine resolution of clock values should never be
// taken too seriously unless one is sure that the clocks being compared are
// directly hitched together, and even then one must allow for quantum leaps
// into the future and time travel into the past.
//
// Finally, note that the Go zero-value of the internal implementation object
// we use in that language does not have a reasonable value when interpreted
// as milliseconds since the UNIX epoch.  For that reason, the general rule is
// that the JSON representation of a zero-value for any field of this type, no
// matter what the originating language, will be to simply omit it from the
// JSON string.  That fact must be taken into account when marshalling and
// unmarshalling data structures that contain such fields.
//
type MillisecondTimestamp struct {
	time.Time
}

// UnmarshalJSON implements json.Unmarshaler.
func (sd *MillisecondTimestamp) UnmarshalJSON(input []byte) error {
	strInput := string(input)

	i, err := strconv.ParseInt(strInput, 10, 64)
	if err != nil {
		return err
	}

	i *= int64(time.Millisecond)

	*sd = MillisecondTimestamp{time.Unix(0, i)}

	return nil
}

// MarshalJSON implements json.Marshaler.
func (sd MillisecondTimestamp) MarshalJSON() ([]byte, error) {
	return []byte(fmt.Sprintf("%d", sd.UnixNano()/int64(time.Millisecond))), nil
}
