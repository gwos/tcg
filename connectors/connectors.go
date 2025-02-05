package connectors

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/gwos/tcg/config"
	"github.com/gwos/tcg/sdk/transit"
	"github.com/gwos/tcg/services"
	"github.com/gwos/tcg/tracing"
	"github.com/rs/zerolog/log"
)

// ExtKeyCheckInterval defines field name
const ExtKeyCheckInterval = "checkIntervalMinutes"

// DefaultCheckInterval defines interval
const DefaultCheckInterval = time.Minute * 2

// CheckInterval comes from extensions field
var CheckInterval = DefaultCheckInterval

var ErrUnsupportedType = errors.New("unsupported value type")

// statusTextPattern is a pattern to be found in status message template and replaced by the value extracted with appropriate method stored in statusTextValGetters
//
//	For example: status message template is "The value is {value}"
//		pattern "{value}" will be extracted by statusTextPattern expression
//		getter stored in statusTextValGetters with key="{value}" will be used to build text to replace such pattern,
//			for example it returned "100"
//		pattern "{value}" will be replaced in status message template with the value returned by appropriate method,
//			in this example status text will be "The value is 100"
//	if no method for pattern found template won't be modified and will be left containing "{value}" text: "The value is {value}"
const statusTextPattern = `\{(.*?)\}`

var statusTextValGetters = map[string]func(service *transit.MonitoredService) (string, error){
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

	// delay transport on application startup
	go func() {
		td := services.GetTransitService().TransportStartRndDelay
		upSince := services.GetTransitService().Stats().UpSince.Value()
		if td > 0 && time.Since(time.UnixMilli(upSince)).Round(time.Second) < 8 {
			time.Sleep(DefaultCheckInterval + time.Second*time.Duration(rand.Intn(td)))
		}
		_ = services.GetTransitService().StartTransport()
	}()
	return nil
}

// SendMetrics processes metrics payload
func SendMetrics(ctx context.Context, resources []transit.MonitoredResource, groups *[]transit.ResourceGroup) error {
	var (
		b   []byte
		err error
	)
	ctxN, span := tracing.StartTraceSpan(ctx, "connectors", "SendMetrics")
	defer func() {
		tracing.EndTraceSpan(span,
			tracing.TraceAttrError(err),
			tracing.TraceAttrPayloadLen(b),
		)
	}()

	request := transit.ResourcesWithServicesRequest{
		Context:   services.GetTransitService().MakeTracerContext(),
		Resources: resources,
	}
	if groups != nil {
		request.Groups = *groups
	}
	for i := range request.Resources {
		monitoredServices := EvaluateExpressions(request.Resources[i].Services)
		request.Resources[i].Services = monitoredServices
		request.Resources[i].LastPluginOutput = buildHostStatusText(monitoredServices)
	}
	b, err = json.Marshal(request)
	if err != nil {
		return err
	}
	err = services.GetTransitService().SendResourceWithMetrics(ctxN, b)
	return err
}

// SendInventory processes inventory payload
func SendInventory(ctx context.Context, resources []transit.InventoryResource, resourceGroups []transit.ResourceGroup, ownershipType transit.HostOwnershipType) error {
	var (
		b   []byte
		err error
	)
	ctxN, span := tracing.StartTraceSpan(ctx, "connectors", "SendInventory")
	defer func() {
		tracing.EndTraceSpan(span,
			tracing.TraceAttrError(err),
			tracing.TraceAttrPayloadLen(b),
		)
	}()

	request := transit.InventoryRequest{
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
func CreateInventoryService(name string, owner string) transit.InventoryService {
	return transit.InventoryService{
		BaseInfo: transit.BaseInfo{
			Name:  name,
			Type:  transit.ResourceTypeService,
			Owner: owner,
		},
	}
}

// makes and modifies a copy, doesn't modify services
func CreateInventoryResource(name string, services []transit.InventoryService) transit.InventoryResource {
	resource := transit.InventoryResource{
		BaseResource: transit.BaseResource{
			BaseInfo: transit.BaseInfo{
				Name: name,
				Type: transit.ResourceTypeHost,
			},
		},
	}
	resource.Services = append(resource.Services, services...)
	return resource
}

func CreateResourceRef(name string, owner string, resourceType transit.ResourceType) transit.ResourceRef {
	resource := transit.ResourceRef{
		Name:  name,
		Type:  resourceType,
		Owner: owner,
	}
	return resource
}

func CreateResourceGroup(name string, description string, groupType transit.GroupType, resources []transit.ResourceRef) transit.ResourceGroup {
	group := transit.ResourceGroup{
		GroupName:   name,
		Type:        groupType,
		Description: description,
	}
	group.Resources = append(group.Resources, resources...)
	return group
}

func FillGroupWithResources(group transit.ResourceGroup, resources []transit.InventoryResource) transit.ResourceGroup {
	var monitoredResourceRefs = make([]transit.ResourceRef, 0, len(resources))
	for _, resource := range resources {
		monitoredResourceRefs = append(monitoredResourceRefs,
			transit.ResourceRef{
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
	StartTimestamp *transit.Timestamp
	EndTimestamp   *transit.Timestamp
	Graphed        bool
	Tags           map[string]string
}

// BuildMetric creates metric based on data provided with metricBuilder
func BuildMetric(metricBuilder MetricBuilder) (*transit.TimeSeries, error) {
	var args []interface{}
	if metricBuilder.UnitType != nil {
		args = append(args, metricBuilder.UnitType)
	}
	if metricBuilder.StartTimestamp != nil && metricBuilder.EndTimestamp != nil {
		timeInterval := &transit.TimeInterval{
			StartTime: metricBuilder.StartTimestamp,
			EndTime:   metricBuilder.EndTimestamp,
		}
		args = append(args, timeInterval)
	} else if metricBuilder.StartTimestamp != nil || metricBuilder.EndTimestamp != nil {
		log.Error().
			Msgf("could not create time interval for metric %s: either start time or end time is not provided",
				metricBuilder.Name)
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
			log.Err(err).Msgf("could not create warning threshold for metric %s", metricBuilder.Name)
		}
		thresholds = append(thresholds, *warningThreshold)
	}
	if metricBuilder.Critical != nil {
		criticalThreshold, err := CreateCriticalThreshold(metricName+"_cr",
			metricBuilder.Critical)
		if err != nil {
			log.Err(err).Msgf("could not create critical threshold for metric %s", metricBuilder.Name)
		}
		thresholds = append(thresholds, *criticalThreshold)
	}
	if len(thresholds) > 0 {
		metric.Thresholds = thresholds
	}

	return metric, nil
}

// CreateMetric
//
//	required parameters: name, value
//	optional parameters: interval, UnitType
//
// CreateMetric("cpu", 30)              // integer value
// CreateMetric("percentFree", 30.59)   // double value
// CreateMetric("cpu", 30.59, transit.PercentCPU) // with optional Unit
// CreateMetric("cpu", 30.59, interval)           // with optional interval
// CreateMetric("cpu", 30.59, transit.PercentCPU, interval) // with optional Unit and optional interval
// with optional Unit and optional Interval and optional UnitType
// CreateMetric("cpu", 30.59, interval, transit.PercentCPU)
// Thresholds must be set separately
func CreateMetric(name string, value interface{}, args ...interface{}) (*transit.TimeSeries, error) {
	// set the value based on variable type
	typedValue := transit.NewTypedValue(value)
	if typedValue == nil {
		return nil, fmt.Errorf("%w: %T", ErrUnsupportedType, value)
	}
	metric := transit.TimeSeries{
		MetricName: name,
		SampleType: transit.Value,
		Value:      typedValue,
	}
	// optional argument processing
	// var arguments []interface{} = make([]interface{}, len(args))
	for _, arg := range args {
		switch arg := arg.(type) {
		case string:
			metric.Unit = transit.UnitType(arg)
		case transit.UnitType:
			metric.Unit = arg
		case *transit.TimeInterval:
			metric.Interval = arg
		case map[string]string:
			metric.Tags = arg
		//case transit.MetricSampleType:
		//	metric.SampleType = arg.(transit.MetricSampleType)
		default:
			return nil, fmt.Errorf("%w: %T", ErrUnsupportedType, arg)
		}
	}
	// optional interval
	if metric.Interval == nil {
		timestamp := transit.NewTimestamp()
		metric.Interval = &transit.TimeInterval{
			EndTime:   timestamp,
			StartTime: timestamp,
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
	typedValue := transit.NewTypedValue(value)
	if typedValue == nil {
		return nil, fmt.Errorf("%w: %T", ErrUnsupportedType, value)
	}
	// create the threshold
	threshold := transit.ThresholdValue{
		SampleType: thresholdType,
		Label:      label,
		Value:      typedValue,
	}
	return &threshold, nil
}

// BuildServiceForMetric creates metric based on data provided in metric builder and if metric successfully created
// creates service with same name as metric which contains only this one metric
// returns the result of service creation
func BuildServiceForMetric(hostName string, metricBuilder MetricBuilder) (*transit.MonitoredService, error) {
	metric, err := BuildMetric(metricBuilder)
	if err != nil {
		log.Err(err).Msgf("could not create metric for process: %s", metricBuilder.Name)
		return nil, errors.New("cannot create service with metric due to metric creation failure")
	}
	serviceName := Name(metricBuilder.Name, metricBuilder.CustomName)

	serviceProperties := make(map[string]interface{})
	// serviceProperties["isGraphed"] = metricBuilder.Graphed

	return CreateService(serviceName, hostName, []transit.TimeSeries{*metric}, serviceProperties)
}

func BuildServiceForMetricWithStatusText(hostName string, metricBuilder MetricBuilder,
	statusMessages map[transit.MonitorStatus]string) (*transit.MonitoredService, error) {
	service, err := BuildServiceForMetric(hostName, metricBuilder)
	if err != nil {
		log.Err(err).Msgf("could not create service %s:%s", hostName, metricBuilder.CustomName)
		return service, err
	}
	// if no message for status provided in a map patternMessage is "", thresholds will be texted anyway if they exist
	patternMessage := statusMessages[service.Status]
	addServiceStatusText(patternMessage, service)
	return service, err
}

func BuildServiceForMultiMetric(hostName string, serviceName string, customName string, metricBuilders []MetricBuilder) (*transit.MonitoredService, error) {
	metrics := make([]transit.TimeSeries, len(metricBuilders))
	for index, metricBuilder := range metricBuilders {
		metric, err := BuildMetric(metricBuilder)
		if err != nil {
			log.Err(err).Msgf("could not create metric for process: %s", metricBuilder.Name)
			return nil, errors.New("cannot create service with metric due to metric creation failure")
		}
		metrics[index] = *metric
	}
	gwServiceName := Name(serviceName, customName)
	return CreateService(gwServiceName, hostName, metrics)
}

func BuildServiceForMetrics(serviceName string, hostName string, metricBuilders []MetricBuilder) (*transit.MonitoredService, error) {
	var timeSeries = make([]transit.TimeSeries, 0)
	for _, metricBuilder := range metricBuilders {
		metric, err := BuildMetric(metricBuilder)
		if err != nil {
			log.Err(err).Msgf("could not create metric for process %s with metric %s", serviceName, metricBuilder.Name)
			return nil, errors.New("cannot create service with metric due to metric creation failure")
		}
		timeSeries = append(timeSeries, *metric)
	}
	service, err := CreateService(serviceName, hostName, timeSeries)
	if err != nil {
		log.Err(err).Msgf("could not create service %s:%s", hostName, serviceName)
	}
	addServiceStatusText("", service)

	return service, nil
}

// CreateService makes node
// required params: name, owner(resource)
// optional params: metrics
func CreateService(name string, owner string, args ...interface{}) (*transit.MonitoredService, error) {
	lastCheckTime := *transit.NewTimestamp()
	nextCheckTime := lastCheckTime.Add(CheckInterval)
	service := transit.MonitoredService{
		BaseInfo: transit.BaseInfo{
			Name:  name,
			Type:  transit.ResourceTypeService,
			Owner: owner,
		},
		MonitoredInfo: transit.MonitoredInfo{
			Status:        transit.ServiceOk,
			LastCheckTime: &lastCheckTime,
			NextCheckTime: &nextCheckTime,
		},
	}
	for _, arg := range args {
		switch arg := arg.(type) {
		case []transit.TimeSeries:
			service.Metrics = arg
			if len(service.Metrics) > 0 {
				lastCheckTime = *service.Metrics[len(service.Metrics)-1].Interval.EndTime
				nextCheckTime = lastCheckTime.Add(CheckInterval)
				service.LastCheckTime = &lastCheckTime
				service.NextCheckTime = &nextCheckTime
			}
		case map[string]interface{}:
			service.CreateProperties(arg)
		default:
			return nil, fmt.Errorf("%w: %T", ErrUnsupportedType, arg)
		}
	}
	if service.Metrics != nil {
		service.Status, _ = transit.CalculateServiceStatus(&service.Metrics)
	} else {
		service.Status = transit.ServiceUnknown
	}
	return &service, nil
}

// CreateResource makes node
// required params: name
// optional params: services
func CreateResource(name string, args ...interface{}) (*transit.MonitoredResource, error) {
	lastCheckTime := *transit.NewTimestamp()
	nextCheckTime := lastCheckTime.Add(CheckInterval)
	resource := transit.MonitoredResource{
		BaseResource: transit.BaseResource{
			BaseInfo: transit.BaseInfo{
				Name: name,
				Type: transit.ResourceTypeHost,
			},
		},
		MonitoredInfo: transit.MonitoredInfo{
			Status:        transit.HostUp,
			LastCheckTime: &lastCheckTime,
			NextCheckTime: &nextCheckTime,
		},
	}
	for _, arg := range args {
		switch arg := arg.(type) {
		case []transit.MonitoredService:
			resource.Services = arg
			if len(resource.Services) > 0 {
				resource.LastCheckTime = resource.Services[0].LastCheckTime
				resource.NextCheckTime = resource.Services[0].NextCheckTime
			}
		case string:
			resource.Device = arg
		default:
			return nil, fmt.Errorf("%w: %T", ErrUnsupportedType, arg)
		}
	}

	if resource.Services != nil && len(resource.Services) != 0 {
		resource.Status = transit.CalculateResourceStatus(resource.Services)
	} else {
		resource.Status = transit.HostUp
	}

	return &resource, nil
}

// EvaluateExpressions calculates synthetic metrics
func EvaluateExpressions(services []transit.MonitoredService) []transit.MonitoredService {
	var (
		vars   = make(map[string]interface{})
		result = make([]transit.MonitoredService, 0, len(services))
	)

	for _, service := range services {
		for _, metric := range service.Metrics {
			if metric.MetricComputeType != transit.Synthetic {
				switch metric.Value.ValueType {
				case transit.IntegerType:
					vars[strings.ReplaceAll(metric.MetricName, ".", "_")] = float64(*metric.Value.IntegerValue)
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
					log.Err(err).
						Interface("expression", metric.MetricExpression).
						Interface("arguments", vars).
						Msg("could not evaluate expression")
					continue
				} else {
					endTime := metric.Interval.EndTime
					startTime := metric.Interval.StartTime
					lastCheckTime := *endTime
					nextCheckTime := lastCheckTime.Add(CheckInterval)
					result[i] = transit.MonitoredService{
						BaseInfo: transit.BaseInfo{
							Name:  result[i].Name,
							Type:  transit.ResourceTypeService,
							Owner: result[i].Owner,
						},
						MonitoredInfo: transit.MonitoredInfo{
							LastPluginOutput: fmt.Sprintf(" Expression: %s", metric.MetricExpression),
							LastCheckTime:    &lastCheckTime,
							NextCheckTime:    &nextCheckTime,
						},
						Metrics: []transit.TimeSeries{
							{
								MetricName: metric.MetricName,
								SampleType: transit.Value,
								Interval: &transit.TimeInterval{
									EndTime:   endTime,
									StartTime: startTime,
								},
								Thresholds: metric.Thresholds,
								Value:      transit.NewTypedValue(value),
							},
						},
					}
					status, err := transit.CalculateServiceStatus(&result[i].Metrics)
					result[i].Status = status
					if err != nil {
						log.Err(err).Msg("could not calculate service status")
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
var Hashsum = config.Hashsum

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
				log.Err(err.(error)).Msg("recovered error in periodic handler")
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
//
//	Following formatting rules are being applied by default (not considering minRound):
//		if the value is less than 3 minutes it will be formatted to seconds
//		if the value is between 3 minutes and 3 hours it will be formatted to minutes
//		if the value is between 3 hours and 3 days (<73 hours) it will be formatted to hours
//		values higher than 72 hours (3 days) will be formatted to days
//	minRound arg determines time unit "limit" lower than that formatting is not allowed
//	i.e. if the value is for example 120 seconds which is gonna be returned as "120 second(s)" by default rules
//		will be returned as "2 minute(s)" if minRound="minute" (i.e. formatting to lower than minute is not allowed)
//		or "0 hour(s)" if minRound="hour" (i.e. formatting to lower than hour is not allowed)
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

var sanitizeRegexp = regexp.MustCompile(`[^\w-_.:]`)

// SanitizeString replaces all special characters with '_'
func SanitizeString(str string) string {
	str = sanitizeRegexp.ReplaceAllString(str, "_")
	str = removeUnderscoreDuplicates(str)

	if str[len(str)-1:] == "_" {
		str = str[0 : len(str)-1]
	}

	return str
}

func removeUnderscoreDuplicates(s string) string {
	var (
		buf  strings.Builder
		last rune
	)
	for i, r := range s {
		if (r != last || string(r) != "_") || i == 0 {
			buf.WriteRune(r)
			last = r
		}
	}
	return buf.String()
}

func addServiceStatusText(patternMessage string, service *transit.MonitoredService) {
	if service == nil {
		log.Error().Msg("service is nil")
		return
	}
	re := regexp.MustCompile(statusTextPattern)
	patterns := re.FindAllString(patternMessage, -1)
	for _, pattern := range patterns {
		if fn, has := statusTextValGetters[pattern]; has {
			replacement, err := fn(service)
			if err != nil {
				log.Warn().
					Err(err).
					Msgf("could not replace pattern <%s> in message <%s> for service %s",
						pattern, patternMessage, service.Name)
			} else {
				patternMessage = strings.ReplaceAll(patternMessage, pattern, replacement)
			}
		} else {
			log.Warn().Msgf("no method to get service value for pattern: %s", pattern)
		}
	}
	statusText := addThresholdsToStatusText(patternMessage, service)
	service.LastPluginOutput = statusText
}

func addThresholdsToStatusText(statusText string, service *transit.MonitoredService) string {
	if service == nil {
		log.Error().Msg("service is nil")
		return statusText
	}

	for _, metric := range service.Metrics {
		mValText, err := getValueText(metric.Value)
		if err != nil {
			log.Warn().Err(err).Msgf("could not get metric %s value for service %s", metric.MetricName, service.Name)
			continue
		}
		wt := noneThresholdText
		crt := noneThresholdText
		if metric.Thresholds != nil {
			for _, th := range metric.Thresholds {
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
						log.Warn().
							Err(err).
							Msgf("could not get %s threshold for service %s", thresholdType, service.Name)
					}
				}
			}
			metricVal, err := strconv.ParseFloat(mValText, 64)
			if err != nil {
				log.Warn().Err(err).Msgf("could not parse metric value %s for service %s", mValText, service.Name)
				continue
			}

			var thText string
			if noneThresholdText != wt && noneThresholdText != crt {
				wtVal, err := strconv.ParseFloat(wt, 64)
				if err != nil {
					log.Warn().Err(err).
						Msgf("could not parse warning threshold value %s for service %s", wt, service.Name)
					continue
				}
				crtVal, err := strconv.ParseFloat(crt, 64)
				if err != nil {
					log.Warn().Err(err).
						Msgf("could not parse critical threshold value %s for service %s", crt, service.Name)
					continue
				}

				if metricVal >= wtVal && metricVal >= crtVal {
					thText = fmt.Sprintf(" | [%s] [VAL=%s] [W/C=%s/%s]", metric.MetricName, mValText, wt, crt)
				} else {
					if metricVal >= wtVal {
						thText = fmt.Sprintf(" | [%s] [VAL=%s] [Warn=%s]", metric.MetricName, mValText, wt)
					}
					if metricVal >= crtVal {
						thText = fmt.Sprintf(" | [%s] [VAL=%s] [Crit=%s]", metric.MetricName, mValText, crt)
					}
				}
			} else if noneThresholdText != wt {
				wtVal, err := strconv.ParseFloat(wt, 64)
				if err != nil {
					log.Warn().Err(err).
						Msgf("could not parse warning threshold value %s for service %s", wt, service.Name)
					continue
				}

				if metricVal >= wtVal {
					thText = fmt.Sprintf(" | [%s] [VAL=%s] [Warn=%s]", metric.MetricName, mValText, wt)
				}
			} else if noneThresholdText != crt {
				crtVal, err := strconv.ParseFloat(crt, 64)
				if err != nil {
					log.Warn().Err(err).
						Msgf("could not parse critical threshold value %s for service %s", crt, service.Name)
					continue
				}
				if metricVal >= crtVal {
					thText = fmt.Sprintf(" | [%s] [VAL=%s] [Crit=%s]", metric.MetricName, mValText, wt)
				}
			}
			statusText = statusText + thText
		}
	}

	if strings.HasPrefix(statusText, " | ") {
		statusText = statusText[3:]
	}

	return statusText
}

func extractValueForStatusText(service *transit.MonitoredService) (string, error) {
	if len(service.Metrics) == 1 {
		return getValueText(service.Metrics[0].Value)
	}
	log.Warn().Msg("not supported for service with more than one metric")
	return "", errors.New("not supported for service with more than one metric")
}

func getValueText(value *transit.TypedValue) (string, error) {
	if value == nil {
		log.Warn().Msg("no value")
		return "", errors.New("no value")
	}
	switch value.ValueType {
	case transit.IntegerType:
		return strconv.Itoa(int(*value.IntegerValue)), nil
	case transit.DoubleType:
		return fmt.Sprintf("%f", *value.DoubleValue), nil
	case transit.StringType:
		return *value.StringValue, nil
	case transit.BooleanType:
		return strconv.FormatBool(*value.BoolValue), nil
	case transit.TimeType:
		return FormatTimeForStatusMessage(time.Duration(value.TimeValue.UnixNano()), time.Second), nil
	}
	log.Warn().Msg("unknown value type")
	return "", errors.New("unknown value type")
}

func extractIntervalForStatusText(service *transit.MonitoredService) (string, error) {
	return FormatTimeForStatusMessage(CheckInterval, time.Minute), nil
}

func buildHostStatusText(services []transit.MonitoredService) string {
	var ok, warn, critical, other int
	for _, service := range services {
		switch service.Status {
		case transit.ServiceOk:
			ok = ok + 1
		case transit.ServiceWarning:
			warn = warn + 1
		case transit.ServiceScheduledCritical,
			transit.ServiceUnscheduledCritical:
			critical = critical + 1
		default:
			other = other + 1
		}
	}
	return fmt.Sprintf("Host has %d OK, %d WARNING, %d CRITICAL and %d other services.",
		ok, warn, critical, other)
}
