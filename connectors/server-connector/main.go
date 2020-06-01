package main

import (
	"bytes"
	"github.com/gin-gonic/gin"
	"github.com/gwos/tcg/cache"
	"github.com/gwos/tcg/config"
	"github.com/gwos/tcg/connectors"
	_ "github.com/gwos/tcg/docs"
	"github.com/gwos/tcg/log"
	"github.com/gwos/tcg/services"
	"github.com/gwos/tcg/transit"
	"net/http"
	"time"
)

// Default values for 'Group' and loop 'Timer'
const (
	DefaultCacheTimer = 1
)

type InitializeConfigResult struct {
	Processes      []string
	Groups         []transit.ResourceGroup
	MetricsProfile transit.MetricsProfile
	Timer          float64
	Ownership      transit.HostOwnershipType
}

// How to test:
//func main() {
//  str := "GW:toPercentageLimit(arg1)"
//	m := map[string]interface{}{
//		"arg1": 1.87,
//		"arg2": 1000.0,
//		"arg3": 1.0,
//	}
//	val, _, err := connectors.EvaluateGroundworkFunction(str, m)
//	fmt.Println(val)
//	fmt.Println(err)
//}

// @title TCG API Documentation
// @version 1.0

// @host localhost:8099
// @BasePath /api/v1
func main() {
	connectors.ControlCHandler()
	go handleCache()

	var transitService = services.GetTransitService()
	var cfg ServerConnectorConfig
	var chksum []byte

	transitService.ConfigHandler = func(data []byte) {
		log.Info("[Server Connector]: Configuration received")
		if monitorConn, profile, gwConnections, err := connectors.RetrieveCommonConnectorInfo(data); err == nil {
			c := InitConfig(monitorConn, profile, gwConnections)
			cfg = *c
			connectors.Timer = cfg.Timer
			chk, err := connectors.Hashsum(
				config.GetConfig().Connector.AgentID,
				config.GetConfig().GWConnections,
				cfg,
			)
			if err != nil || !bytes.Equal(chksum, chk) {
				log.Info("[Server Connector]: Sending inventory ...")
				_ = connectors.SendInventory(
					[]transit.InventoryResource{*Synchronize(cfg.MetricsProfile.Metrics)},
					cfg.Groups,
					cfg.Ownership,
				)
			}
			if err == nil {
				chksum = chk
			}
		} else {
			log.Error("[Server Connector]: Error during parsing config. Aborting ...")
			return
		}
	}

	log.Info("[Server Connector]: Waiting for configuration to be delivered ...")
	if err := transitService.DemandConfig(
		services.Entrypoint{
			Url:    "/suggest/services/:viewName/:name",
			Method: "Get",
			Handler: func(c *gin.Context) {
				if c.Param("viewName") == string(transit.Process) {
					c.JSON(http.StatusOK, listSuggestions(c.Param("name")))
				} else {
					c.JSON(http.StatusOK, []transit.MetricDefinition{})
				}
			},
		},
		services.Entrypoint{
			Url:    "/suggest/expressions/:name",
			Method: "Get",
			Handler: func(c *gin.Context) {
				c.JSON(http.StatusOK, connectors.ListExpressions(c.Param("name")))
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
		if len(cfg.MetricsProfile.Metrics) > 0 {
			log.Info("[Server Connector]: Monitoring resources ...")
			if err := connectors.SendMetrics([]transit.MonitoredResource{
				*CollectMetrics(cfg.MetricsProfile.Metrics),
			}); err != nil {
				monitoredResource := CollectMetrics(cfg.MetricsProfile.Metrics)
				monitoredResource.Services = connectors.EvaluateExpressions(monitoredResource.Services)
				err := connectors.SendMetrics([]transit.MonitoredResource{*monitoredResource})
				if err != nil {
					log.Error(err.Error())
				}
			}
			time.Sleep(time.Duration(connectors.Timer * int64(time.Second)))
		}
	}
}

func handleCache() {
	for {
		cache.ProcessesCache.SetDefault("processes", collectProcessesNames())
		time.Sleep(DefaultCacheTimer * time.Minute)
	}
}
