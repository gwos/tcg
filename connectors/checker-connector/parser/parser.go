package parser

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/gwos/tcg/connectors"
	"github.com/gwos/tcg/milliseconds"
	"github.com/gwos/tcg/services"
	"github.com/gwos/tcg/transit"
)

var (
	bronxRegexp = regexp.MustCompile(
		`^(?P<type>H|S);(?P<ts>.*?);(?P<resName>.*?);((?P<svcName>.*?);)?(?P<status>.*?);(?P<msg>.*?)\s*\|\s*(?P<perf>.*?)$`)
	nscaRegexp = regexp.MustCompile(
		`^((?P<ts>.*?);)?(?P<resName>.*?);(?P<svcName>.*?);(?P<status>.*?);(?P<msg>.*?)\s*\|\s*(?P<perf>.*?)$`)
	perfDataRegexp = regexp.MustCompile(
		`^(?P<label>.*?)=(?P<val>.*?)(?P<unitType>\D*?);(?P<warn>.*?)(\D*?);(?P<crit>.*?)(\D*?);$`)
	perfDataWithMinRegexp = regexp.MustCompile(
		`^(?P<label>.*?)=(?P<val>.*?)(?P<unitType>\D*?);(?P<warn>.*?)(\D*?);(?P<crit>.*?)(\D*?);(?P<min>.*?)(\D*?);$`)
	perfDataWithMinMaxRegexp = regexp.MustCompile(
		`^(?P<label>.*?)=(?P<val>.*?)(?P<unitType>\D*?);(?P<warn>.*?)(\D*?);(?P<crit>.*?)(\D*?);(?P<min>.*?)(\D*?);(?P<max>.*?)(\D*?);$`)
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

func ProcessMetrics(ctx context.Context, payload []byte, dataFormat DataFormat) (*[]transit.DynamicMonitoredResource, error) {
	var (
		ctxN               context.Context
		err                error
		monitoredResources *[]transit.DynamicMonitoredResource
		span               services.TraceSpan
	)

	ctxN, span = services.StartTraceSpan(ctx, "connectors", "parseBody")
	monitoredResources, err = parse(payload, dataFormat)

	services.EndTraceSpan(span,
		services.TraceAttrError(err),
		services.TraceAttrPayloadLen(payload),
	)

	if err != nil {
		return nil, err
	}

	if err := connectors.SendMetrics(ctxN, *monitoredResources, nil); err != nil {
		return nil, err
	}

	return monitoredResources, nil
}

func parse(payload []byte, dataFormat DataFormat) (*[]transit.DynamicMonitoredResource, error) {
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
		return nil, errors.New("unknown data format provided")
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
		return nil, errors.New("unknown data format provided")
	}
	if err != nil {
		return nil, err
	}

	for key, value := range resourceNameToServicesMap {
		monitoredResources = append(monitoredResources, transit.DynamicMonitoredResource{
			BaseResource: transit.BaseResource{
				BaseTransitData: transit.BaseTransitData{
					Name: key,
					Type: transit.Host,
				},
			},
			Status:        connectors.CalculateResourceStatus(value),
			LastCheckTime: milliseconds.MillisecondTimestamp{Time: time.Now()},
			NextCheckTime: milliseconds.MillisecondTimestamp{Time: time.Now().Add(connectors.CheckInterval)},
			Services:      value,
		})
	}

	return &monitoredResources, nil
}

func getTime(str string) (*milliseconds.MillisecondTimestamp, error) {
	i, err := strconv.ParseInt(str, 10, 64)
	if err != nil {
		return nil, err
	}

	i *= int64(time.Millisecond)
	return &milliseconds.MillisecondTimestamp{Time: time.Unix(0, i)}, nil
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
		if len(match) < 5 {
			return nil, errors.New("invalid metric format")
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
		pdArr := strings.Split(strings.TrimSpace(perfData), " ")
		for _, metric := range pdArr {
			var values []string
			var label, val, warn, crit string
			switch len(strings.Split(metric, ";")) {
			case 0, 1, 2, 3:
				return nil, errors.New("invalid metric format")
			case 4:
				values = perfDataRegexp.FindStringSubmatch(metric)
				label = values[perfDataRegexp.SubexpIndex("label")]
				val = values[perfDataRegexp.SubexpIndex("val")]
				warn = values[perfDataRegexp.SubexpIndex("warn")]
				crit = values[perfDataRegexp.SubexpIndex("crit")]
			case 5:
				values = perfDataWithMinRegexp.FindStringSubmatch(metric)
				label = values[perfDataWithMinRegexp.SubexpIndex("label")]
				val = values[perfDataWithMinRegexp.SubexpIndex("val")]
				warn = values[perfDataWithMinRegexp.SubexpIndex("warn")]
				crit = values[perfDataWithMinRegexp.SubexpIndex("crit")]
			case 6:
				values = perfDataWithMinMaxRegexp.FindStringSubmatch(metric)
				label = values[perfDataWithMinMaxRegexp.SubexpIndex("label")]
				val = values[perfDataWithMinMaxRegexp.SubexpIndex("val")]
				warn = values[perfDataWithMinMaxRegexp.SubexpIndex("warn")]
				crit = values[perfDataWithMinMaxRegexp.SubexpIndex("crit")]
			}

			var value, warning, critical float64
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
		if len(match) < 5 {
			return nil, errors.New("invalid metric format")
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
		if len(match) != 8 {
			return nil, errors.New("invalid metric format")
		}
		timestamp, err := getTime(match[re.SubexpIndex("ts")])
		if err != nil {
			return nil, err
		}
		resName := match[re.SubexpIndex("resName")]
		svcName := match[re.SubexpIndex("svcName")]
		perfData := match[re.SubexpIndex("perf")]
		pdArr := strings.Split(strings.TrimSpace(perfData), " ")
		for _, metric := range pdArr {
			var values []string
			var label, val, warn, crit string
			switch len(strings.Split(metric, ";")) {
			case 0, 1, 2, 3:
				return nil, errors.New("invalid metric format")
			case 4:
				values = perfDataRegexp.FindStringSubmatch(metric)
				label = values[perfDataRegexp.SubexpIndex("label")]
				val = values[perfDataRegexp.SubexpIndex("val")]
				warn = values[perfDataRegexp.SubexpIndex("warn")]
				crit = values[perfDataRegexp.SubexpIndex("crit")]
			case 5:
				values = perfDataWithMinRegexp.FindStringSubmatch(metric)
				label = values[perfDataWithMinRegexp.SubexpIndex("label")]
				val = values[perfDataWithMinRegexp.SubexpIndex("val")]
				warn = values[perfDataWithMinRegexp.SubexpIndex("warn")]
				crit = values[perfDataWithMinRegexp.SubexpIndex("crit")]
			case 6:
				values = perfDataWithMinMaxRegexp.FindStringSubmatch(metric)
				label = values[perfDataWithMinMaxRegexp.SubexpIndex("label")]
				val = values[perfDataWithMinMaxRegexp.SubexpIndex("val")]
				warn = values[perfDataWithMinMaxRegexp.SubexpIndex("warn")]
				crit = values[perfDataWithMinMaxRegexp.SubexpIndex("crit")]
			}

			var value, warning, critical float64
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
		if len(match) != 8 {
			return nil, errors.New("invalid metric format")
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
