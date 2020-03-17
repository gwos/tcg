package main

import (
	"encoding/json"
	"github.com/gwos/tng/connectors"
	_ "github.com/gwos/tng/docs"
	"github.com/gwos/tng/log"
	"github.com/gwos/tng/milliseconds"
	"github.com/gwos/tng/services"
	"github.com/gwos/tng/transit"
	"time"
)

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
	var transitService = services.GetTransitService()

	chanel := make(chan bool)

	timer := DefaultTimer
	var processes []string
	var groups []transit.ResourceGroup

	transitService.ConfigHandler = func(data []byte) {
		if p, g, t, err := initializeConfig(data); err == nil {
			processes = p
			groups = g
			timer = t
			chanel <- true
		} else {
			return
		}
	}

	if err := transitService.DemandConfig(); err != nil {
		log.Error(err)
		return
	}

	log.Info("[Server Connector]: Waiting for configuration to be delivered ...")
	<-chanel
	log.Info("[Server Connector]: Configuration received")

	if err := connectors.Start(); err != nil {
		log.Error(err)
		return
	}
	connectors.ControlCHandler()

	for {
		if transitService.Status().Transport != services.Stopped {
			log.Info("TNG ServerConnector: sending inventory ...")
			// TODO: ownership type
			_ = connectors.SendInventory([]transit.InventoryResource{*Synchronize(processes)}, groups, transit.Creator)
		} else {
			log.Info("TNG ServerConnector is stopped ...")
		}
		for i := 0; i < 10; i++ {
			if transitService.Status().Transport != services.Stopped {
				log.Info("TNG ServerConnector: monitoring resources ...")
				err := connectors.SendMetrics([]transit.MonitoredResource{*CollectMetrics(processes, time.Duration(timer))})
				if err != nil {
					log.Error(err.Error())
				}
			}
			LastCheck = milliseconds.MillisecondTimestamp{Time: time.Now()}
			time.Sleep(time.Duration(int64(timer) * int64(time.Second)))
		}
	}
}

func initializeConfig(data []byte) ([]string, []transit.ResourceGroup, int, error) {
	var connector = struct {
		Connection transit.MonitorConnection `json:"monitorConnection"`
	}{}

	err := json.Unmarshal(data, &connector)
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
}
