package main

import (
	"encoding/json"
	"github.com/gwos/tng/connectors"
	_ "github.com/gwos/tng/docs"
	"github.com/gwos/tng/log"
	"github.com/gwos/tng/milliseconds"
	"github.com/gwos/tng/services"
	"github.com/gwos/tng/transit"
	"os/exec"
	"time"
)

// Default values for 'Group' and loop 'Timer'
const (
	DefaultHostGroupName = "LocalServer"
	DefaultTimer         = 120
)

var transitService *services.TransitService

// @title TNG API Documentation
// @version 1.0

// @host localhost:8099
// @BasePath /api/v1
func main() {
	connectors.ControlCHandler()
	transitService = services.GetTransitService()
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

	_, groups, timer, err := getConfig()
	if err != nil {
		log.Error(err)
		return
	}
	processes := []string{"watchdogd", "Terminal", "WiFiAgent"}
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
				err := sendMonitoredResources(*CollectMetrics(processes, time.Duration(timer)))
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

