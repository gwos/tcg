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
	DefaultCacheTimer = 1
)

// @title TCG API Documentation
// @version 1.0

// @host localhost:8099
// @BasePath /api/v1
func main() {
	connectors.ControlCHandler()

	go handleCache()

	var transitService = services.GetTransitService()
	var config ServerConnectorConfig
	var configMark []byte
	agentIdMark := transitService.Connector.AgentID

	transitService.ConfigHandler = func(data []byte) {
		log.Info("[Server Connector]: Configuration received")
		connection, profile, gwConnections := connectors.RetrieveCommonConnectorInfo(data)
		cfg := InitConfig(&connection, &profile, gwConnections)
		config = *cfg
		connectors.Timer = config.Timer
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

func handleCache() {
	for {
		cache.ProcessesCache.SetDefault("processes", collectProcessesNames())
		time.Sleep(DefaultCacheTimer * time.Minute)
	}
}
