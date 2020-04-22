package main

import (
	"github.com/gin-gonic/gin"
	"github.com/gwos/tng/connectors"
	"github.com/gwos/tng/connectors/elastic-connector/model"
	_ "github.com/gwos/tng/docs"
	"github.com/gwos/tng/log"
	"github.com/gwos/tng/services"
	"net/http"
	"time"
)

func main() {
	var transitService = services.GetTransitService()

	chanel := make(chan bool)

	var config *model.ElasticConnectorConfig

	transitService.ConfigHandler = func(data []byte) {
		connection, profile, ownership := connectors.RetrieveCommonConnectorInfo(data)
		cfg := model.InitConfig(&connection, &profile, ownership)
		config = cfg
		chanel <- true
	}

	var connector *ElasticConnector

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

	log.Info("[Elastic Connector]: Waiting for configuration to be delivered ...")
	<-chanel
	log.Info("[Elastic Connector]: Configuration received")

	if err := connectors.Start(); err != nil {
		log.Error(err)
		return
	}
	connectors.ControlCHandler()

	var err error
	connector, err = InitElasticConnector(config)
	if err != nil {
		log.Error("Cannot initialize ElasticConnector")
		return
	}
	_, irs, rgs := connector.CollectMetrics()
	if transitService.Status().Transport != services.Stopped {
		log.Info("[Elastic Connector]: Sending inventory ...")
		_ = connectors.SendInventory(irs, rgs, config.Ownership)
	}

	for {
		if transitService.Status().Transport != services.Stopped {
			select {
			case <-chanel:
				err := connector.ReloadConfig(config)
				if err != nil {
					log.Error("Cannot reload ElasticConnector config: ", err)
				}
			default:
				log.Info("ElasticConnector: No new config received.")
			}
			mrs, irs, rgs := connector.CollectMetrics()
			log.Info("[Elastic Connector]: Sending inventory ...")
			err := connectors.SendInventory(irs, rgs, config.Ownership)
			if err != nil {
				log.Error(err.Error())
			}
			log.Info("[Elastic Connector]: Monitoring resources ...")
			err = connectors.SendMetrics(mrs)
			if err != nil {
				log.Error(err.Error())
			}
		} else {
			log.Info("[Elastic Connector]: Transport is stopped ...")
		}

		time.Sleep(time.Duration(int64(config.Timer) * int64(time.Second)))
	}
}
