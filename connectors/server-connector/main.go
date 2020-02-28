package main

import (
	"encoding/json"
	_ "github.com/gwos/tng/docs"
	"github.com/gwos/tng/log"
	"github.com/gwos/tng/milliseconds"
	"github.com/gwos/tng/services"
	"github.com/gwos/tng/transit"
	"math/rand"
	"os/exec"
	"time"
)

var transitService = services.GetTransitService()

// Default values for 'Group' and loop 'Timer'
const (
	DefaultHostGroupName = "LocalServer"
	DefaultTimer         = 120
)

// @title TNG API Documentation
// @version 1.0

// @host localhost:8099
// @BasePath /api/v1
func main() {
	isConfig := false
	transitService.ConfigHandler = func(data []byte) {
		log.Debug("#ConfigHandler: ", string(data))
		isConfig = true
	}
	for {
		if isConfig {
			break
		}
	}

	err := transitService.StartNats()
	if err != nil {
		log.Error(err.Error())
		return
	}
	err = transitService.StartTransport()
	if err != nil {
		log.Error(err.Error())
		return
	}
	err = transitService.StartController()
	if err != nil {
		log.Error(err.Error())
		return
	}
	defer func() {
		err = transitService.StopNats()
		if err != nil {
			log.Error(err.Error())
		}
		err = transitService.StopTransport()
		if err != nil {
			log.Error(err.Error())
		}
		cmd := exec.Command("rm", "-rf", "src")
		_, err = cmd.Output()
		if err != nil {
			log.Error(err.Error())
		}
		err = transitService.StopController()
		if err != nil {
			log.Error(err.Error())
		}
	}()

	processes, groups, timer, err := getConfig()
	if err != nil {
		log.Error(err)
		return
	}

	for {
		if transitService.Status().Transport != services.Stopped {
			log.Info("TNG ServerConnector: sending inventory ...")
			err = sendInventoryResources(*Synchronize(processes), groups)
		} else {
			log.Info("TNG ServerConnector is stopped ...")
		}
		for i := 0; i < 10; i++ {
			if transitService.Status().Transport != services.Stopped {
				log.Info("TNG ServerConnector: monitoring resources ...")
				err := sendMonitoredResources(*CollectMetrics(processes))
				if err != nil {
					log.Error(err.Error())
				}
			}
			LastCheck = milliseconds.MillisecondTimestamp{Time: time.Now()}
			time.Sleep(time.Duration(int64(timer) * int64(time.Second)))
		}
	}
}

func sendInventoryResources(resource transit.InventoryResource, resourceGroups []transit.ResourceGroup) error {

	monitoredResourceRef := transit.MonitoredResourceRef{
		Name: resource.Name,
		Type: transit.Host,
	}

	for i := range resourceGroups {
		resourceGroups[i].Resources = append(resourceGroups[i].Resources, monitoredResourceRef)
	}

	inventoryRequest := transit.InventoryRequest{
		Context:   transitService.MakeTracerContext(),
		Resources: []transit.InventoryResource{resource},
		Groups:    resourceGroups,
	}

	b, err := json.Marshal(inventoryRequest)
	if err != nil {
		return err
	}

	err = transitService.SynchronizeInventory(b)

	return err
}

func sendMonitoredResources(resource transit.MonitoredResource) error {
	request := transit.ResourcesWithServicesRequest{
		Context:   transitService.MakeTracerContext(),
		Resources: []transit.MonitoredResource{resource},
	}
	// Test a Time type sample
	timeSample := transit.TimeSeries{
		MetricName: "timeSample",
		SampleType: transit.Value,
		Interval: &transit.TimeInterval{
			EndTime:   milliseconds.MillisecondTimestamp{Time: time.Now()},
			StartTime: milliseconds.MillisecondTimestamp{Time: time.Now()},
		},
		Value: &transit.TypedValue{
			ValueType: transit.TimeType,
			TimeValue: &milliseconds.MillisecondTimestamp{Time: time.Now()},
		},
	}
	timeSampleService := transit.MonitoredService{
		Name:             "timeSample",
		Type:             transit.Service,
		Status:           transit.ServiceOk,
		LastCheckTime:    milliseconds.MillisecondTimestamp{Time: time.Now()},
		NextCheckTime:    milliseconds.MillisecondTimestamp{Time: time.Now()},
		LastPlugInOutput: "test",
		Owner:            request.Resources[0].Name, // set host
		Metrics:          []transit.TimeSeries{timeSample},
	}
	request.Resources[0].Services = append(request.Resources[0].Services, timeSampleService)
	b, err := json.Marshal(request)
	if err != nil {
		return err
	}

	return transitService.SendResourceWithMetrics(b)
}

func getConfig() ([]string, []transit.ResourceGroup, int, error) {
	if res, clErr := transitService.DSClient.FetchConnector(transitService.AgentID); clErr == nil {
		var connector = struct {
			Connection transit.MonitorConnection `json:"monitorConnection"`
		}{}
		err := json.Unmarshal(res, &connector)
		if err != nil {
			return []string{}, []transit.ResourceGroup{}, -1, err
		}
		timer := float64(DefaultTimer)
		if _, present := connector.Connection.Extensions["timer"]; present {
			timer = connector.Connection.Extensions["timer"].(float64)
		}
		var processes []string
		if _, present := connector.Connection.Extensions["processes"]; present {
			processesInterface := connector.Connection.Extensions["processes"].([]interface{})
			for _, process := range processesInterface {
				processes = append(processes, process.(string))
			}
		}
		var groups []transit.ResourceGroup
		if _, present := connector.Connection.Extensions["groups"]; present {
			groupsInterface := connector.Connection.Extensions["groups"].([]interface{})
			for _, gr := range groupsInterface {
				groupMap := gr.(map[string]interface{})
				groups = append(groups, transit.ResourceGroup{GroupName: groupMap["name"].(string), Type: transit.GroupType(groupMap["type"].(string))})
			}
		} else {
			groups = append(groups, transit.ResourceGroup{GroupName: DefaultHostGroupName, Type: transit.HostGroup})
		}

		return processes, groups, int(timer), nil
	} else {
		return []string{}, []transit.ResourceGroup{}, -1, clErr
	}
}

//////////////////////////////////////////////////////////
func example() {
	warningThreshold := transit.ThresholdValue{
		SampleType: transit.Warning,
		Label:      "local_load_5_wn",
		Value:      &transit.TypedValue{ValueType: transit.DoubleType, DoubleValue: 70.0}}
	errorThreshold := transit.ThresholdValue{
		SampleType: transit.Critical,
		Label:      "local_load_5_cr",
		Value:      &transit.TypedValue{ValueType: transit.DoubleType, DoubleValue: 85.0}}
	random := rand.Float64() * 100.0
	now := milliseconds.MillisecondTimestamp{Time: time.Now()}
	sampleValue := transit.TimeSeries{
		MetricName: "local_load_5",
		SampleType: transit.Value,
		Interval:   &transit.TimeInterval{EndTime: now, StartTime: now},
		Value:      &transit.TypedValue{ValueType: transit.DoubleType, DoubleValue: random},
		Thresholds: &[]transit.ThresholdValue{warningThreshold, errorThreshold},
		Unit:       "%{cpu}",
	}

	// Example Service
	var localLoadService = transit.MonitoredService{
		Name:             "local_load",
		Type:             transit.Service,
		Status:           transit.ServiceOk,
		LastCheckTime:    milliseconds.MillisecondTimestamp{Time: time.Now()},
		NextCheckTime:    milliseconds.MillisecondTimestamp{Time: time.Now().Add(time.Minute * 5)},
		LastPlugInOutput: "foo | bar",
		Properties: map[string]transit.TypedValue{
			"stateType":       {StringValue: "SOFT"},
			"checkType":       {StringValue: "ACTIVE"},
			"PerformanceData": {StringValue: "007-321 RAD"},
			"ExecutionTime":   {DoubleValue: 3.0},
			"CurrentAttempt":  {IntegerValue: 2},
			"InceptionTime":   {TimeValue: &milliseconds.MillisecondTimestamp{Time: time.Now()}},
		},
		Metrics: []transit.TimeSeries{sampleValue},
	}

	geneva := transit.MonitoredResource{
		Name:             "geneva",
		Type:             transit.Host,
		Status:           transit.HostUp,
		LastCheckTime:    milliseconds.MillisecondTimestamp{Time: time.Now()},
		NextCheckTime:    milliseconds.MillisecondTimestamp{Time: time.Now().Add(time.Minute * 5)},
		LastPlugInOutput: "44/55/888 QA00005-BC",
		Properties: map[string]transit.TypedValue{
			"stateType":       {StringValue: "SOFT"},
			"checkType":       {StringValue: "ACTIVE"},
			"PerformanceData": {StringValue: "007-321 RAD"},
			"ExecutionTime":   {DoubleValue: 3.0},
			"CurrentAttempt":  {IntegerValue: 2},
			"InceptionTime":   {TimeValue: &milliseconds.MillisecondTimestamp{Time: time.Now()}},
		},
		Services: []transit.MonitoredService{localLoadService},
	}

	// Build Monitored Resources
	resources := []transit.MonitoredResource{geneva}

	// TODO: call into API

	b, err := json.Marshal(resources)
	if err == nil {
		s := string(b)
		println(s)
	}
}
