package main

import (
	"errors"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/gwos/tcg/connectors"
	"github.com/gwos/tcg/log"
	"github.com/gwos/tcg/milliseconds"
	"github.com/gwos/tcg/services"
	"github.com/gwos/tcg/transit"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"
)

var (
	re = regexp.MustCompile(`^(.*?);(.*?);(.*?);(.*?);(.*?);(.*?)\|\s(.*?)=(.*?);(.*?);(.*?)$`)
)

func initializeEntrypoints() []services.Entrypoint {
	var entrypoints []services.Entrypoint

	entrypoints = append(entrypoints,
		services.Entrypoint{
			Url:     "/receiver",
			Method:  "Post",
			Handler: receiverHandler,
		},
	)

	return entrypoints
}

func receiverHandler(c *gin.Context) {
	body, err := c.GetRawData()
	if err != nil {
		log.Error(err.Error())
		c.JSON(http.StatusBadRequest, err.Error())
		return
	}

	if err := processMetrics(body); err != nil {
		c.JSON(http.StatusBadRequest, err.Error())
	}
}

func processMetrics(body []byte) error {
	_, err := parseBody(body)
	if err != nil {
		return err
	}

	return nil
}

func parseBody(body []byte) (*[]transit.MonitoredResource, error) {
	metricsLines := strings.Split(string(body), "\n")
	var monitoredResources []transit.MonitoredResource

	serviceNameToMetricsMap, err := getMetrics(metricsLines)
	if err != nil {
		return nil, err
	}

	resourceNameToServicesMap, err := getServices(serviceNameToMetricsMap, metricsLines)
	if err != nil {
		return nil, err
	}

	for key, value := range resourceNameToServicesMap {
		monitoredResources = append(monitoredResources, transit.MonitoredResource{
			Name:             key,
			Type:             transit.Host,
			Owner:            "",
			Status:           connectors.CalculateResourceStatus(value),
			LastCheckTime:    milliseconds.MillisecondTimestamp{Time: time.Now()},
			NextCheckTime:    milliseconds.MillisecondTimestamp{Time: time.Now().Add(connectors.DefaultTimer)},
			LastPlugInOutput: "",
			Properties:       nil,
			Services:         value,
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
	return &milliseconds.MillisecondTimestamp{Time: time.Unix(0, i).UTC()}, nil
}

func getStatus(str string) (transit.MonitorStatus, error) {
	switch str {
	case "0":
		return transit.HostUp, nil
	default:
		return "nil", errors.New("unknown status provided")
	}
}

func getMetrics(metricsLines []string) (map[string][]transit.TimeSeries, error) {
	metricsMap := make(map[string][]transit.TimeSeries)
	for _, metric := range metricsLines {
		arr := re.FindStringSubmatch(metric)[1:]

		if len(arr) != 10 {
			return nil, errors.New("invalid metric format")
		}

		value, err := strconv.ParseFloat(arr[7], 64)
		warning, err := strconv.ParseFloat(arr[8], 64)
		critical, err := strconv.ParseFloat(arr[9], 64)

		timestamp, err := getTime(arr[1])
		if err != nil {
			return nil, err
		}

		timeSeries, err := connectors.BuildMetric(connectors.MetricBuilder{
			Name:           arr[6],
			ComputeType:    transit.Query,
			Value:          value,
			UnitType:       transit.MB,
			Warning:        warning,
			Critical:       critical,
			StartTimestamp: &milliseconds.MillisecondTimestamp{Time: timestamp.Time},
			EndTimestamp:   &milliseconds.MillisecondTimestamp{Time: timestamp.Time},
		})
		if err != nil {
			return nil, err
		}
		metricsMap[fmt.Sprintf("%s:%s", arr[2], arr[3])] = append(metricsMap[fmt.Sprintf("%s:%s", arr[2], arr[3])], *timeSeries)
	}

	return metricsMap, nil
}

func getServices(metricsMap map[string][]transit.TimeSeries, metricsLines []string) (map[string][]transit.MonitoredService, error) {
	servicesMap := make(map[string][]transit.MonitoredService)
	for _, metric := range metricsLines {
		arr := re.FindStringSubmatch(metric)[1:]

		if len(arr) != 10 {
			return nil, errors.New("invalid metric format")
		}

		timestamp, err := getTime(arr[1])
		if err != nil {
			return nil, err
		}

		status, err := getStatus(arr[4])
		if err != nil {
			return nil, err
		}

		servicesMap[arr[2]] = append(servicesMap[arr[2]], transit.MonitoredService{
			Name:             arr[3],
			Type:             transit.Service,
			Owner:            arr[2],
			Status:           status,
			LastCheckTime:    *timestamp,
			NextCheckTime:    milliseconds.MillisecondTimestamp{Time: timestamp.Add(connectors.DefaultTimer)},
			LastPlugInOutput: arr[5],
			Properties:       nil,
			Metrics:          metricsMap[fmt.Sprintf("%s:%s", arr[2], arr[3])],
		})
	}

	return removeDuplicateServices(servicesMap), nil
}

func removeDuplicateServices(servicesMap map[string][]transit.MonitoredService) map[string][]transit.MonitoredService {
	for key, value := range servicesMap {
		keys := make(map[string]bool)
		var list []transit.MonitoredService
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
