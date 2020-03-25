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
	var metricsProfile transit.MetricsProfile

	transitService.ConfigHandler = func(data []byte) {
		if p, g, t, m, err := initializeConfig(data); err == nil {
			processes = p
			groups = g
			timer = t
			metricsProfile = m
			chanel <- true
		} else {
			log.Error("[Server Connector]: Error during parsing config. Aborting ...")
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

	if transitService.Status().Transport != services.Stopped {
		log.Info("[Server Connector]: Sending inventory ...")
		_ = connectors.SendInventory([]transit.InventoryResource{*Synchronize(metricsProfile.Metrics)}, groups)
	}

	for {
		if transitService.Status().Transport != services.Stopped {
			select {
			case <-chanel:
				log.Info("[Server Connector]: Sending inventory ...")
				_ = connectors.SendInventory([]transit.InventoryResource{*Synchronize(metricsProfile.Metrics)}, groups)
			default:
				log.Info("[Server Connector]: No new config received, skipping inventory ...")
			}
		} else {
			log.Info("[Server Connector]: Transport is stopped ...")
		}
		if transitService.Status().Transport != services.Stopped && len(metricsProfile.Metrics) > 0 {
			log.Info("[Server Connector]: Monitoring resources ...")
			err := connectors.SendMetrics([]transit.MonitoredResource{*CollectMetrics(metricsProfile.Metrics, time.Duration(timer))})
			if err != nil {
				log.Error(err.Error())
			}
		}
		LastCheck = milliseconds.MillisecondTimestamp{Time: time.Now()}
		time.Sleep(time.Duration(int64(timer) * int64(time.Second)))
	}
}

func initializeConfig(data []byte) ([]string, []transit.ResourceGroup, int, transit.MetricsProfile, error) {
	var connector = struct {
		MonitorConnection transit.MonitorConnection `json:"monitorConnection"`
		MetricsProfile    transit.MetricsProfile    `json:"metricsProfile"`
	}{}

	err := json.Unmarshal(data, &connector)
	if err != nil {
		return []string{}, []transit.ResourceGroup{}, -1, transit.MetricsProfile{}, err
	}
	timer := float64(DefaultTimer)
	if _, present := connector.MonitorConnection.Extensions["timer"]; present {
		timer = connector.MonitorConnection.Extensions["timer"].(float64)
	}
	var processes []string
	if _, present := connector.MonitorConnection.Extensions["processes"]; present {
		processesInterface := connector.MonitorConnection.Extensions["processes"].([]interface{})
		for _, process := range processesInterface {
			processes = append(processes, process.(string))
		}
	}
	var groups []transit.ResourceGroup
	if _, present := connector.MonitorConnection.Extensions["groups"]; present {
		groupsInterface := connector.MonitorConnection.Extensions["groups"].([]interface{})
		for _, gr := range groupsInterface {
			groupMap := gr.(map[string]interface{})
			groups = append(groups, transit.ResourceGroup{GroupName: groupMap["name"].(string), Type: transit.GroupType(groupMap["type"].(string))})
		}
	} else {
		groups = append(groups, transit.ResourceGroup{GroupName: DefaultHostGroupName, Type: transit.HostGroup})
	}

	return processes, groups, int(timer), connector.MetricsProfile, nil
}
