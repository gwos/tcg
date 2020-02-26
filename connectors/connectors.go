package connectors

import (
	"encoding/json"
	"fmt"
	"github.com/gwos/tng/milliseconds"
	"github.com/gwos/tng/services"
	"github.com/gwos/tng/transit"
	"os"
	"os/signal"
	"reflect"
	"syscall"
	"time"
)

var transitService *services.TransitService

func Start() error {
	transitService = services.GetTransitService()
	// TODO: implement from ServerConnector main... start Nats etc
	return nil
}

func SendMetrics(resources []transit.MonitoredResource) error {
	request := transit.ResourcesWithServicesRequest{
		Context:   transitService.MakeTracerContext(),
		Resources: resources,
	}
	// TODO: fill in all resource LastCheck and Next Check

	b, err := json.Marshal(request)
	if err != nil {
		return err
	}
	return transitService.SendResourceWithMetrics(b)

}

func SendInventory(resource []transit.InventoryResource, resourceGroups []transit.ResourceGroup) error {
	// TODO: implement from ServerConnector
	return nil
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

// Metric Constructors

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
		if warning == nil || critical.IntegerValue == -1 {
			if value.IntegerValue >= critical.IntegerValue {
				return transit.ServiceUnscheduledCritical
			}
			return transit.ServiceOk
		}
		if critical == nil || warning.IntegerValue == -1 {
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
		if warning == nil || critical.DoubleValue == -1 {
			if value.DoubleValue >= critical.DoubleValue {
				return transit.ServiceUnscheduledCritical
			}
			return transit.ServiceOk
		}
		if critical == nil || warning.DoubleValue == -1 {
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
		// TODO: shutdown NATS and everything else cleanly, with DRY coding
		os.Exit(0)
	}()
}
