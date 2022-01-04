package transit

import (
	"fmt"
	"strconv"
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
	Gauge                 MetricKind = "GAUGE"
	Delta                 MetricKind = "DELTA"
	Cumulative            MetricKind = "CUMULATIVE"
)

// ValueType defines the data type of the value of a metric
type ValueType string

// Data type of the value of a metric
const (
	IntegerType     ValueType = "IntegerType"
	DoubleType      ValueType = "DoubleType"
	StringType      ValueType = "StringType"
	BooleanType     ValueType = "BooleanType"
	TimeType        ValueType = "TimeType"
	UnspecifiedType ValueType = "UnspecifiedType"
)

// UnitType - Supported units are a subset of The Unified Code for Units of Measure
// (http://unitsofmeasure.org/ucum.html) standard, added as we encounter
// the need for them in monitoring contexts.
type UnitType string

// Supported units
const (
	UnitCounter UnitType = "1"
	PercentCPU  UnitType = "%{cpu}"
	KB          UnitType = "KB"
	MB          UnitType = "MB"
	GB          UnitType = "GB"
)

// ComputeType defines CloudHub Compute Types
type ComputeType string

// CloudHub Compute Types
const (
	Query         ComputeType = "Query"
	Regex         ComputeType = "Regex"
	Synthetic     ComputeType = "Synthetic"
	Informational ComputeType = "Informational"
	Performance   ComputeType = "Performance"
	Health        ComputeType = "Health"
)

// MonitorStatus represents Groundwork service monitor status
type MonitorStatus string

// Groundwork Standard Monitored Resource Statuses
const (
	ServiceOk                  MonitorStatus = "SERVICE_OK"
	ServiceWarning             MonitorStatus = "SERVICE_WARNING"
	ServiceUnscheduledCritical MonitorStatus = "SERVICE_UNSCHEDULED_CRITICAL"
	ServicePending             MonitorStatus = "SERVICE_PENDING"
	ServiceScheduledCritical   MonitorStatus = "SERVICE_SCHEDULED_CRITICAL"
	ServiceUnknown             MonitorStatus = "SERVICE_UNKNOWN"
	HostUp                     MonitorStatus = "HOST_UP"
	HostUnscheduledDown        MonitorStatus = "HOST_UNSCHEDULED_DOWN"
	HostWarning                MonitorStatus = "HOST_WARNING"
	HostPending                MonitorStatus = "HOST_PENDING"
	HostScheduledDown          MonitorStatus = "HOST_SCHEDULED_DOWN"
	HostUnreachable            MonitorStatus = "HOST_UNREACHABLE"
	HostUnchanged              MonitorStatus = "HOST_UNCHANGED"
)

// ResourceType defines the resource type
type ResourceType string

// The resource type uniquely defining the resource type
// General Nagios Types are host and service, whereas CloudHub can have richer complexity
const (
	ResourceTypeHost           ResourceType = "host"
	ResourceTypeService        ResourceType = "service"
	ResourceTypeHypervisor     ResourceType = "hypervisor"
	ResourceTypeInstance       ResourceType = "instance"
	ResourceTypeVirtualMachine ResourceType = "virtual-machine"
	ResourceTypeCloudApp       ResourceType = "cloud-app"
	ResourceTypeCloudFunction  ResourceType = "cloud-function"
	ResourceTypeLoadBalancer   ResourceType = "load-balancer"
	ResourceTypeContainer      ResourceType = "container"
	ResourceTypeStorage        ResourceType = "storage"
	ResourceTypeNetwork        ResourceType = "network"
	ResourceTypeNetworkSwitch  ResourceType = "network-switch"
	ResourceTypeNetworkDevice  ResourceType = "network-device"
)

// ServiceType defines the service type
type ServiceType string

// Possible Types
const (
	ServiceTypeProcess ServiceType = "Process"
	ServiceTypeService ServiceType = "Service"
)

// GroupType defines the foundation group type
type GroupType string

// The group type uniquely defining corresponding foundation group type
const (
	HostGroup    GroupType = "HostGroup"
	ServiceGroup GroupType = "ServiceGroup"
	CustomGroup  GroupType = "CustomGroup"
)

// MetricSampleType defines TimeSeries Metric Sample Possible Types
type MetricSampleType string

// TimeSeries Metric Sample Possible Types
const (
	Value    MetricSampleType = "Value"
	Warning  MetricSampleType = "Warning"
	Critical MetricSampleType = "Critical"
	Min      MetricSampleType = "Min"
	Max      MetricSampleType = "Max"
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
	EndTime *Timestamp `json:"endTime"`

	// StartTime: Optional. The beginning of the time interval. The default
	// value for the start time is the end time. The start time must not be
	// later than the end time.
	StartTime *Timestamp `json:"startTime,omitempty"`
}

// String implements Stringer interface
func (value TimeInterval) String() string {
	return fmt.Sprintf("[%s, %s]",
		value.EndTime,
		value.StartTime,
	)
}

// TypedValue defines a single strongly-typed value.
type TypedValue struct {
	ValueType ValueType `json:"valueType"`

	// BoolValue: A Boolean value: true or false.
	BoolValue *bool `json:"boolValue,omitempty"`

	// DoubleValue: A 64-bit double-precision floating-point number. Its
	// magnitude is approximately &plusmn;10<sup>&plusmn;300</sup> and it
	// has 16 significant digits of precision.
	DoubleValue *float64 `json:"doubleValue,omitempty"`

	// Int64Value: A 64-bit integer. Its range is approximately
	// &plusmn;9.2x10<sup>18</sup>.
	IntegerValue *int64 `json:"integerValue,omitempty"`

	// StringValue: A variable-length string value.
	StringValue *string `json:"stringValue,omitempty"`

	// a time stored as full timestamp
	TimeValue *Timestamp `json:"timeValue,omitempty"`
}

// String implements Stringer interface
func (value TypedValue) String() string {
	switch value.ValueType {
	case IntegerType:
		return strconv.FormatInt(*value.IntegerValue, 10)
	case StringType:
		return *value.StringValue
	case DoubleType:
		return fmt.Sprintf("%f", *value.DoubleValue)
	case BooleanType:
		return strconv.FormatBool(*value.BoolValue)
	case TimeType:
		return value.TimeValue.String()
	}
	return ""
}

// NewTypedValue returns a reference to TypedValue or nil
func NewTypedValue(v interface{}) *TypedValue {
	p := new(TypedValue)
	switch v := v.(type) {
	case bool:
		p.ValueType = BooleanType
		p.BoolValue = new(bool)
		*p.BoolValue = v
	case *bool:
		p.ValueType = BooleanType
		p.BoolValue = new(bool)
		*p.BoolValue = *v
	case float32:
		p.ValueType = DoubleType
		p.DoubleValue = new(float64)
		*p.DoubleValue = float64(v)
	case *float32:
		p.ValueType = DoubleType
		p.DoubleValue = new(float64)
		*p.DoubleValue = float64(*v)
	case float64:
		p.ValueType = DoubleType
		p.DoubleValue = new(float64)
		*p.DoubleValue = v
	case *float64:
		p.ValueType = DoubleType
		p.DoubleValue = new(float64)
		*p.DoubleValue = *v
	case int:
		p.ValueType = IntegerType
		p.IntegerValue = new(int64)
		*p.IntegerValue = int64(v)
	case *int:
		p.ValueType = IntegerType
		p.IntegerValue = new(int64)
		*p.IntegerValue = int64(*v)
	case int8:
		p.ValueType = IntegerType
		p.IntegerValue = new(int64)
		*p.IntegerValue = int64(v)
	case *int8:
		p.ValueType = IntegerType
		p.IntegerValue = new(int64)
		*p.IntegerValue = int64(*v)
	case int16:
		p.ValueType = IntegerType
		p.IntegerValue = new(int64)
		*p.IntegerValue = int64(v)
	case *int16:
		p.ValueType = IntegerType
		p.IntegerValue = new(int64)
		*p.IntegerValue = int64(*v)
	case int32:
		p.ValueType = IntegerType
		p.IntegerValue = new(int64)
		*p.IntegerValue = int64(v)
	case *int32:
		p.ValueType = IntegerType
		p.IntegerValue = new(int64)
		*p.IntegerValue = int64(*v)
	case int64:
		p.ValueType = IntegerType
		p.IntegerValue = new(int64)
		*p.IntegerValue = v
	case *int64:
		p.ValueType = IntegerType
		p.IntegerValue = new(int64)
		*p.IntegerValue = *v
	case uint:
		p.ValueType = IntegerType
		p.IntegerValue = new(int64)
		*p.IntegerValue = int64(v)
	case *uint:
		p.ValueType = IntegerType
		p.IntegerValue = new(int64)
		*p.IntegerValue = int64(*v)
	case uint8:
		p.ValueType = IntegerType
		p.IntegerValue = new(int64)
		*p.IntegerValue = int64(v)
	case *uint8:
		p.ValueType = IntegerType
		p.IntegerValue = new(int64)
		*p.IntegerValue = int64(*v)
	case uint16:
		p.ValueType = IntegerType
		p.IntegerValue = new(int64)
		*p.IntegerValue = int64(v)
	case *uint16:
		p.ValueType = IntegerType
		p.IntegerValue = new(int64)
		*p.IntegerValue = int64(*v)
	case uint32:
		p.ValueType = IntegerType
		p.IntegerValue = new(int64)
		*p.IntegerValue = int64(v)
	case *uint32:
		p.ValueType = IntegerType
		p.IntegerValue = new(int64)
		*p.IntegerValue = int64(*v)
	case uint64:
		p.ValueType = IntegerType
		p.IntegerValue = new(int64)
		*p.IntegerValue = int64(v)
	case *uint64:
		p.ValueType = IntegerType
		p.IntegerValue = new(int64)
		*p.IntegerValue = int64(*v)
	case string:
		p.ValueType = StringType
		p.StringValue = new(string)
		*p.StringValue = v
	case *string:
		p.ValueType = StringType
		p.StringValue = new(string)
		*p.StringValue = *v
	case Timestamp:
		p.ValueType = TimeType
		p.TimeValue = new(Timestamp)
		*p.TimeValue = v
	case *Timestamp:
		p.ValueType = TimeType
		p.TimeValue = new(Timestamp)
		*p.TimeValue = *v
	case TypedValue:
		*p = v
	case *TypedValue:
		*p = *v
	default:
		return nil
	}
	return p
}

// ThresholdValue describes threshold
type ThresholdValue struct {
	SampleType MetricSampleType `json:"sampleType"`
	Label      string           `json:"label"`
	Value      *TypedValue      `json:"value"`
}

func (p *ThresholdValue) SetValue(v interface{}) {
	p.Value = NewTypedValue(v)
}

// String implements Stringer interface
func (p ThresholdValue) String() string {
	return fmt.Sprintf("[%s, %s, %s]",
		p.SampleType, p.Label, p.Value.String())
}

// TimeSeries defines a single Metric Sample, its time interval, and 0 or more thresholds
type TimeSeries struct {
	MetricName string           `json:"metricName"`
	SampleType MetricSampleType `json:"sampleType,omitempty"`
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
	Thresholds        []ThresholdValue  `json:"thresholds,omitempty"`
	MetricComputeType ComputeType       `json:"-"`
	MetricExpression  string            `json:"-"`
}

func (p *TimeSeries) AddThreshold(t ThresholdValue) {
	p.Thresholds = append(p.Thresholds, t)
}

func (p *TimeSeries) SetName(s string) {
	p.MetricName = s
}

func (p *TimeSeries) SetSampleType(s MetricSampleType) {
	p.SampleType = s
}

func (p *TimeSeries) SetIntervalEnd(t *Timestamp) {
	if p.Interval == nil {
		p.Interval = new(TimeInterval)
	}
	p.Interval.EndTime = t
}

func (p *TimeSeries) SetIntervalStart(t *Timestamp) {
	if p.Interval == nil {
		p.Interval = new(TimeInterval)
	}
	p.Interval.StartTime = t
}

func (p *TimeSeries) SetValue(v interface{}) {
	p.Value = NewTypedValue(v)
}

func (p *TimeSeries) SetTag(k string, v string) {
	if p.Tags == nil {
		p.Tags = make(map[string]string)
	}
	p.Tags[k] = v
}

func (p *TimeSeries) SetUnit(s UnitType) {
	p.Unit = s
}

// String implements Stringer interface
func (p TimeSeries) String() string {
	return fmt.Sprintf("[%s, %s, %s, %s, %s, %s, %s]",
		p.MetricName, p.SampleType, p.Interval.String(), p.Value.String(),
		p.Tags, p.Unit, p.Thresholds)
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

// String implements Stringer interface
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

// String implements Stringer interface
func (labelDescriptor LabelDescriptor) String() string {
	return fmt.Sprintf("[%s, %s, %s]", labelDescriptor.Description, labelDescriptor.Key, labelDescriptor.ValueType)
}

// ThresholdDescriptor defines a Threshold
type ThresholdDescriptor struct {
	// Key: The threshold key.
	Key   string `json:"key"`
	Value int32  `json:"value"`
}

// String implements Stringer interface
func (thresholdDescriptor ThresholdDescriptor) String() string {
	return fmt.Sprintf("[%s, %d]", thresholdDescriptor.Key, thresholdDescriptor.Value)
}

// TracerContext describes a Transit call
type TracerContext struct {
	AppType    string        `json:"appType"`
	AgentID    string        `json:"agentId"`
	TraceToken string        `json:"traceToken"`
	TimeStamp  *Timestamp    `json:"timeStamp"`
	Version    VersionString `json:"version"`
}

// SetContext sets values if not empty
func (p *TracerContext) SetContext(c TracerContext) {
	if c.AgentID != "" {
		p.AgentID = c.AgentID
	}
	if c.AppType != "" {
		p.AppType = c.AppType
	}
	if c.TraceToken != "" {
		p.TraceToken = c.TraceToken
	}
	if c.Version != "" {
		p.Version = c.Version
	}
	if c.TimeStamp != nil {
		p.TimeStamp = c.TimeStamp
	}
}

// String implements Stringer interface
func (p TracerContext) String() string {
	return fmt.Sprintf("[%s, %s, %s, %s, %s]",
		p.AppType, p.AgentID, p.TraceToken,
		p.TimeStamp, p.Version,
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

// String implements Stringer interface
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

// String implements Stringer interface
func (operationResults OperationResults) String() string {
	return fmt.Sprintf("[%d, %d, %s, %s, %d, %d, %s]",
		operationResults.ResourcesAdded, operationResults.ResourcesDeleted, operationResults.EntityType,
		operationResults.Operation, operationResults.Warning, operationResults.Count, *operationResults.Results,
	)
}

type View struct {
	Name        string                 `json:"name"`
	DisplayName string                 `json:"displayName"`
	Enabled     bool                   `json:"enabled"`
	Extensions  map[string]interface{} `json:"extensions,omitempty"`
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

// String implements Stringer interface
func (monitorConnection MonitorConnection) String() string {
	return fmt.Sprintf("[%d, %s, %s, %s, %t, %s, %s, %d]",
		monitorConnection.ID, monitorConnection.Server, monitorConnection.UserName, monitorConnection.Password,
		monitorConnection.SslEnabled, monitorConnection.URL, monitorConnection.Extensions, monitorConnection.ConnectorID,
	)
}

type MetricsProfile struct {
	Name        string             `json:"name"`
	ProfileType string             `json:"profileType"`
	IsTemplate  bool               `json:"isTemplate"`
	Metrics     []MetricDefinition `json:"metrics"`
}

// String implements Stringer interface
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

// String implements Stringer interface
func (metricDefinition MetricDefinition) String() string {
	return fmt.Sprintf("[%s, %s, %s, %t, %t, %s, %s, %s, %s, %s, %d, %d, %s, %s]",
		metricDefinition.Name, metricDefinition.CustomName, metricDefinition.Description, metricDefinition.Monitored,
		metricDefinition.Graphed, metricDefinition.MetricType, metricDefinition.ComputeType, metricDefinition.ServiceType,
		metricDefinition.SourceType, metricDefinition.AggregateType, metricDefinition.WarningThreshold,
		metricDefinition.CriticalThreshold, metricDefinition.Expression, metricDefinition.Format,
	)
}

// AgentIdentity defines TCG Agent Identity
type AgentIdentity struct {
	AgentID string `json:"agentId" yaml:"agentId"`
	AppName string `json:"appName" yaml:"appName"`
	AppType string `json:"appType" yaml:"appType"`
}

func CalculateResourceStatus(services []MonitoredService) MonitorStatus {

	// TODO: implement logic

	return HostUp
}

func CalculateServiceStatus(metrics *[]TimeSeries) (MonitorStatus, error) {
	if metrics == nil || len(*metrics) == 0 {
		return ServiceUnknown, nil
	}
	previousStatus := ServiceOk
	for _, metric := range *metrics {
		if metric.Thresholds != nil {
			var warning, critical ThresholdValue
			for _, threshold := range metric.Thresholds {
				switch threshold.SampleType {
				case Warning:
					warning = threshold
				case Critical:
					critical = threshold
				default:
					return ServiceOk, fmt.Errorf("unsupported threshold Sample type")
				}
			}

			status := CalculateStatus(metric.Value, warning.Value, critical.Value)
			if MonitorStatusWeightService[status] > MonitorStatusWeightService[previousStatus] {
				previousStatus = status
			}
		}
	}
	return previousStatus, nil
}

func CalculateStatus(value *TypedValue, warning *TypedValue, critical *TypedValue) MonitorStatus {
	if warning == nil && critical == nil {
		return ServiceOk
	}

	var warningValue float64
	var criticalValue float64

	if warning != nil {
		switch warning.ValueType {
		case IntegerType:
			warningValue = float64(*warning.IntegerValue)
		case DoubleType:
			warningValue = *warning.DoubleValue
		}
	}

	if critical != nil {
		switch critical.ValueType {
		case IntegerType:
			criticalValue = float64(*critical.IntegerValue)
		case DoubleType:
			criticalValue = *critical.DoubleValue
		}
	}

	switch value.ValueType {
	case IntegerType:
		if warning == nil && criticalValue == -1 {
			if float64(*value.IntegerValue) >= criticalValue {
				return ServiceUnscheduledCritical
			}
			return ServiceOk
		}
		if critical == nil && (warning != nil && warningValue == -1) {
			if float64(*value.IntegerValue) >= warningValue {
				return ServiceWarning
			}
			return ServiceOk
		}
		if (warning != nil && warningValue == -1) && (critical != nil && criticalValue == -1) {
			return ServiceOk
		}
		// is it a reverse comparison (low to high)
		if (warning != nil && critical != nil) && warningValue > criticalValue {
			if float64(*value.IntegerValue) <= criticalValue {
				return ServiceUnscheduledCritical
			}
			if float64(*value.IntegerValue) <= warningValue {
				return ServiceWarning
			}
			return ServiceOk
		} else {
			if (warning != nil && critical != nil) && float64(*value.IntegerValue) >= criticalValue {
				return ServiceUnscheduledCritical
			}
			if (warning != nil && critical != nil) && float64(*value.IntegerValue) >= warningValue {
				return ServiceWarning
			}
			return ServiceOk
		}
	case DoubleType:
		if warning == nil && criticalValue == -1 {
			if *value.DoubleValue >= criticalValue {
				return ServiceUnscheduledCritical
			}
			return ServiceOk
		}
		if critical == nil && (warning != nil && warningValue == -1) {
			if *value.DoubleValue >= warningValue {
				return ServiceWarning
			}
			return ServiceOk
		}
		if (warning != nil && critical != nil) && (warningValue == -1 || criticalValue == -1) {
			return ServiceOk
		}
		// is it a reverse comparison (low to high)
		if warningValue > criticalValue {
			if *value.DoubleValue <= criticalValue {
				return ServiceUnscheduledCritical
			}
			if *value.DoubleValue <= warningValue {
				return ServiceWarning
			}
			return ServiceOk
		} else {
			if *value.DoubleValue >= criticalValue {
				return ServiceUnscheduledCritical
			}
			if *value.DoubleValue >= warningValue {
				return ServiceWarning
			}
			return ServiceOk
		}
	}
	return ServiceOk
}

// MonitorStatusWeightService defines weight of Monitor Status for multi-state comparison
var MonitorStatusWeightService = map[MonitorStatus]int{
	ServiceOk:                  0,
	ServicePending:             10,
	ServiceUnknown:             20,
	ServiceWarning:             30,
	ServiceScheduledCritical:   50,
	ServiceUnscheduledCritical: 100,
}
