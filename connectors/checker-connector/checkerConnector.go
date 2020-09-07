package main

import (
	"bytes"
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
	bronxRegexp              = regexp.MustCompile(`^(.*?);(.*?);(.*?);(.*?);(.*?);(.*?)\|\s(.*?)=(.*?);(.*?);(.*?)$`)
	nscaRegexp               = regexp.MustCompile(`^(?:(.*?);)?(.*?);(.*?);(.*?);(.*?)\|(.*?)$`)
	perfDataRegexp           = regexp.MustCompile(`^(.*?)=(.*?);(.*?);(.*?);$`)
	perfDataWithMinRegexp    = regexp.MustCompile(`^(.*?)=(.*?);(.*?);(.*?);(.*?);$`)
	perfDataWithMinMaxRegexp = regexp.MustCompile(`^(.*?)=(.*?);(.*?);(.*?);(.*?);(.*?);$`)
)

type DataFormat string

// Data formats of the received body
const (
	NSCA  DataFormat = "NSCA"
	Bronx            = "Bronx"
)

// ScheduleTask defines command
type ScheduleTask struct {
	CombinedOutput bool     `json:"combinedOutput,omitempty"`
	Command        []string `json:"command"`
	Cron           string   `json:"cron"`
	Environment    []string `json:"environment,omitempty"`
}

// ExtConfig defines the MonitorConnection extensions configuration
type ExtConfig struct {
	Schedule []ScheduleTask `json:"schedule"`
}

// Validate validates value
func (cfg ExtConfig) Validate() error {
	for _, task := range cfg.Schedule {
		if len(task.Command) == 0 {
			return fmt.Errorf("ExtConfig Schedule item error: Command is empty")
		}
	}
	return nil
}

func initializeEntrypoints() []services.Entrypoint {
	var entrypoints []services.Entrypoint
  
	entrypoints = append(entrypoints,
		services.Entrypoint{
			URL:     "/bronx",
			Method:  "Post",
			Handler: bronxHandler,
		},
		services.Entrypoint{
			URL:     "/nsca",
			Method:  "Post",
			Handler: nscaHandler,
		},
	)

	return entrypoints
}

func bronxHandler(c *gin.Context) {
	body, err := c.GetRawData()
	if err != nil {
		log.Error(err.Error())
		c.JSON(http.StatusBadRequest, err.Error())
		return
	}

	if err := processMetrics(body, Bronx); err != nil {
		c.JSON(http.StatusBadRequest, err.Error())
	}
}

func nscaHandler(c *gin.Context) {
	body, err := c.GetRawData()
	if err != nil {
		log.Error(err.Error())
		c.JSON(http.StatusBadRequest, err.Error())
		return
	}

	if err := processMetrics(body, NSCA); err != nil {
		c.JSON(http.StatusBadRequest, err.Error())
	}
}

func processMetrics(body []byte, format DataFormat) error {
	monitoredResources, err := parseBody(body, format)
	if err != nil {
		return err
	}

	inventoryResources := connectors.BuildInventory(monitoredResources)
	if connectors.ValidateInventory(*inventoryResources) {
		err := connectors.SendMetrics(*monitoredResources)
		if err != nil {
			return err
		}
	} else {
		err := connectors.SendInventory(*inventoryResources, nil, transit.Yield)
		if err != nil {
			return err
		}
		time.Sleep(2 * time.Second)
		err = connectors.SendMetrics(*monitoredResources)
		if err != nil {
			return err
		}
	}

	return nil
}

func parseBody(body []byte, format DataFormat) (*[]transit.MonitoredResource, error) {
	metricsLines := strings.Split(string(bytes.Trim(body, " \n\r")), "\n")

	var (
		monitoredResources        []transit.MonitoredResource
		serviceNameToMetricsMap   map[string][]transit.TimeSeries
		resourceNameToServicesMap map[string][]transit.MonitoredService
		err                       error
	)

	switch format {
	case Bronx:
		serviceNameToMetricsMap, err = getBronxMetrics(metricsLines)
	case NSCA:
		serviceNameToMetricsMap, err = getNscaMetrics(metricsLines)
	default:
		return nil, errors.New("unknown data format provided")
	}
	if err != nil {
		return nil, err
	}

	switch format {
	case Bronx:
		resourceNameToServicesMap, err = getBronxServices(serviceNameToMetricsMap, metricsLines)
	case NSCA:
		resourceNameToServicesMap, err = getNscaServices(serviceNameToMetricsMap, metricsLines)
	default:
		return nil, errors.New("unknown data format provided")
	}
	if err != nil {
		return nil, err
	}

	for key, value := range resourceNameToServicesMap {
		monitoredResources = append(monitoredResources, transit.MonitoredResource{
			Name:          key,
			Type:          transit.Host,
			Status:        connectors.CalculateResourceStatus(value),
			LastCheckTime: milliseconds.MillisecondTimestamp{Time: time.Now()},
			NextCheckTime: milliseconds.MillisecondTimestamp{Time: time.Now().Add(connectors.DefaultCheckInterval)},
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
	return &milliseconds.MillisecondTimestamp{Time: time.Unix(0, i).UTC()}, nil
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

func getNscaMetrics(metricsLines []string) (map[string][]transit.TimeSeries, error) {
	metricsMap := make(map[string][]transit.TimeSeries)
	for _, metric := range metricsLines {
		arr := nscaRegexp.FindStringSubmatch(metric)[1:]

		var timestamp = &milliseconds.MillisecondTimestamp{Time: time.Now()}
		var err error
		if len(arr) > 5 && arr[0] == "" {
			arr = arr[1:]
		} else {
			timestamp, err = getTime(arr[0])
			if err != nil {
				return nil, err
			}
		}

		perfData := arr[len(arr)-1]
		pdArr := strings.Split(strings.TrimSpace(perfData), " ")

		for _, metric := range pdArr {
			var values []string
			switch len(strings.Split(metric, ";")) {
			case 4:
				values = perfDataRegexp.FindStringSubmatch(metric)[1:]
			case 5:
				values = perfDataWithMinRegexp.FindStringSubmatch(metric)[1:]
			case 6:
				values = perfDataWithMinMaxRegexp.FindStringSubmatch(metric)[1:]
			}
			if values == nil || len(values) < 4 {
				return nil, errors.New("invalid metric format")
			}
			var value, warning, critical float64
			if v, err := strconv.ParseFloat(values[1], 64); err == nil {
				value = v
			} else {
				return nil, err
			}
			if w, err := strconv.ParseFloat(values[2], 64); err == nil {
				warning = w
			} else {
				return nil, err
			}
			if c, err := strconv.ParseFloat(values[3], 64); err == nil {
				critical = c
			} else {
				return nil, err
			}

			timeSeries, err := connectors.BuildMetric(connectors.MetricBuilder{
				Name:           values[0],
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
			metricsMap[fmt.Sprintf("%s:%s", arr[len(arr)-5], arr[len(arr)-4])] =
				append(metricsMap[fmt.Sprintf("%s:%s", arr[len(arr)-5], arr[len(arr)-4])], *timeSeries)
		}
	}

	return metricsMap, nil
}

func getNscaServices(metricsMap map[string][]transit.TimeSeries, metricsLines []string) (map[string][]transit.MonitoredService, error) {
	servicesMap := make(map[string][]transit.MonitoredService)

	for _, metric := range metricsLines {
		arr := nscaRegexp.FindStringSubmatch(metric)[1:]
		var timestamp = &milliseconds.MillisecondTimestamp{Time: time.Now()}
		var err error
		if len(arr) > 5 && arr[0] == "" {
			arr = arr[1:]
		} else {
			timestamp, err = getTime(arr[0])
			if err != nil {
				return nil, err
			}
		}

		status, err := getStatus(arr[len(arr)-3])
		if err != nil {
			return nil, err
		}

		servicesMap[arr[len(arr)-5]] = append(servicesMap[arr[len(arr)-5]], transit.MonitoredService{
			Name:             arr[len(arr)-4],
			Type:             transit.Service,
			Owner:            arr[len(arr)-5],
			Status:           status,
			LastCheckTime:    *timestamp,
			NextCheckTime:    milliseconds.MillisecondTimestamp{Time: timestamp.Add(connectors.CheckInterval)},
			LastPlugInOutput: arr[len(arr)-2],
			Metrics:          metricsMap[fmt.Sprintf("%s:%s", arr[len(arr)-5], arr[len(arr)-4])],
		})
	}

	return removeDuplicateServices(servicesMap), nil
}

func getBronxMetrics(metricsLines []string) (map[string][]transit.TimeSeries, error) {
	metricsMap := make(map[string][]transit.TimeSeries)
	for _, metric := range metricsLines {
		arr := bronxRegexp.FindStringSubmatch(metric)[1:]

		if len(arr) != 10 {
			return nil, errors.New("invalid metric format")
		}

		var value, warning, critical float64
		if v, err := strconv.ParseFloat(arr[7], 64); err == nil {
			value = v
		} else {
			return nil, err
		}
		if w, err := strconv.ParseFloat(arr[8], 64); err == nil {
			warning = w
		} else {
			return nil, err
		}
		if c, err := strconv.ParseFloat(arr[9], 64); err == nil {
			critical = c
		} else {
			return nil, err
		}

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

func getBronxServices(metricsMap map[string][]transit.TimeSeries, metricsLines []string) (map[string][]transit.MonitoredService, error) {
	servicesMap := make(map[string][]transit.MonitoredService)
	for _, metric := range metricsLines {
		arr := bronxRegexp.FindStringSubmatch(metric)[1:]

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
			NextCheckTime:    milliseconds.MillisecondTimestamp{Time: timestamp.Add(connectors.DefaultCheckInterval)},
			LastPlugInOutput: arr[5],
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
