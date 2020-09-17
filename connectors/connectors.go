package connectors

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/gwos/tcg/log"
	"github.com/gwos/tcg/milliseconds"
	"github.com/gwos/tcg/services"
	"github.com/gwos/tcg/transit"
	"hash/fnv"
	"reflect"
	"strings"
	"time"
)

// ExtKeyCheckInterval defines field name
const ExtKeyCheckInterval = "checkIntervalMinutes"

// DefaultCheckInterval defines interval
const DefaultCheckInterval = time.Duration(2) * time.Minute

// CheckInterval comes from extensions field
var CheckInterval = DefaultCheckInterval

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
func SendMetrics(resources []transit.MonitoredResource) error {
	var b []byte
	var err error
	if services.GetTransitService().TelemetryProvider != nil {
		tr := services.GetTransitService().TelemetryProvider.Tracer("connectors")
		_, span := tr.Start(context.Background(), "SendMetrics")
		defer func() {
			span.SetAttribute("error", err)
			span.SetAttribute("payloadLen", len(b))
			span.End()
		}()
	}
	request := transit.ResourcesWithServicesRequest{
		Context:   services.GetTransitService().MakeTracerContext(),
		Resources: resources,
	}
	for i := range request.Resources {
		request.Resources[i].Services = EvaluateExpressions(request.Resources[i].Services)
	}

	b, err = json.Marshal(request)
	if err != nil {
		return err
	}
	return services.GetTransitService().SendResourceWithMetrics(b)
}

// SendInventory processes inventory payload
func SendInventory(resources []transit.InventoryResource, resourceGroups []transit.ResourceGroup, ownershipType transit.HostOwnershipType) error {
	var b []byte
	var err error
	if services.GetTransitService().TelemetryProvider != nil {
		tr := services.GetTransitService().TelemetryProvider.Tracer("connectors")
		_, span := tr.Start(context.Background(), "SendInventory")
		defer func() {
			span.SetAttribute("error", err)
			span.SetAttribute("payloadLen", len(b))
			span.End()
		}()
	}

	inventoryRequest := transit.InventoryRequest{
		Context:       services.GetTransitService().MakeTracerContext(),
		OwnershipType: ownershipType,
		Resources:     resources,
		Groups:        resourceGroups,
	}

	b, err = json.Marshal(inventoryRequest)
	if err != nil {
		return err
	}

	return services.GetTransitService().SynchronizeInventory(b)
}

// Inventory Constructors
func CreateInventoryService(name string, owner string) transit.InventoryService {
	return transit.InventoryService{
		Name:  name,
		Type:  transit.Service,
		Owner: owner,
	}
}

// makes and modifies a copy, doesn't modify services
func CreateInventoryResource(name string, services []transit.InventoryService) transit.InventoryResource {
	resource := transit.InventoryResource{
		Name: name,
		Type: transit.Host,
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

func FillGroupWithResources(group transit.ResourceGroup, resources []transit.InventoryResource) transit.ResourceGroup {
	var monitoredResourceRefs []transit.MonitoredResourceRef
	for _, resource := range resources {
		monitoredResourceRefs = append(monitoredResourceRefs,
			transit.MonitoredResourceRef{
				Name: resource.Name,
				Type: resource.Type,
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
func BuildServiceForMetric(hostName string, metricBuilder MetricBuilder) (*transit.MonitoredService, error) {
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

func BuildServiceForMetrics(serviceName string, hostName string, metricBuilders []MetricBuilder) (*transit.MonitoredService, error) {
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
func CreateService(name string, owner string, args ...interface{}) (*transit.MonitoredService, error) {
	service := transit.MonitoredService{
		Name:          name,
		Type:          transit.Service,
		Owner:         owner,
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
func CreateResource(name string, args ...interface{}) (*transit.MonitoredResource, error) {
	resource := transit.MonitoredResource{
		Name:          name,
		Type:          transit.Host,
		Status:        transit.HostUp,
		LastCheckTime: milliseconds.MillisecondTimestamp{Time: time.Now().Local()},
		NextCheckTime: milliseconds.MillisecondTimestamp{Time: time.Now().Local().Add(CheckInterval)},
	}
	for _, arg := range args {
		switch arg.(type) {
		case []transit.MonitoredService:
			resource.Services = arg.([]transit.MonitoredService)
		default:
			return nil, fmt.Errorf("unsupported value type: %T", reflect.TypeOf(arg))
		}
	}
	// TODO: trickle up calculation from services?
	return &resource, nil
}

func CalculateResourceStatus(services []transit.MonitoredService) transit.MonitorStatus {

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

// EvaluateExpressions calcs synthetic metrics
func EvaluateExpressions(services []transit.MonitoredService) []transit.MonitoredService {
	var result []transit.MonitoredService
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
					result[i] = transit.MonitoredService{
						Name:             result[i].Name,
						Type:             transit.Service,
						Owner:            result[i].Owner,
						Status:           "SERVICE_OK",
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
