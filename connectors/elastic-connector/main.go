package main

import (
	"bytes"
	"github.com/gin-gonic/gin"
	"github.com/gwos/tcg/config"
	"github.com/gwos/tcg/connectors"
	"github.com/gwos/tcg/connectors/elastic-connector/model"
	_ "github.com/gwos/tcg/docs"
	"github.com/gwos/tcg/log"
	"github.com/gwos/tcg/services"
	"net/http"
	"time"
)

func main() {
	connectors.ControlCHandler()

	var transitService = services.GetTransitService()

	connector := ElasticConnector{}

	var cfg *model.ElasticConnectorConfig
	var chksum []byte

	transitService.ConfigHandler = func(data []byte) {
		log.Info("[Elastic Connector]: Configuration received")
		connection, profile, gwConnections := connectors.RetrieveCommonConnectorInfo(data)
		c := model.InitConfig(transitService.Connector.AppType, transitService.Connector.AgentID,
			&connection, &profile, gwConnections)
		cfg = c

		chk, err := connectors.Hashsum(
			config.GetConfig().GWConnections,
			cfg,
		)
		if err != nil || !bytes.Equal(chksum, chk) {
			if err := connector.LoadConfig(cfg); err != nil {
				log.Error("Cannot reload ElasticConnector config: ", err)
			} else {
				_, irs, rgs := connector.CollectMetrics()
				log.Info("[Elastic Connector]: Sending inventory ...")
				err := connectors.SendInventory(irs, rgs, connector.config.OwnershipType)
				if err != nil {
					log.Error(err.Error())
				}
			}
		}
		if err == nil {
			chksum = chk
		}
	}

	if err := transitService.DemandConfig(
		services.Entrypoint{
			Url:    "/suggest/:viewName/:name",
			Method: "Get",
			Handler: func(c *gin.Context) {
				c.JSON(http.StatusOK, connector.ListSuggestions(c.Param("viewName"), c.Param("name")))
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
		if connector.config != nil {
			mrs, irs, rgs := connector.CollectMetrics()

			log.Info("[Elastic Connector]: Sending inventory ...")
			err := connectors.SendInventory(irs, rgs, connector.config.OwnershipType)
			if err != nil {
				log.Error(err.Error())
			}

			log.Info("[Elastic Connector]: Monitoring resources ...")
			err = connectors.SendMetrics(mrs)
			if err != nil {
				log.Error(err.Error())
			}
			time.Sleep(time.Duration(int64(connector.config.Timer) * int64(time.Second)))
		} else {
			time.Sleep(time.Duration(int64(connectors.Timer) * int64(time.Second)))
		}
	}
}
