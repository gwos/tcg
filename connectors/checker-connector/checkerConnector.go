package main

import (
	"errors"
	"github.com/gin-gonic/gin"
	"github.com/gwos/tcg/connectors"
	"github.com/gwos/tcg/log"
	"github.com/gwos/tcg/milliseconds"
	"github.com/gwos/tcg/services"
	"github.com/gwos/tcg/transit"
	"net/http"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

var (
	re = regexp.MustCompile("[;|=]+")
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
	if _, err := parseBody(body); err == nil {
		// fmt.Println(monitoredResource)
		return nil
	} else {
		return err
	}
}

// Every new line - new metric
// Pattern: {???};{timestamp};{host-name};{service-name};{status};{message}| {metric-name}={value};{warning};{critical}
// TODO: check for matching pattern
func parseBody(body []byte) (*[]transit.MonitoredResource, error) {
	metricsLines := strings.Split(string(body), "\n")
	sort.Strings(metricsLines)
	var monitoredResources []transit.MonitoredResource

	//resourceToServicesMap := make(map[string]map[string]transit.TimeSeries)
	serviceNameToMetricsMap := make(map[string][]transit.TimeSeries)

	for _, metric := range metricsLines {
		arr := re.Split(metric, -1)

		value, err := strconv.ParseFloat(arr[7], 64)
		warning, err := strconv.ParseFloat(arr[8], 64)
		critical, err := strconv.ParseFloat(arr[9], 64)

		timeSeries, err := connectors.BuildMetric(connectors.MetricBuilder{
			Name:           arr[6],
			ComputeType:    transit.Query,
			Value:          value,
			UnitType:       transit.MB,
			Warning:        warning,
			Critical:       critical,
			StartTimestamp: nil,
			EndTimestamp:   nil,
		})
		if err != nil {
			return nil, err
		}
		serviceNameToMetricsMap[arr[3]] = append(serviceNameToMetricsMap[arr[3]], *timeSeries)
	}

	for _, metric := range metricsLines {
		arr := re.Split(metric, -1)

		timestamp, err := getTime(arr[1])
		if err != nil {
			return nil, err
		}

		status, err := getStatus(arr[4])
		if err != nil {
			return nil, err
		}

		monitoredResources = append(monitoredResources, transit.MonitoredResource{
			Name:             arr[2],
			Type:             transit.Host,
			Owner:            "",
			Status:           status,
			LastCheckTime:    *timestamp,
			NextCheckTime:    milliseconds.MillisecondTimestamp{Time: timestamp.Add(connectors.DefaultTimer)},
			LastPlugInOutput: arr[5],
			Properties:       nil,
			Services:         nil,
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

func getType(str string) (transit.ResourceType, error) {
	switch str {
	case "H":
		return transit.Host, nil
	case "S":
		return transit.Service, nil
	default:
		return "nil", errors.New("unknown type provided")
	}
}

func getStatus(str string) (transit.MonitorStatus, error) {
	switch str {
	case "0":
		return transit.HostUp, nil
	default:
		return "nil", errors.New("unknown status provided")
	}
}
