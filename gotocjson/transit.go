package transit

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/gwos/tng/config"
	"github.com/gwos/tng/milliseconds"
	"net/http"
	"net/url"
	"strconv"
)

// MetricKind defines the metric kind of the time series.
type MetricKind string

// MetricKindUnspecified - Do not use this default value.
// Gauge - An instantaneous measurement of a value.
// Delta - The change in a value during a time interval.
// Cumulative - A value accumulated over a time interval. Cumulative
const (
	Gauge                 MetricKind = "GAUGE"
	Delta                            = "DELTA"
	Cumulative                       = "CUMULATIVE"
	MetricKindUnspecified            = "METRIC_KIND_UNSPECIFIED"
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
)

// ComputeType defines CloudHub Compute Types
type ComputeType string

// CloudHub Compute Types
const (
	Query       ComputeType = "Query"
	Regex                   = "Regex"
	Synthetic               = "Synthetic"
	Info                    = "Info"
	Performance             = "Performance"
	Health                  = "Health"
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
)

type MonitoredResourceType string

// The resource type uniquely defining the resource type
// General Nagios Types are host and service, where as CloudHub can be more rich
const (
	ServiceResource MonitoredResourceType = "service"
	HostResource                          = "host"
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
	TimeValue milliseconds.MillisecondTimestamp `json:"timeValue,omitempty"`
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
	Unit          UnitType          `json:"unit,omitempty"`
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
	// FIX MAJOR:  Should this be "Type MonitoredResourceType" instead?  Yes, probably.
	Type string `json:"type,required"`
	// Owner relationship for associations like host->service
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

// ResourceStatus defines the current status of a monitored resource
type ResourceStatus struct {
	// The unique name of the resource
	Name string `json:"name,required"`
	// Type: Required. The resource type uniquely defining the resource type
	// General Nagios Types are host and service, where as CloudHub can be more rich
	// FIX MAJOR:  Should this be "Type MonitoredResourceType" instead?  yes, probably.
	Type string `json:"type,required"`
	// Owner relationship for associations like host->service
	Owner string `json:"owner,omitempty"`
	// Restrict to a Groundwork Monitor Status
	Status MonitorStatus `json:"status,required"`
	// The last status check time on this resource
	LastCheckTime milliseconds.MillisecondTimestamp `json:"lastCheckTime,omitempty"`
	// The next status check time on this resource
	NextCheckTime milliseconds.MillisecondTimestamp `json:"nextCheckTime,omitempty"`
	// Nagios plugin output string
	LastPlugInOutput string `json:"lastPlugInOutput,omitempty"`
	// Foundation Properties
	Properties map[string]TypedValue `json:"properties,omitempty"`
}

// MonitoredResource defines the resource entity
type MonitoredResource struct {
	// The unique name of the resource
	Name string `json:"name,required"`
	// Type: Required. The resource type uniquely defining the resource type
	// General Nagios Types are host and service, where as CloudHub can be more rich
	Type MonitoredResourceType `json:"type,required"`
	// Owner relationship for associations like host->service
	Owner string `json:"owner,omitempty"`
}

// TracerContext describes a Transit call
type TracerContext struct {
	AppType    string                            `json:"appType"`
	AgentID    string                            `json:"agentID"`
	TraceToken string                            `json:"traceToken"`
	TimeStamp  milliseconds.MillisecondTimestamp `json:"timeStamp"`
}

// SendInventoryRequest defines SendInventory payload
type SendInventoryRequest struct {
	// Context   *TracerContext       `json:"context"`
	Inventory *[]InventoryResource `json:"resources"`
	Groups    *[]ResourceGroup     `json:"groups"`
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

// OperationResult defines API answer
type OperationResult struct {
	Entity   string `json:"entity"`
	Status   string `json:"status"`
	Message  string `json:"message"`
	Location string `json:"location"`
	EntityID int    `json:"entityID"`
}

// ResourceGroup defines group entity
type ResourceGroup struct {
	GroupName string              `json:"groupName"`
	Resources []MonitoredResource `json:"resources"`
}

// ResourceWithMetrics combines resource data
type ResourceWithMetrics struct {
	Resource ResourceStatus `json:"resource"`
	Metrics  []TimeSeries   `json:"metrics"`
}

// ResourceWithMetricsRequest defines SendResourcesWithMetrics payload
type ResourceWithMetricsRequest struct {
	Context   TracerContext         `json:"context"`
	Resources []ResourceWithMetrics `json:"resources"`
}

// Operations defines Groundwork operations interface
type Operations interface {
	Connect() error
	Disconnect() error
	ValidateToken(appName, apiToken string) error
	SendResourcesWithMetrics(request []byte) (*OperationResults, error)
	SynchronizeInventory(request []byte) (*OperationResults, error)
}

// Transit implements Operations interface
type Transit struct {
	*config.Config
}

// Connect implements Operations.Connect.
func (transit *Transit) Connect() error {
	formValues := map[string]string{
		"gwos-app-name": transit.GroundworkConfig.AppName,
		"user":          transit.GroundworkConfig.Account,
		"password":      transit.GroundworkConfig.Password,
	}

	headers := map[string]string{
		"Accept":       "text/plain",
		"Content-Type": "application/x-www-form-urlencoded",
	}

	entrypoint := url.URL{
		Scheme: "http",
		Host:   transit.GroundworkConfig.Host,
		Path:   transit.GroundworkActions.Connect.Entrypoint,
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

// Disconnect implements Operations.Disconnect.
func (transit Transit) Disconnect() error {
	formValues := map[string]string{
		"gwos-app-name":  transit.GroundworkConfig.AppName,
		"gwos-api-token": transit.GroundworkConfig.Token,
	}

	headers := map[string]string{
		"Accept":       "text/plain",
		"Content-Type": "application/x-www-form-urlencoded",
	}

	entrypoint := url.URL{
		Scheme: "http",
		Host:   transit.GroundworkConfig.Host,
		Path:   transit.GroundworkActions.Disconnect.Entrypoint,
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

// ValidateToken implements Operations.ValidateToken.
func (transit Transit) ValidateToken(appName, apiToken string) error {
	headers := map[string]string{
		"Accept":       "text/plain",
		"Content-Type": "application/x-www-form-urlencoded",
	}

	formValues := map[string]string{
		"gwos-app-name":  appName,
		"gwos-api-token": apiToken,
	}

	entrypoint := url.URL{
		Scheme: "http",
		Host:   transit.GroundworkConfig.Host,
		Path:   transit.GroundworkActions.ValidateToken.Entrypoint,
	}

	statusCode, byteResponse, err := SendRequest(http.MethodPost, entrypoint.String(), headers, formValues, nil)

	if err == nil {
		if statusCode == 200 {
			b, _ := strconv.ParseBool(string(byteResponse))
			if b {
				return nil
			} else {
				return errors.New("invalid gwos-app-name or gwos-api-token")
			}
		} else {
			return errors.New(string(byteResponse))
		}
	}

	return err
}

// SynchronizeInventory implements Operations.SynchronizeInventory.
func (transit *Transit) SynchronizeInventory(inventory []byte) (*OperationResults, error) {
	headers := map[string]string{
		"Accept":         "application/json",
		"Content-Type":   "application/json",
		"GWOS-API-TOKEN": transit.GroundworkConfig.Token,
		"GWOS-APP-NAME":  transit.GroundworkConfig.AppName,
	}

	entrypoint := url.URL{
		Scheme: "http",
		Host:   transit.GroundworkConfig.Host,
		Path:   transit.GroundworkActions.SynchronizeInventory.Entrypoint,
	}
	statusCode, byteResponse, err := SendRequest(http.MethodPost, entrypoint.String(), headers, nil, inventory)
	if statusCode == 401 {
		err = transit.Connect()
		if err != nil {
			return nil, err
		}
		headers["GWOS-API-TOKEN"] = transit.GroundworkConfig.Token
		statusCode, byteResponse, err = SendRequest(http.MethodPost, entrypoint.String(), headers, nil, inventory)
	}
	if err != nil {
		return nil, err
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

// SendResourcesWithMetrics implements Operations.SendResourcesWithMetrics.
func (transit *Transit) SendResourcesWithMetrics(resources []byte) (*OperationResults, error) {
	headers := map[string]string{
		"Accept":         "application/json",
		"Content-Type":   "application/json",
		"GWOS-API-TOKEN": transit.GroundworkConfig.Token,
		"GWOS-APP-NAME":  transit.GroundworkConfig.AppName,
	}

	entrypoint := url.URL{
		Scheme: "http",
		Host:   transit.GroundworkConfig.Host,
		Path:   transit.GroundworkActions.SendResourceWithMetrics.Entrypoint,
	}
	statusCode, byteResponse, err := SendRequest(http.MethodPost, entrypoint.String(), headers, nil, resources)
	if statusCode == 401 {
		err = transit.Connect()
		if err != nil {
			return nil, err
		}
		headers["GWOS-API-TOKEN"] = transit.GroundworkConfig.Token
		statusCode, byteResponse, err = SendRequest(http.MethodPost, entrypoint.String(), headers, nil, resources)
	}
	if err != nil {
		return nil, err
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

// ListMetrics TODO: implement
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
		MetricKind:  Gauge,
		ComputeType: Query,
		CustomName:  "load-one-minute",
		Unit:        UnitCounter,
		ValueType:   DoubleType,
		Thresholds: []*ThresholdDescriptor{
			{Key: "critical", Value: 200},
			{Key: "warning", Value: 100},
		},
	}
	load5 := MetricDescriptor{
		Type:        "local_load_5",
		Description: "Local Load for 5 minute",
		DisplayName: "LocalLoad5",
		Labels:      []*LabelDescriptor{&cores, &sampleTime},
		MetricKind:  Gauge,
		ComputeType: Query,
		CustomName:  "load-five-minutes",
		Unit:        UnitCounter,
		ValueType:   DoubleType,
		Thresholds: []*ThresholdDescriptor{
			{Key: "critical", Value: 205},
			{Key: "warning", Value: 105},
		},
	}
	load15 := MetricDescriptor{
		Type:        "local_load_15",
		Description: "Local Load for 15 minute",
		DisplayName: "LocalLoad15",
		Labels:      []*LabelDescriptor{&cores, &sampleTime},
		MetricKind:  Gauge,
		ComputeType: Query,
		CustomName:  "load-fifteen-minutes",
		Unit:        UnitCounter,
		ValueType:   DoubleType,
		Thresholds: []*ThresholdDescriptor{
			{Key: "critical", Value: 215},
			{Key: "warning", Value: 115},
		},
	}
	arr := []MetricDescriptor{load1, load5, load15}
	return &arr, nil
}
