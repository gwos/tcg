package connectors

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"hash/fnv"
	"reflect"
	regexp2 "regexp"
	"strconv"
	"strings"
	"time"

	"github.com/gwos/tcg/log"
	"github.com/gwos/tcg/milliseconds"
	"github.com/gwos/tcg/services"
	"github.com/gwos/tcg/transit"
	"go.opentelemetry.io/otel/label"
)

// ExtKeyCheckInterval defines field name
const ExtKeyCheckInterval = "checkIntervalMinutes"

// DefaultCheckInterval defines interval
const DefaultCheckInterval = time.Duration(2) * time.Minute

// CheckInterval comes from extensions field
var CheckInterval = DefaultCheckInterval

// statusTextPattern is a pattern to be found in status message template and replaced by the value extracted with appropriate method stored in statusTextValGetters
// For example: status message template is "The value is {value}"
//    pattern "{value}" will be extracted by statusTextPattern expression
//    getter stored in statusTextValGetters with key="{value}" will be used to build text to replace such pattern,
//       for example it returned "100"
//    pattern "{value}" will be replaced in status message template with the value returned by appropriate method,
//       in this example status text will be "The value is 100"
// if no method for pattern found template won't be modified and will be left containing "{value}" text: "The value is {value}"
const statusTextPattern = `\{(.*?)\}`

var statusTextValGetters = map[string]func(service *transit.DynamicMonitoredService) (string, error){
	"{value}":    extractValueForStatusText,
	"{interval}": extractIntervalForStatusText,
}

const noneThresholdText = "-1"

// UnmarshalConfig updates args with data
func UnmarshalConfig(data []byte, metricsProfile *transit.MetricsProfile, monitorConnection *transit.MonitorConnection) error {
	/* grab CheckInterval from MonitorConnection extensions */
	var s struct {
		MonitorConnection struct {
			Extensions struct {
				ExtensionsKeyTimer int `json:"checkIntervalMinutes"`
			} `json:"extensions"`
		} `json:"monitorConnection"`
	}
	if err := json.Unmarshal(data, &s); err == nil {
		if s.MonitorConnection.Extensions.ExtensionsKeyTimer > 0 {
			CheckInterval = time.Minute * time.Duration(s.MonitorConnection.Extensions.ExtensionsKeyTimer)
		} else {
			CheckInterval = DefaultCheckInterval
		}
	}
	/* process args */
	cfg := struct {
		MetricsProfile    *transit.MetricsProfile    `json:"metricsProfile"`
		MonitorConnection *transit.MonitorConnection `json:"monitorConnection"`
	}{metricsProfile, monitorConnection}
	return json.Unmarshal(data, &cfg)
}

// Start starts services
func Start() error {
	if err := services.GetTransitService().StartNats(); err != nil {
		return err
	}
	if err := services.GetTransitService().StartTransport(); err != nil {
		return err
	}
	return nil
}

// SendMetrics processes metrics payload
func SendMetrics(ctx context.Context, resources []transit.DynamicMonitoredResource, groups *[]transit.ResourceGroup) error {
	var (
		b   []byte
		err error
	)
	ctxN, span := services.StartTraceSpan(ctx, "connectors", "SendMetrics")
	defer func() {
		span.SetAttributes(
			label.Int("payloadLen", len(b)),
			label.String("error", fmt.Sprint(err)),
		)
		span.End()
	}()

	request := transit.DynamicResourcesWithServicesRequest{
		Context:   services.GetTransitService().MakeTracerContext(),
		Resources: resources,
	}
	if groups != nil {
		request.Groups = *groups
	}
	for i := range request.Resources {
		monitoredServices := EvaluateExpressions(request.Resources[i].Services)
		request.Resources[i].Services = monitoredServices
		request.Resources[i].LastPlugInOutput = buildHostStatusText(monitoredServices)
	}
	b, err = json.Marshal(request)
	if err != nil {
		return err
	}
	err = services.GetTransitService().SendResourceWithMetrics(ctxN, b)
	return err
}

// SendInventory processes inventory payload
func SendInventory(ctx context.Context, resources []transit.DynamicInventoryResource, resourceGroups []transit.ResourceGroup, ownershipType transit.HostOwnershipType) error {
	var (
		b   []byte
		err error
	)
	ctxN, span := services.StartTraceSpan(ctx, "connectors", "SendInventory")
	defer func() {
		span.SetAttributes(
			label.Int("payloadLen", len(b)),
			label.String("error", fmt.Sprint(err)),
		)
		span.End()
	}()

	request := transit.DynamicInventoryRequest{
		Context:       services.GetTransitService().MakeTracerContext(),
		OwnershipType: ownershipType,
		Resources:     resources,
		Groups:        resourceGroups,
	}
	b, err = json.Marshal(request)
	if err != nil {
		return err
	}
	err = services.GetTransitService().SynchronizeInventory(ctxN, b)
	return err
}

// Inventory Constructors
func CreateInventoryService(name string, owner string) transit.DynamicInventoryService {
	return transit.DynamicInventoryService{
		BaseTransitData: transit.BaseTransitData{
			Name:  name,
			Type:  transit.Service,
			Owner: owner,
		},
	}
}

// makes and modifies a copy, doesn't modify services
func CreateInventoryResource(name string, services []transit.DynamicInventoryService) transit.DynamicInventoryResource {
	resource := transit.DynamicInventoryResource{
		BaseResource: transit.BaseResource{
			BaseTransitData: transit.BaseTransitData{
				Name: name,
				Type: transit.Host,
			},
		},
	}
	for _, s := range services {
		resource.Services = append(resource.Services, s)
	}
	return resource
}

func CreateMonitoredResourceRef(name string, owner string, resourceType transit.ResourceType) transit.MonitoredResourceRef {
	resource := transit.MonitoredResourceRef{
		Name:  name,
		Type:  resourceType,
		Owner: owner,
	}
	return resource
}

func CreateResourceGroup(name string, description string, groupType transit.GroupType, resources []transit.MonitoredResourceRef) transit.ResourceGroup {
	group := transit.ResourceGroup{
		GroupName:   name,
		Type:        groupType,
		Description: description,
	}
	for _, r := range resources {
		group.Resources = append(group.Resources, r)
	}
	return group
}

func FillGroupWithResources(group transit.ResourceGroup, resources []transit.DynamicInventoryResource) transit.ResourceGroup {
	var monitoredResourceRefs []transit.MonitoredResourceRef
	for _, resource := range resources {
		monitoredResourceRefs = append(monitoredResourceRefs,
			transit.MonitoredResourceRef{
				Name: resource.BaseResource.Name,
				Type: resource.BaseResource.Type,
			},
		)
	}
	group.Resources = monitoredResourceRefs
	return group
}

// Metric Constructors

type MetricBuilder struct {
	Name           string
	CustomName     string
	ComputeType    transit.ComputeType
	Expression     string
	Value          interface{}
	UnitType       interface{}
	Warning        interface{}
	Critical       interface{}
	StartTimestamp *milliseconds.MillisecondTimestamp
	EndTimestamp   *milliseconds.MillisecondTimestamp
	Graphed        bool
	Tags           map[string]string
}

// Creates metric based on data provided with metricBuilder
func BuildMetric(metricBuilder MetricBuilder) (*transit.TimeSeries, error) {
	var args []interface{}
	if metricBuilder.UnitType != nil {
		args = append(args, metricBuilder.UnitType)
	}
	if metricBuilder.StartTimestamp != nil && metricBuilder.EndTimestamp != nil {
		timeInterval := &transit.TimeInterval{
			StartTime: *metricBuilder.StartTimestamp,
			EndTime:   *metricBuilder.EndTimestamp,
		}
		args = append(args, timeInterval)
	} else if metricBuilder.StartTimestamp != nil || metricBuilder.EndTimestamp != nil {
		log.Error("|connectors.go| : [BuildMetric] : Error creating time interval for metric ", metricBuilder.Name,
			": either start time or end time is not provided")
	}
	if metricBuilder.Tags != nil && len(metricBuilder.Tags) != 0 {
		args = append(args, metricBuilder.Tags)
	}

	metricName := Name(metricBuilder.Name, metricBuilder.CustomName)
	metric, err := CreateMetric(metricName, metricBuilder.Value, args...)
	if err != nil {
		return metric, err
	}

	metric.MetricComputeType = metricBuilder.ComputeType
	metric.MetricExpression = metricBuilder.Expression

	var thresholds []transit.ThresholdValue
	if metricBuilder.Warning != nil {
		warningThreshold, err := CreateWarningThreshold(metricName+"_wn",
			metricBuilder.Warning)
		if err != nil {
			log.Error("|connectors.go| : [BuildMetric]: Error creating warning threshold for metric ", metricBuilder.Name,
				": ", err)
		}
		thresholds = append(thresholds, *warningThreshold)
	}
	if metricBuilder.Critical != nil {
		criticalThreshold, err := CreateCriticalThreshold(metricName+"_cr",
			metricBuilder.Critical)
		if err != nil {
			log.Error("|connectors.go| : [BuildMetric] : Error creating critical threshold for metric ", metricBuilder.Name,
				": ", err)
		}
		thresholds = append(thresholds, *criticalThreshold)
	}
	if len(thresholds) > 0 {
		metric.Thresholds = &thresholds
	}

	return metric, nil
}

// CreateMetric
//	required parameters: name, value
//  optional parameters: interval, UnitType
// CreateMetric("cpu", 30)  			// integer value
// CreateMetric("percentFree", 30.59)   // double value
// CreateMetric("cpu", 30.59, transit.PercentCPU) // with optional Unit
// CreateMetric("cpu", 30.59, interval) // with optional interval
// CreateMetric("cpu", 30.59, transit.PercentCPU, interval) // with optional Unit and optional interval
// with optional Unit and optional Interval and optional UnitType
// CreateMetric("cpu", 30.59, interval, transit.PercentCPU)
// Thresholds must be set separately
//
func CreateMetric(name string, value interface{}, args ...interface{}) (*transit.TimeSeries, error) {
	// set the value based on variable type
	var typedValue transit.TypedValue
	switch value.(type) {
	case int:
		typedValue = transit.TypedValue{
			ValueType:    transit.IntegerType,
			IntegerValue: int64(value.(int)),
		}
	case int64:
		typedValue = transit.TypedValue{
			ValueType:    transit.IntegerType,
			IntegerValue: value.(int64),
		}
	case float64:
		typedValue = transit.TypedValue{
			ValueType:   transit.DoubleType,
			DoubleValue: value.(float64),
		}
	default:
		return nil, fmt.Errorf("unsupported value type: %T", reflect.TypeOf(value))
	}
	metric := transit.TimeSeries{
		MetricName: name,
		SampleType: transit.Value,
		Value:      &typedValue,
	}
	// optional argument processing
	// var arguments []interface{} = make([]interface{}, len(args))
	for _, arg := range args {
		switch arg.(type) {
		case string:
			metric.Unit = transit.UnitType(arg.(string))
		case transit.UnitType:
			metric.Unit = arg.(transit.UnitType)
		case *transit.TimeInterval:
			metric.Interval = arg.(*transit.TimeInterval)
		case map[string]string:
			metric.Tags = arg.(map[string]string)
		//case transit.MetricSampleType:
		//	metric.SampleType = arg.(transit.MetricSampleType)
		default:
			return nil, fmt.Errorf("unsupported arg type: %T", reflect.TypeOf(arg))
		}
	}
	// optional interval
	if metric.Interval == nil {
		interval := time.Now()
		metric.Interval = &transit.TimeInterval{
			EndTime:   milliseconds.MillisecondTimestamp{Time: interval},
			StartTime: milliseconds.MillisecondTimestamp{Time: interval},
		}
	}
	if metric.Unit == "" {
		metric.Unit = transit.UnitCounter
	}
	return &metric, nil
}

func CreateWarningThreshold(label string, value interface{}) (*transit.ThresholdValue, error) {
	return CreateThreshold(transit.Warning, label, value)
}

func CreateCriticalThreshold(label string, value interface{}) (*transit.ThresholdValue, error) {
	return CreateThreshold(transit.Critical, label, value)
}

func CreateThreshold(thresholdType transit.MetricSampleType, label string, value interface{}) (*transit.ThresholdValue, error) {
	// create the threshold type
	// set the value based on variable type
	var typedValue transit.TypedValue
	switch value.(type) {
	case int:
		typedValue = transit.TypedValue{
			ValueType:    transit.IntegerType,
			IntegerValue: int64(value.(int)),
		}
	case int64:
		typedValue = transit.TypedValue{
			ValueType:    transit.IntegerType,
			IntegerValue: value.(int64),
		}
	case float64:
		typedValue = transit.TypedValue{
			ValueType:   transit.DoubleType,
			DoubleValue: value.(float64),
		}
	default:
		return nil, fmt.Errorf("unsupported value type: %T", reflect.TypeOf(value))
	}
	// create the threshold
	threshold := transit.ThresholdValue{
		SampleType: thresholdType,
		Label:      label,
		Value:      &typedValue,
	}
	return &threshold, nil
}

// Creates metric based on data provided in metric builder and if metric successfully created
// creates service with same name as metric which contains only this one metric
// returns the result of service creation
func BuildServiceForMetric(hostName string, metricBuilder MetricBuilder) (*transit.DynamicMonitoredService, error) {
	metric, err := BuildMetric(metricBuilder)
	if err != nil {
		log.Error("|connectors.go| : [BuildServiceForMetric] : Error creating metric for process: ", metricBuilder.Name,
			" Reason: ", err)
		return nil, errors.New("cannot create service with metric due to metric creation failure")
	}
	serviceName := Name(metricBuilder.Name, metricBuilder.CustomName)

	serviceProperties := make(map[string]interface{})
	serviceProperties["isGraphed"] = metricBuilder.Graphed

	return CreateService(serviceName, hostName, []transit.TimeSeries{*metric}, serviceProperties)
}

func BuildServiceForMetricWithStatusText(hostName string, metricBuilder MetricBuilder,
	statusMessages map[transit.MonitorStatus]string) (*transit.DynamicMonitoredService, error) {
	service, err := BuildServiceForMetric(hostName, metricBuilder)
	if err != nil {
		log.Error("|connectors.go| : [BuildServiceForMetricWithStatusText] : Error when creating service ",
			hostName, ":", metricBuilder.CustomName, " Reason: ", err)
		return service, err
	}
	// if no message for status provided in a map patternMessage is "", thresholds will be texted anyway if they exist
	patternMessage := statusMessages[service.Status]
	addServiceStatusText(patternMessage, service)
	return service, err
}

func BuildServiceForMultiMetric(hostName string, serviceName string, customName string, metricBuilders []MetricBuilder) (*transit.DynamicMonitoredService, error) {
	metrics := make([]transit.TimeSeries, len(metricBuilders))
	for index, metricBuilder := range metricBuilders {
		metric, err := BuildMetric(metricBuilder)
		if err != nil {
			log.Error("Error creating metric for process: ", metricBuilder.Name)
			log.Error(err)
			return nil, errors.New("cannot create service with metric due to metric creation failure")
		}
		metrics[index] = *metric
	}
	gwServiceName := Name(serviceName, customName)
	return CreateService(gwServiceName, hostName, metrics)
}

func BuildServiceForMetrics(serviceName string, hostName string, metricBuilders []MetricBuilder) (*transit.DynamicMonitoredService, error) {
	var timeSeries []transit.TimeSeries
	for _, metricBuilder := range metricBuilders {
		metric, err := BuildMetric(metricBuilder)
		if err != nil {
			log.Error("|connectors.go| : [BuildServiceForMetrics]: Error creating metric for process: ", serviceName,
				". With metric: ", metricBuilder.Name, "\n\t", err.Error())
			return nil, errors.New("cannot create service with metric due to metric creation failure")
		}
		timeSeries = append(timeSeries, *metric)
	}
	return CreateService(serviceName, hostName, timeSeries)
}

// CreateService makes node
// required params: name, owner(resource)
// optional params: metrics
func CreateService(name string, owner string, args ...interface{}) (*transit.DynamicMonitoredService, error) {
	service := transit.DynamicMonitoredService{
		BaseTransitData: transit.BaseTransitData{
			Name:  name,
			Type:  transit.Service,
			Owner: owner,
		},
		Status:        transit.ServiceOk,
		LastCheckTime: milliseconds.MillisecondTimestamp{Time: time.Now().Local()},
		NextCheckTime: milliseconds.MillisecondTimestamp{Time: time.Now().Local().Add(CheckInterval)},
	}
	for _, arg := range args {
		switch arg.(type) {
		case []transit.TimeSeries:
			service.Metrics = arg.([]transit.TimeSeries)
		case map[string]interface{}:
			service.CreateProperties(arg.(map[string]interface{}))
		default:
			return nil, fmt.Errorf("unsupported value type: %T", reflect.TypeOf(arg))
		}
	}
	if service.Metrics != nil {
		service.Status, _ = CalculateServiceStatus(&service.Metrics)
	}
	return &service, nil
}

// CreateResource makes node
// required params: name
// optional params: services
func CreateResource(name string, args ...interface{}) (*transit.DynamicMonitoredResource, error) {
	resource := transit.DynamicMonitoredResource{
		BaseResource: transit.BaseResource{
			BaseTransitData: transit.BaseTransitData{
				Name: name,
				Type: transit.Host,
			},
		},
		Status:        transit.HostUp,
		LastCheckTime: milliseconds.MillisecondTimestamp{Time: time.Now().Local()},
		NextCheckTime: milliseconds.MillisecondTimestamp{Time: time.Now().Local().Add(CheckInterval)},
	}
	for _, arg := range args {
		switch arg.(type) {
		case []transit.DynamicMonitoredService:
			resource.Services = arg.([]transit.DynamicMonitoredService)
		default:
			return nil, fmt.Errorf("unsupported value type: %T", reflect.TypeOf(arg))
		}
	}
	// TODO: trickle up calculation from services?
	return &resource, nil
}

func CalculateResourceStatus(services []transit.DynamicMonitoredService) transit.MonitorStatus {

	// TODO: implement logic

	return transit.HostUp
}

func CalculateServiceStatus(metrics *[]transit.TimeSeries) (transit.MonitorStatus, error) {
	previousStatus := transit.ServiceOk
	for _, metric := range *metrics {
		if metric.Thresholds != nil {
			var warning, critical transit.ThresholdValue
			for _, threshold := range *metric.Thresholds {
				switch threshold.SampleType {
				case transit.Warning:
					warning = threshold
				case transit.Critical:
					critical = threshold
				default:
					return transit.ServiceOk, fmt.Errorf("unsupported threshold Sample type")
				}
			}
			status := CalculateStatus(metric.Value, warning.Value, critical.Value)
			if monitorStatusWeightService[status] > monitorStatusWeightService[previousStatus] {
				previousStatus = status
			}
		}
	}
	return previousStatus, nil
}

// Weight of Monitor Status for multi-state comparison
var monitorStatusWeightService = map[transit.MonitorStatus]int{
	transit.ServiceOk:                  0,
	transit.ServicePending:             10,
	transit.ServiceUnknown:             20,
	transit.ServiceWarning:             30,
	transit.ServiceScheduledCritical:   50,
	transit.ServiceUnscheduledCritical: 100,
}

func CalculateStatus(value *transit.TypedValue, warning *transit.TypedValue, critical *transit.TypedValue) transit.MonitorStatus {
	if warning == nil && critical == nil {
		return transit.ServiceOk
	}
	switch value.ValueType {
	case transit.IntegerType:
		if warning == nil && critical.IntegerValue == -1 {
			if value.IntegerValue >= critical.IntegerValue {
				return transit.ServiceUnscheduledCritical
			}
			return transit.ServiceOk
		}
		if critical == nil && (warning != nil && warning.IntegerValue == -1) {
			if value.IntegerValue >= warning.IntegerValue {
				return transit.ServiceWarning
			}
			return transit.ServiceOk
		}
		if (warning != nil && warning.IntegerValue == -1) && (critical != nil && critical.IntegerValue == -1) {
			return transit.ServiceOk
		}
		// is it a reverse comparison (low to high)
		if (warning != nil && critical != nil) && warning.IntegerValue > critical.IntegerValue {
			if value.IntegerValue <= critical.IntegerValue {
				return transit.ServiceUnscheduledCritical
			}
			if value.IntegerValue <= warning.IntegerValue {
				return transit.ServiceWarning
			}
			return transit.ServiceOk
		} else {
			if (warning != nil && critical != nil) && value.IntegerValue >= critical.IntegerValue {
				return transit.ServiceUnscheduledCritical
			}
			if (warning != nil && critical != nil) && value.IntegerValue >= warning.IntegerValue {
				return transit.ServiceWarning
			}
			return transit.ServiceOk
		}
	case transit.DoubleType:
		if warning == nil && critical.DoubleValue == -1 {
			if value.DoubleValue >= critical.DoubleValue {
				return transit.ServiceUnscheduledCritical
			}
			return transit.ServiceOk
		}
		if critical == nil && (warning != nil && warning.DoubleValue == -1) {
			if value.DoubleValue >= warning.DoubleValue {
				return transit.ServiceWarning
			}
			return transit.ServiceOk
		}
		if (warning != nil && critical != nil) && (warning.DoubleValue == -1 || critical.DoubleValue == -1) {
			return transit.ServiceOk
		}
		// is it a reverse comparison (low to high)
		if warning.DoubleValue > critical.DoubleValue {
			if value.DoubleValue <= critical.DoubleValue {
				return transit.ServiceUnscheduledCritical
			}
			if value.DoubleValue <= warning.DoubleValue {
				return transit.ServiceWarning
			}
			return transit.ServiceOk
		} else {
			if value.DoubleValue >= critical.DoubleValue {
				return transit.ServiceUnscheduledCritical
			}
			if value.DoubleValue >= warning.DoubleValue {
				return transit.ServiceWarning
			}
			return transit.ServiceOk
		}
	}
	return transit.ServiceOk
}

// EvaluateExpressions calculates synthetic metrics
func EvaluateExpressions(services []transit.DynamicMonitoredService) []transit.DynamicMonitoredService {
	var result []transit.DynamicMonitoredService
	vars := make(map[string]interface{})

	for _, service := range services {
		for _, metric := range service.Metrics {
			if metric.MetricComputeType != transit.Synthetic {
				switch metric.Value.ValueType {
				case transit.IntegerType:
					vars[strings.ReplaceAll(metric.MetricName, ".", "_")] = float64(metric.Value.IntegerValue)
				case transit.DoubleType:
					vars[strings.ReplaceAll(metric.MetricName, ".", "_")] = metric.Value.DoubleValue
				}
			}
		}
		result = append(result, service)
	}

	for i := range result {
		for _, metric := range result[i].Metrics {
			if metric.MetricComputeType == transit.Synthetic {
				if value, _, err := EvaluateGroundworkExpression(metric.MetricExpression, vars, 0); err != nil {
					log.Error("|connectors.go| : [EvaluateExpressions] : ", err)
					continue
				} else {
					result[i] = transit.DynamicMonitoredService{
						BaseTransitData: transit.BaseTransitData{
							Name:  result[i].Name,
							Type:  transit.Service,
							Owner: result[i].Owner,
						},
						LastPlugInOutput: fmt.Sprintf(" Expression: %s", metric.MetricExpression),
						LastCheckTime:    milliseconds.MillisecondTimestamp{Time: time.Now().Local()},
						NextCheckTime:    milliseconds.MillisecondTimestamp{Time: time.Now().Local().Add(CheckInterval)},
						Metrics: []transit.TimeSeries{
							{
								MetricName: metric.MetricName,
								SampleType: transit.Value,
								Interval: &transit.TimeInterval{
									EndTime:   milliseconds.MillisecondTimestamp{Time: time.Now().Local()},
									StartTime: milliseconds.MillisecondTimestamp{Time: time.Now().Local()},
								},
								Thresholds: metric.Thresholds,
								Value: &transit.TypedValue{
									ValueType:    metric.Value.ValueType,
									IntegerValue: int64(value),
									DoubleValue:  value,
								},
							},
						},
					}
					status, err := CalculateServiceStatus(&result[i].Metrics)
					result[i].Status = status
					if err != nil {
						log.Error(fmt.Sprintf("|connectors.go| : [EvaluateExpressions] : %s (default status - \"STATUS_OK\")", err))
					}
				}
			}
		}
	}
	return result
}

// Name defines metric name
func Name(defaultName string, customName string) string {
	if customName == "" {
		return defaultName
	}
	return customName
}

// Hashsum calculates FNV non-cryptographic hash suitable for checking the equality
func Hashsum(args ...interface{}) ([]byte, error) {
	var b bytes.Buffer
	for _, arg := range args {
		s, err := json.Marshal(arg)
		if err != nil {
			return nil, err
		}
		if _, err := b.Write(s); err != nil {
			return nil, err
		}
	}
	h := fnv.New128()
	if _, err := h.Write(b.Bytes()); err != nil {
		return nil, err
	}
	return h.Sum(nil), nil
}

// MaxDuration returns maximum value
func MaxDuration(x time.Duration, rest ...time.Duration) time.Duration {
	m := x
	for _, y := range rest[:] {
		if m < y {
			m = y
		}
	}
	return m
}

// StartPeriodic starts periodic event loop
// context can provide ctx.Done channel
// and "notOften" guard with 1 minute defaults
func StartPeriodic(ctx context.Context, t time.Duration, fn func()) {
	notOften := time.Minute
	if ctx == nil {
		ctx = context.Background()
	}
	if v := ctx.Value("notOften"); v != nil {
		notOften = v.(time.Duration)
	}
	ticker := time.NewTicker(MaxDuration(t, notOften))
	handler := func() {
		defer func() {
			if err := recover(); err != nil {
				log.Error("[Recovered]: ", err)
			}
		}()
		fn()
	}
	go func() {
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				go handler()
			}
		}
	}()
	/* call handler immediately */
	go handler()
}

// Returns duration provided as a "human-readable" string, like: "N second(s)/minute(s)/hour(s)/day(s)"
// Following formatting rules are being applied by default (not considering minRound):
//    if the value is less than 3 minutes it will be formatted to seconds
//    if the value is between 3 minutes and 3 hours it will be formatted to minutes
//    if the value is between 3 hours and 3 days (<73 hours) it will be formatted to hours
//    values higher than 72 hours (3 days) will be formatted to days
// minRound arg determines time unit "limit" lower than that formatting is not allowed
// i.e. if the value is for example 120 seconds which is gonna be returned as "120 second(s)" by default rules
//  will be returned as "2 minute(s)" if minRound="minute" (i.e. formatting to lower than minute is not allowed)
//  or "0 hour(s)" if minRound="hour" (i.e. formatting to lower than hour is not allowed)
func FormatTimeForStatusMessage(value time.Duration, minRound time.Duration) string {
	h := value.Hours()
	m := value.Minutes()
	s := value.Seconds()
	// if duration is more than 3 hours or minRound doesn't allow to format to "lower" than hour
	if h > 3 || minRound > time.Minute {
		if h < 73 {
			// if duration is between 3 and 73 hours format to hours
			return fmt.Sprintf("%.0f hour(s)", h)
		} else {
			// if duration is 73 hours or more (i.e. 3+ days) format to days
			d, _ := time.ParseDuration(fmt.Sprintf("%.0fh", h))
			days := d.Hours() / 24
			return fmt.Sprintf("%.0f day(s)", days)
		}
		// extend with months, years
	}
	// if duration is less than 3 hours but more than 3 minutes or minRound doesn't allow to format to "lower" than minute format to minutes
	if m > 3 || minRound > time.Second {
		return fmt.Sprintf("%.0f minute(s)", m)
	}
	// if duration is less than 3 minutes and minRound allows formatting to seconds format to seconds
	return fmt.Sprintf("%.0f second(s)", s)
}

func addServiceStatusText(patternMessage string, service *transit.DynamicMonitoredService) {
	if service == nil {
		log.Error("|connectors.go| : [addServiceStatusText] : service is nil")
		return
	}
	re := regexp2.MustCompile(statusTextPattern)
	patterns := re.FindAllString(patternMessage, -1)
	for _, pattern := range patterns {
		if fn, has := statusTextValGetters[pattern]; has {
			replacement, err := fn(service)
			if err != nil {
				log.Warn("|connectors.go| : [addServiceStatusText] : Failed to replace pattern: ",
					pattern, " in message ", pattern, "for service ", service.Name, " : ", err)
			} else {
				patternMessage = strings.ReplaceAll(patternMessage, pattern, replacement)
			}
		} else {
			log.Warn("|connectors.go| : [addServiceStatusText] : No method to get service value for pattern: ", pattern)
		}
	}
	statusText := addThresholdsToStatusText(patternMessage, service)
	service.LastPlugInOutput = statusText
}

func addThresholdsToStatusText(statusText string, service *transit.DynamicMonitoredService) string {
	if service == nil {
		log.Error("|connectors.go| : [addThresholdsStatusText] : service is nil")
		return statusText
	}
	if len(service.Metrics) == 1 {
		wt := noneThresholdText
		crt := noneThresholdText
		metric := service.Metrics[0]
		if metric.Thresholds != nil {
			for _, th := range *metric.Thresholds {
				thresholdType := th.SampleType
				if transit.Warning == thresholdType || transit.Critical == thresholdType {
					value, err := getValueText(th.Value)
					if err == nil {
						switch thresholdType {
						case transit.Warning:
							wt = value
						case transit.Critical:
							crt = value
						}
					} else {
						log.Warn("|connectors.go| : [addThresholdsToStatusText] :"+
							" Failed to get ", thresholdType, " threshold for service ", service.Name, " : ", err)
					}
				}
			}
			var thText string
			if noneThresholdText != wt && noneThresholdText != crt {
				thText = strings.ReplaceAll(" [W/C={w}/{cr}]", "{w}", wt)
				thText = strings.ReplaceAll(thText, "{cr}", crt)
			} else if noneThresholdText != wt {
				thText = strings.ReplaceAll(" [WARN={w}]", "{w}", wt)
			} else if noneThresholdText != crt {
				thText = strings.ReplaceAll(" [CRITICAL={cr}]", "{cr}", crt)
			}
			statusText = statusText + thText
		}
	} else {
		log.Warn("|connectors.go| : [addThresholdsToStatusText] : Not supported for service with more than one metric")
	}
	return statusText
}

func extractValueForStatusText(service *transit.DynamicMonitoredService) (string, error) {
	if len(service.Metrics) == 1 {
		return getValueText(service.Metrics[0].Value)
	}
	log.Warn("|connectors.go| : [extractValueForStatusText] : Not supported for service with more than one metric")
	return "", errors.New("not supported for service with more than one metric")
}

func getValueText(value *transit.TypedValue) (string, error) {
	if value == nil {
		log.Warn("|connectors.go| : [getValueText] : no value")
		return "", errors.New("no value")
	}
	switch value.ValueType {
	case transit.IntegerType:
		return strconv.Itoa(int(value.IntegerValue)), nil
	case transit.DoubleType:
		return fmt.Sprintf("%f", value.DoubleValue), nil
	case transit.StringType:
		return value.StringValue, nil
	case transit.BooleanType:
		return strconv.FormatBool(value.BoolValue), nil
	case transit.TimeType:
		return FormatTimeForStatusMessage(time.Duration(value.TimeValue.UnixNano()), time.Second), nil
	}
	log.Warn("|connectors.go| : [getValueText] : Unknown value type")
	return "", errors.New("unknown value type")
}

func extractIntervalForStatusText(service *transit.DynamicMonitoredService) (string, error) {
	return FormatTimeForStatusMessage(CheckInterval, time.Minute), nil
}

func buildHostStatusText(services []transit.DynamicMonitoredService) string {
	var ok, warn, critical, other int
	for _, service := range services {
		switch service.Status {
		case transit.ServiceOk:
			ok = ok + 1
			break
		case transit.ServiceWarning:
			warn = warn + 1
			break
		case transit.ServiceScheduledCritical:
		case transit.ServiceUnscheduledCritical:
			critical = critical + 1
			break
		default:
			other = other + 1
			break
		}
	}
	return fmt.Sprintf("Host has %d OK, %d WARNING, %d CRITICAL and %d other services.",
		ok, warn, critical, other)
}
