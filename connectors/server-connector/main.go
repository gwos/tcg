package main

import (
	"bytes"
	"encoding/json"
	"github.com/gin-gonic/gin"
	"github.com/gwos/tcg/cache"
	"github.com/gwos/tcg/connectors"
	_ "github.com/gwos/tcg/docs"
	"github.com/gwos/tcg/log"
	"github.com/gwos/tcg/milliseconds"
	"github.com/gwos/tcg/services"
	"github.com/gwos/tcg/transit"
	"net/http"
	"time"
)

// Default values for 'Group' and loop 'Timer'
const (
	DefaultHostGroupName = "LocalServer"
	DefaultTimer         = 120
	DefaultCacheTimer    = 1
)

type InitializeConfigResult struct {
	Processes      []string
	Groups         []transit.ResourceGroup
	MetricsProfile transit.MetricsProfile
	Timer          float64
	Ownership      transit.HostOwnershipType
}

// @title TCG API Documentation
// @version 1.0

// @host localhost:8099
// @BasePath /api/v1
func main() {
	connectors.ControlCHandler()
	go handleCache()

	var transitService = services.GetTransitService()
	var config InitializeConfigResult
	var configMark []byte
	agentIdMark := transitService.Connector.AgentID

	transitService.ConfigHandler = func(data []byte) {
		log.Info("[Server Connector]: Configuration received")
		if cfg, err := initializeConfig(data); err == nil {
			config = *cfg
			cfgMark, _ := json.Marshal(cfg)

			if !bytes.Equal(configMark, cfgMark) || agentIdMark != transitService.Connector.AgentID {
				configMark = cfgMark
				agentIdMark = transitService.Connector.AgentID
				log.Info("[Server Connector]: Sending inventory ...")
				_ = connectors.SendInventory(
					[]transit.InventoryResource{*Synchronize(config.MetricsProfile.Metrics)},
					config.Groups,
					config.Ownership,
				)
			}
		} else {
			log.Error("[Server Connector]: Error during parsing config. Aborting ...")
			return
		}
	}

	log.Info("[Server Connector]: Waiting for configuration to be delivered ...")
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

	if err := connectors.Start(); err != nil {
		log.Error(err)
		return
	}

	for {
		if len(config.MetricsProfile.Metrics) > 0 {
			log.Info("[Server Connector]: Monitoring resources ...")
			if err := connectors.SendMetrics([]transit.MonitoredResource{
				*CollectMetrics(config.MetricsProfile.Metrics, time.Duration(config.Timer)),
			}); err != nil {
				log.Error(err.Error())
			}
		}

		LastCheck = milliseconds.MillisecondTimestamp{Time: time.Now()}
		time.Sleep(time.Duration(int64(config.Timer) * int64(time.Second)))
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
		Processes:      []string{},
		Groups:         []transit.ResourceGroup{},
		MetricsProfile: transit.MetricsProfile{},
		Timer:          0,
		Ownership:      transit.Yield,
	}

	err := json.Unmarshal(data, &connector)
	if err != nil {
		return &InitializeConfigResult{
			Processes:      []string{},
			Groups:         []transit.ResourceGroup{},
			MetricsProfile: transit.MetricsProfile{},
			Timer:          0,
			Ownership:      transit.Yield,
		}, err
	}
	config.Timer = float64(DefaultTimer)
	if _, present := connector.MonitorConnection.Extensions["timer"]; present {
		config.Timer = connector.MonitorConnection.Extensions["timer"].(float64)
	}

	if _, present := connector.MonitorConnection.Extensions["processes"]; present {
		processesInterface := connector.MonitorConnection.Extensions["processes"].([]interface{})
		for _, process := range processesInterface {
			config.Processes = append(config.Processes, process.(string))
		}
	}

	if _, present := connector.MonitorConnection.Extensions["groups"]; present {
		groupsInterface := connector.MonitorConnection.Extensions["groups"].([]interface{})
		for _, gr := range groupsInterface {
			groupMap := gr.(map[string]interface{})
			config.Groups = append(config.Groups, transit.ResourceGroup{GroupName: groupMap["name"].(string), Type: transit.GroupType(groupMap["type"].(string))})
		}
	} else {
		config.Groups = append(config.Groups, transit.ResourceGroup{GroupName: DefaultHostGroupName, Type: transit.HostGroup})
	}

	if len(connector.Connections) > 0 {
		config.Ownership = connector.Connections[0].DeferOwnership
	}

	config.MetricsProfile = connector.MetricsProfile

	return &config, nil
}

func handleCache() {
	for {
		cache.ProcessesCache.SetDefault("processes", collectProcessesNames())
		time.Sleep(DefaultCacheTimer * time.Minute)
	}
}
