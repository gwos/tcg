package parser

import (
	"bytes"
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/gwos/tcg/connectors"
	"github.com/gwos/tcg/milliseconds"
	"github.com/gwos/tcg/transit"
)

var (
	bronxRegexp = regexp.MustCompile(
		`^(?P<type>H|S);(?P<ts>.*?);(?P<resName>.*?);((?P<svcName>.*?);)?(?P<status>.*?);(?P<msg>.*?)\s*\|\s*(?P<perf>.*?)$`)
	nscaRegexp = regexp.MustCompile(
		`^((?P<ts>.*?);)?(?P<resName>.*?);(?P<svcName>.*?);(?P<status>.*?);(?P<msg>.*?)\s*\|\s*(?P<perf>.*?)$`)
	perfDataRegexp = regexp.MustCompile(
		`^(?P<label>.*?)=(?P<val>.*?)(?P<unitType>\D*?);(?P<warn>.*?)(\D*?);(?P<crit>.*?)(\D*?);` +
			`((?P<min>.*?)(\D*?);)?((?P<max>.*?)(\D*?);)?$`)

	ErrInvalidMetricFormat = errors.New("invalid metric format")
	ErrUnknownMetricFormat = errors.New("unknown metric format")
)

// DataFormat describes incoming payload
type DataFormat string

// Data formats of the received body
const (
	Bronx   DataFormat = "bronx"
	NSCA    DataFormat = "nsca"
	NSCAAlt DataFormat = "nsca-alt"
)

type MetricsMap map[string][]transit.TimeSeries
type ServicesMap map[string][]transit.DynamicMonitoredService

func Parse(payload []byte, dataFormat DataFormat) (*[]transit.DynamicMonitoredResource, error) {
	metricsLines := strings.Split(string(bytes.Trim(payload, " \n\r")), "\n")

	var (
		monitoredResources        []transit.DynamicMonitoredResource
		serviceNameToMetricsMap   MetricsMap
		resourceNameToServicesMap ServicesMap
		err                       error
	)

	switch dataFormat {
	case Bronx:
		serviceNameToMetricsMap, err = getBronxMetrics(metricsLines)
	case NSCA, NSCAAlt:
		serviceNameToMetricsMap, err = getNscaMetrics(metricsLines)
	default:
		return nil, ErrUnknownMetricFormat
	}
	if err != nil {
		return nil, err
	}

	switch dataFormat {
	case Bronx:
		resourceNameToServicesMap, err = getBronxServices(serviceNameToMetricsMap, metricsLines)
	case NSCA, NSCAAlt:
		resourceNameToServicesMap, err = getNscaServices(serviceNameToMetricsMap, metricsLines)
	default:
		return nil, ErrUnknownMetricFormat
	}
	if err != nil {
		return nil, err
	}

	for resName, services := range resourceNameToServicesMap {
		res := transit.DynamicMonitoredResource{
			BaseResource: transit.BaseResource{
				BaseTransitData: transit.BaseTransitData{
					Name: resName,
					Type: transit.Host,
				},
			},
			Services: make([]transit.DynamicMonitoredService, 0, len(services)),
		}
		/* filter and apply host-check results */
		resFlag := false
		for _, svc := range services {
			if svc.Name == "" {
				resFlag = true
				res.LastPlugInOutput = svc.LastPlugInOutput
				res.LastCheckTime = svc.LastCheckTime
				res.NextCheckTime = svc.NextCheckTime
				res.Status = svc.Status
				continue
			}
			res.Services = append(res.Services, svc)
		}
		if !resFlag {
			res.LastCheckTime = milliseconds.MillisecondTimestamp{Time: time.Now()}
			res.NextCheckTime = milliseconds.MillisecondTimestamp{Time: time.Now().Add(connectors.CheckInterval)}
			res.Status = transit.CalculateResourceStatus(res.Services)
		}
		monitoredResources = append(monitoredResources, res)
	}

	return &monitoredResources, nil
}

func getTime(str string) (*milliseconds.MillisecondTimestamp, error) {
	i, err := strconv.ParseInt(str, 10, 64)
	if err != nil {
		return nil, err
	}
	return &milliseconds.MillisecondTimestamp{Time: time.Unix(i, 0)}, nil
}

func getStatus(str string) (transit.MonitorStatus, error) {
	switch str {
	case "0":
		return transit.ServiceOk, nil
	case "1":
		return transit.ServiceWarning, nil
	case "2":
		return transit.ServiceUnscheduledCritical, nil
	case "3":
		return transit.ServiceUnknown, nil
	default:
		return "nil", errors.New("unknown status provided")
	}
}

func getNscaMetrics(metricsLines []string) (MetricsMap, error) {
	metricsMap := make(MetricsMap)
	re := nscaRegexp
	for _, metric := range metricsLines {
		match := re.FindStringSubmatch(metric)
		if match == nil {
			return nil, fmt.Errorf("%w: %v", ErrInvalidMetricFormat, "resource")
		}
		timestamp := &milliseconds.MillisecondTimestamp{Time: time.Now()}
		if ts := match[re.SubexpIndex("ts")]; ts != "" {
			if t, err := getTime(ts); err == nil {
				timestamp = t
			}
		}
		resName := match[re.SubexpIndex("resName")]
		svcName := match[re.SubexpIndex("svcName")]
		perfData := match[re.SubexpIndex("perf")]
		for _, metric := range strings.Split(strings.TrimSpace(perfData), " ") {
			var (
				match                    []string
				label, val, warn, crit   string
				value, warning, critical float64
			)
			match = perfDataRegexp.FindStringSubmatch(metric)
			if match == nil {
				return nil, fmt.Errorf("%w: %v", ErrInvalidMetricFormat, "perf data")
			}
			label = match[perfDataRegexp.SubexpIndex("label")]
			val = match[perfDataRegexp.SubexpIndex("val")]
			warn = match[perfDataRegexp.SubexpIndex("warn")]
			crit = match[perfDataRegexp.SubexpIndex("crit")]

			if len(val) > 0 {
				if v, err := strconv.ParseFloat(val, 64); err == nil {
					value = v
				} else {
					return nil, err
				}
			}
			if len(warn) > 0 {
				if w, err := strconv.ParseFloat(warn, 64); err == nil {
					warning = w
				} else {
					return nil, err
				}
			}
			if len(crit) > 0 {
				if c, err := strconv.ParseFloat(crit, 64); err == nil {
					critical = c
				} else {
					return nil, err
				}
			}

			timeSeries, err := connectors.BuildMetric(connectors.MetricBuilder{
				Name:           label,
				ComputeType:    transit.Query,
				Value:          value,
				UnitType:       transit.MB,
				Warning:        warning,
				Critical:       critical,
				StartTimestamp: timestamp,
				EndTimestamp:   timestamp,
			})
			if err != nil {
				return nil, err
			}
			metricsMap[fmt.Sprintf("%s:%s", resName, svcName)] =
				append(metricsMap[fmt.Sprintf("%s:%s", resName, svcName)], *timeSeries)
		}
	}
	return metricsMap, nil
}

func getNscaServices(metricsMap MetricsMap, metricsLines []string) (ServicesMap, error) {
	servicesMap := make(ServicesMap)
	re := nscaRegexp
	for _, metric := range metricsLines {
		match := re.FindStringSubmatch(metric)
		if match == nil {
			return nil, fmt.Errorf("%w: %v", ErrInvalidMetricFormat, "service")
		}
		timestamp := &milliseconds.MillisecondTimestamp{Time: time.Now()}
		if ts := match[re.SubexpIndex("ts")]; ts != "" {
			if t, err := getTime(ts); err == nil {
				timestamp = t
			}
		}
		status, err := getStatus(match[re.SubexpIndex("status")])
		if err != nil {
			return nil, err
		}
		resName := match[re.SubexpIndex("resName")]
		svcName := match[re.SubexpIndex("svcName")]
		msg := match[re.SubexpIndex("msg")]
		servicesMap[resName] = append(servicesMap[resName], transit.DynamicMonitoredService{
			BaseTransitData: transit.BaseTransitData{
				Name:  svcName,
				Type:  transit.Service,
				Owner: resName,
			},
			Status:           status,
			LastCheckTime:    *timestamp,
			NextCheckTime:    milliseconds.MillisecondTimestamp{Time: timestamp.Add(connectors.CheckInterval)},
			LastPlugInOutput: msg,
			Metrics:          metricsMap[fmt.Sprintf("%s:%s", resName, svcName)],
		})
	}
	return removeDuplicateServices(servicesMap), nil
}

func getBronxMetrics(metricsLines []string) (MetricsMap, error) {
	metricsMap := make(MetricsMap)
	re := bronxRegexp
	for _, metric := range metricsLines {
		match := re.FindStringSubmatch(metric)
		if match == nil {
			return nil, fmt.Errorf("%w: %v", ErrInvalidMetricFormat, "resource")
		}
		timestamp, err := getTime(match[re.SubexpIndex("ts")])
		if err != nil {
			return nil, err
		}
		resName := match[re.SubexpIndex("resName")]
		svcName := match[re.SubexpIndex("svcName")]
		perfData := match[re.SubexpIndex("perf")]
		for _, metric := range strings.Split(strings.TrimSpace(perfData), " ") {
			var (
				match                    []string
				label, val, warn, crit   string
				value, warning, critical float64
			)
			match = perfDataRegexp.FindStringSubmatch(metric)
			if match == nil {
				return nil, fmt.Errorf("%w: %v", ErrInvalidMetricFormat, "perf data")
			}
			label = match[perfDataRegexp.SubexpIndex("label")]
			val = match[perfDataRegexp.SubexpIndex("val")]
			warn = match[perfDataRegexp.SubexpIndex("warn")]
			crit = match[perfDataRegexp.SubexpIndex("crit")]

			if len(val) > 0 {
				if v, err := strconv.ParseFloat(val, 64); err == nil {
					value = v
				} else {
					return nil, err
				}
			}
			if len(warn) > 0 {
				if w, err := strconv.ParseFloat(warn, 64); err == nil {
					warning = w
				} else {
					return nil, err
				}
			}
			if len(crit) > 0 {
				if c, err := strconv.ParseFloat(crit, 64); err == nil {
					critical = c
				} else {
					return nil, err
				}
			}

			timeSeries, err := connectors.BuildMetric(connectors.MetricBuilder{
				Name:           label,
				ComputeType:    transit.Query,
				Value:          value,
				UnitType:       transit.MB,
				Warning:        warning,
				Critical:       critical,
				StartTimestamp: timestamp,
				EndTimestamp:   timestamp,
			})
			if err != nil {
				return nil, err
			}
			metricsMap[fmt.Sprintf("%s:%s", resName, svcName)] =
				append(metricsMap[fmt.Sprintf("%s:%s", resName, svcName)], *timeSeries)
		}
	}
	return metricsMap, nil
}

func getBronxServices(metricsMap MetricsMap, metricsLines []string) (ServicesMap, error) {
	servicesMap := make(ServicesMap)
	re := bronxRegexp
	for _, metric := range metricsLines {
		match := re.FindStringSubmatch(metric)
		if match == nil {
			return nil, fmt.Errorf("%w: %v", ErrInvalidMetricFormat, "service")
		}
		timestamp, err := getTime(match[re.SubexpIndex("ts")])
		if err != nil {
			return nil, err
		}
		status, err := getStatus(match[re.SubexpIndex("status")])
		if err != nil {
			return nil, err
		}
		resName := match[re.SubexpIndex("resName")]
		svcName := match[re.SubexpIndex("svcName")]
		msg := match[re.SubexpIndex("msg")]
		servicesMap[resName] = append(servicesMap[resName], transit.DynamicMonitoredService{
			BaseTransitData: transit.BaseTransitData{
				Name:  svcName,
				Type:  transit.Service,
				Owner: resName,
			},
			Status:           status,
			LastCheckTime:    *timestamp,
			NextCheckTime:    milliseconds.MillisecondTimestamp{Time: timestamp.Add(connectors.CheckInterval)},
			LastPlugInOutput: msg,
			Metrics:          metricsMap[fmt.Sprintf("%s:%s", resName, svcName)],
		})
	}
	return removeDuplicateServices(servicesMap), nil
}

func removeDuplicateServices(servicesMap ServicesMap) ServicesMap {
	for key, value := range servicesMap {
		keys := make(map[string]bool)
		var list []transit.DynamicMonitoredService
		for _, entry := range value {
			if _, value := keys[entry.Name]; !value {
				keys[entry.Name] = true
				list = append(list, entry)
			}
		}
		servicesMap[key] = list
	}
	return servicesMap
}
