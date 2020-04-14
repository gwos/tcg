package main

import (
	"encoding/json"
	"github.com/gin-gonic/gin"
	"github.com/gwos/tng/connectors"
	_ "github.com/gwos/tng/docs"
	"github.com/gwos/tng/log"
	"github.com/gwos/tng/milliseconds"
	"github.com/gwos/tng/services"
	"github.com/gwos/tng/transit"
	"net/http"
	"time"
)

// Default values for 'Group' and loop 'Timer'
const (
	DefaultHostGroupName = "LocalServer"
	DefaultTimer         = 120
)

type InitializeConfigResult struct {
	processes      []string
	groups         []transit.ResourceGroup
	metricsProfile transit.MetricsProfile
	timer          float64
	ownership      transit.HostOwnershipType
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
		if cfg, err := initializeConfig(data); err == nil {
			config = cfg
			chanel <- true
		} else {
			log.Error("[Server Connector]: Error during parsing config. Aborting ...")
			return
		}
	}

	if err := transitService.DemandConfig(
		services.Entrypoint{
			Url:    "/suggest/:viewName/:name",
			Method: "Get",
			Handler: func(c *gin.Context) {
				if c.Param("viewName") == string(transit.Process) {
					c.JSON(http.StatusOK, listSuggestions(c.Param("name")))
				} else {
					c.JSON(http.StatusOK, []transit.MetricDefinition{})
				}
			},
		},
	); err != nil {
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
		_ = connectors.SendInventory([]transit.InventoryResource{*Synchronize(config.metricsProfile.Metrics)}, config.groups, config.ownership)
	}

	for {
		if transitService.Status().Transport != services.Stopped {
			select {
			case <-chanel:
				log.Info("[Server Connector]: Sending inventory ...")
				_ = connectors.SendInventory([]transit.InventoryResource{*Synchronize(config.metricsProfile.Metrics)}, config.groups, config.ownership)
			default:
				log.Info("[Server Connector]: No new config received, skipping inventory ...")
			}
		} else {
			log.Info("[Server Connector]: Transport is stopped ...")
		}
		if transitService.Status().Transport != services.Stopped && len(config.metricsProfile.Metrics) > 0 {
			log.Info("[Server Connector]: Monitoring resources ...")
			err := connectors.SendMetrics([]transit.MonitoredResource{*CollectMetrics(config.metricsProfile.Metrics, time.Duration(config.timer))})
			if err != nil {
				log.Error(err.Error())
			}
		}
		LastCheck = milliseconds.MillisecondTimestamp{Time: time.Now()}
		time.Sleep(time.Duration(int64(config.timer) * int64(time.Second)))
	}
}

func initializeConfig(data []byte) (*InitializeConfigResult, error) {
	var connector = struct {
		MonitorConnection transit.MonitorConnection `json:"monitorConnection"`
		MetricsProfile    transit.MetricsProfile    `json:"metricsProfile"`
		Connections       []struct {
			DeferOwnership transit.HostOwnershipType `json:"deferOwnership"`
		} `json:"groundworkConnections"`
	}{}

	config := InitializeConfigResult{
		processes:      []string{},
		groups:         []transit.ResourceGroup{},
		metricsProfile: transit.MetricsProfile{},
		timer:          0,
		ownership:      transit.Yield,
	}

	err := json.Unmarshal(data, &connector)
	if err != nil {
		return &InitializeConfigResult{
			processes:      []string{},
			groups:         []transit.ResourceGroup{},
			metricsProfile: transit.MetricsProfile{},
			timer:          0,
			ownership:      transit.Yield,
		}, err
	}
	config.timer = float64(DefaultTimer)
	if _, present := connector.MonitorConnection.Extensions["timer"]; present {
		config.timer = connector.MonitorConnection.Extensions["timer"].(float64)
	}

	if _, present := connector.MonitorConnection.Extensions["processes"]; present {
		processesInterface := connector.MonitorConnection.Extensions["processes"].([]interface{})
		for _, process := range processesInterface {
			config.processes = append(config.processes, process.(string))
		}
	}

	if _, present := connector.MonitorConnection.Extensions["groups"]; present {
		groupsInterface := connector.MonitorConnection.Extensions["groups"].([]interface{})
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

	config.metricsProfile = connector.MetricsProfile

	return &config, nil
}
