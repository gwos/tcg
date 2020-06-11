package connectors

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/gwos/tcg/config"
	"github.com/gwos/tcg/log"
	"github.com/gwos/tcg/milliseconds"
	"github.com/gwos/tcg/services"
	"github.com/gwos/tcg/transit"
	"hash/fnv"
	"os"
	"os/signal"
	"reflect"
	"syscall"
	"time"
)

const DefaultTimer = int64(120)

// will come from extensions field
var Timer = DefaultTimer

const ExtensionsKeyTimer = "checkIntervalMinutes"

func Start() error {
	if err := services.GetTransitService().StartNats(); err != nil {
		return err
	}
	if err := services.GetTransitService().StartTransport(); err != nil {
		return err
	}
	return nil
}

func RetrieveCommonConnectorInfo(data []byte) (*transit.MonitorConnection, *transit.MetricsProfile, config.GWConnections, error) {
	var connector = struct {
		MonitorConnection transit.MonitorConnection `json:"monitorConnection"`
		MetricsProfile    transit.MetricsProfile    `json:"metricsProfile"`
		Connections       config.GWConnections      `json:"groundworkConnections"`
	}{}

	err := json.Unmarshal(data, &connector)
	if err != nil {
		log.Error("Error parsing common connector data: ", err)
		return nil, nil, nil, err
	}

	return &connector.MonitorConnection, &connector.MetricsProfile, connector.Connections, nil
}

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
	setCheckTimes(resources)
	request := transit.ResourcesWithServicesRequest{
		Context:   services.GetTransitService().MakeTracerContext(),
		Resources: resources,
	}
	b, err = json.Marshal(request)
	if err != nil {
		return err
	}
	return services.GetTransitService().SendResourceWithMetrics(b)
}

func setCheckTimes(resources []transit.MonitoredResource) {
	lastCheckTime := time.Now().Local()
	nextCheckTime := lastCheckTime.Add(time.Second * time.Duration(Timer))
	for i := range resources {
		resources[i].LastCheckTime = milliseconds.MillisecondTimestamp{Time: lastCheckTime}
		resources[i].NextCheckTime = milliseconds.MillisecondTimestamp{Time: nextCheckTime}
		for j := range resources[i].Services {
			resources[i].Services[j].LastCheckTime = milliseconds.MillisecondTimestamp{Time: lastCheckTime}
			resources[i].Services[j].NextCheckTime = milliseconds.MillisecondTimestamp{Time: nextCheckTime}
		}
	}
}

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

	var monitoredResourceRefs []transit.MonitoredResourceRef
	for _, resource := range resources {
		monitoredResourceRefs = append(monitoredResourceRefs,
			transit.MonitoredResourceRef{
				Name: resource.Name,
				Type: transit.Host,
			},
		)
	}

	for i := range resourceGroups {
		resourceGroups[i].Resources = monitoredResourceRefs
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

	err = services.GetTransitService().SynchronizeInventory(b)

	return err
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

// Metric Constructors

type MetricBuilder struct {
	Name           string
	CustomName     string
	Value          interface{}
	UnitType       interface{}
	Warning        interface{}
	Critical       interface{}
	StartTimestamp *milliseconds.MillisecondTimestamp
	EndTimestamp   *milliseconds.MillisecondTimestamp
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
		log.Error("Error creating time interval for metric ", metricBuilder.Name,
			": either start time or end time is not provided")
	}

	metricName := Name(metricBuilder.Name, metricBuilder.CustomName)
	metric, err := CreateMetric(metricName, metricBuilder.Value, args...)
	if err != nil {
		return metric, err
	}

	var thresholds []transit.ThresholdValue
	if metricBuilder.Warning != nil {
		warningThreshold, err := CreateWarningThreshold(metricName+"_wn",
			metricBuilder.Warning)
		if err != nil {
			log.Error("Error creating warning threshold for metric ", metricBuilder.Name,
				": ", err)
		}
		thresholds = append(thresholds, *warningThreshold)
	}
	if metricBuilder.Critical != nil {
		criticalThreshold, err := CreateCriticalThreshold(metricName+"_cr",
			metricBuilder.Critical)
		if err != nil {
			log.Error("Error creating critical threshold for metric ", metricBuilder.Name,
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
		log.Error("Error creating metric for process: ", metricBuilder.Name)
		log.Error(err)
		return nil, errors.New("cannot create service with metric due to metric creation failure")
	}
	serviceName := Name(metricBuilder.Name, metricBuilder.CustomName)
	return CreateService(serviceName, hostName, []transit.TimeSeries{*metric})
}

// Create Service
// required params: name, owner(resource)
// optional params: metrics
func CreateService(name string, owner string, args ...interface{}) (*transit.MonitoredService, error) {
	service := transit.MonitoredService{
		Name:   name,
		Type:   transit.Service,
		Owner:  owner,
		Status: transit.ServiceOk,
	}
	for _, arg := range args {
		switch arg.(type) {
		case []transit.TimeSeries:
			service.Metrics = arg.([]transit.TimeSeries)
		default:
			return nil, fmt.Errorf("unsupported value type: %T", reflect.TypeOf(arg))
		}
	}
	if service.Metrics != nil {
		service.Status, _ = CalculateServiceStatus(&service.Metrics)
	}
	return &service, nil
}

// Create Resource
// required params: name
// optional params: services
func CreateResource(name string, args ...interface{}) (*transit.MonitoredResource, error) {
	resource := transit.MonitoredResource{
		Name:          name,
		Type:          transit.Host,
		Status:        transit.HostUp,
		LastCheckTime: milliseconds.MillisecondTimestamp{},
		NextCheckTime: milliseconds.MillisecondTimestamp{},
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
		if critical == nil && warning.IntegerValue == -1 {
			if value.IntegerValue >= warning.IntegerValue {
				return transit.ServiceWarning
			}
			return transit.ServiceOk
		}
		if warning.IntegerValue == -1 && critical.IntegerValue == -1 {
			return transit.ServiceOk
		}
		// is it a reverse comparison (low to high)
		if warning.IntegerValue > critical.IntegerValue {
			if value.IntegerValue <= critical.IntegerValue {
				return transit.ServiceUnscheduledCritical
			}
			if value.IntegerValue <= warning.IntegerValue {
				return transit.ServiceWarning
			}
			return transit.ServiceOk
		} else {
			if value.IntegerValue >= critical.IntegerValue {
				return transit.ServiceUnscheduledCritical
			}
			if value.IntegerValue >= warning.IntegerValue {
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
		if critical == nil && warning.DoubleValue == -1 {
			if value.DoubleValue >= warning.DoubleValue {
				return transit.ServiceWarning
			}
			return transit.ServiceOk
		}
		if warning.DoubleValue == -1 || critical.DoubleValue == -1 {
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

func ControlCHandler() {
	c := make(chan os.Signal, 2)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		fmt.Println("\r- Ctrl+C pressed in Terminal")
		if err := services.GetTransitService().StopTransport(); err != nil {
			log.Error(err.Error())
		}
		if err := services.GetTransitService().StopNats(); err != nil {
			log.Error(err.Error())
		}
		if err := services.GetTransitService().StopController(); err != nil {
			log.Error(err.Error())
		}
		os.Exit(0)
	}()
}

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

type Version struct {
	Number         string `json:"number"`
	BuildTimestamp string `json:"buildTimestamp"`
}
