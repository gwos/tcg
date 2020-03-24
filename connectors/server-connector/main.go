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

type InitializeConfigResult struct {
	processes []string
	groups []transit.ResourceGroup
	timer int
	ownership transit.HostOwnershipType
}


// @title TNG API Documentation
// @version 1.0

// @host localhost:8099
// @BasePath /api/v1
func main() {
	var transitService = services.GetTransitService()

	chanel := make(chan bool)
	var config *InitializeConfigResult
	transitService.ConfigHandler = func(data []byte) {
		cfg, err := initializeConfig(data)
		if err != nil {
			return
		}
		config = cfg
		chanel <- true
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
			_ = connectors.SendInventory([]transit.InventoryResource{*Synchronize(config.processes)}, config.groups, config.ownership)
		} else {
			log.Info("TNG ServerConnector is stopped ...")
		}
		for i := 0; i < 10; i++ {
			if transitService.Status().Transport != services.Stopped {
				log.Info("TNG ServerConnector: monitoring resources ...")
				err := connectors.SendMetrics([]transit.MonitoredResource{*CollectMetrics(config.processes, time.Duration(config.	timer))})
				if err != nil {
					log.Error(err.Error())
				}
			}
			LastCheck = milliseconds.MillisecondTimestamp{Time: time.Now()}
			time.Sleep(time.Duration(int64(config.timer) * int64(time.Second)))
		}
	}
}

func initializeConfig(data []byte) (*InitializeConfigResult, error) {
	// just grab the fields that are needed by this connector
	var connector = struct {
		Connection transit.MonitorConnection `json:"monitorConnection"`
		Connections []struct {
			DeferOwnership transit.HostOwnershipType	`json:"deferOwnership"`
		} `json:"groundworkConnections"`
	}{}
	config := InitializeConfigResult{
		processes:   []string{},
		groups: []transit.ResourceGroup{},
		timer: 0,
		ownership: transit.Yield,
	}
	err := json.Unmarshal(data, &connector)
	if err != nil {
		return &InitializeConfigResult{
			processes:   []string{},
			groups: []transit.ResourceGroup{},
			timer: 0,
			ownership: transit.Yield,
		}, err
	}
	timer := float64(DefaultTimer)
	if _, present := connector.Connection.Extensions["timer"]; present {
		timer = connector.Connection.Extensions["timer"].(float64)
	}
	config.timer = int(timer)

	if _, present := connector.Connection.Extensions["processes"]; present {
		processesInterface := connector.Connection.Extensions["processes"].([]interface{})
		for _, process := range processesInterface {
			config.processes = append(config.processes, process.(string))
		}
	}
	if _, present := connector.Connection.Extensions["groups"]; present {
		groupsInterface := connector.Connection.Extensions["groups"].([]interface{})
		for _, gr := range groupsInterface {
			groupMap := gr.(map[string]interface{})
			config.groups = append(config.groups, transit.ResourceGroup{GroupName: groupMap["name"].(string), Type: transit.GroupType(groupMap["type"].(string))})
		}
	} else {
		config.groups = append(config.groups, transit.ResourceGroup{GroupName: DefaultHostGroupName, Type: transit.HostGroup})
	}
	if len(connector.Connections) > 0 {
		config.ownership = connector.Connections[0].DeferOwnership
	}
	return &config, nil
}
