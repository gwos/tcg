package main

import (
	"github.com/gwos/tng/connectors"
	"github.com/gwos/tng/connectors/elastic-connector/model"
	_ "github.com/gwos/tng/docs"
	"github.com/gwos/tng/log"
	"github.com/gwos/tng/milliseconds"
	"github.com/gwos/tng/services"
	"time"
)

// LastCheck provide time of last processes state check
var LastCheck milliseconds.MillisecondTimestamp

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

	if err := transitService.DemandConfig(); err != nil {
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

	connector := ElasticConnector{Config: config}

	_, irs, rgs := connector.CollectMetrics()
	if transitService.Status().Transport != services.Stopped {
		log.Info("[Elastic Connector]: Sending inventory ...")
		_ = connectors.SendInventory(irs, rgs, config.Ownership)
	}

	for {
		mrs, irs, rgs := connector.CollectMetrics()
		if transitService.Status().Transport != services.Stopped {
			select {
			case <-chanel:
				log.Info("[Elastic Connector]: Sending inventory ...")
				_ = connectors.SendInventory(irs, rgs, config.Ownership)
			default:
				log.Info("[Elastic Connector]: No new config received, skipping inventory ...")
			}
		} else {
			log.Info("[Elastic Connector]: Transport is stopped ...")
		}
		if transitService.Status().Transport != services.Stopped && len(mrs) > 0 {
			log.Info("[Elastic Connector]: Monitoring resources ...")
			err := connectors.SendMetrics(mrs)
			if err != nil {
				log.Error(err.Error())
			}
		}
		LastCheck = milliseconds.MillisecondTimestamp{Time: time.Now()}
		time.Sleep(time.Duration(int64(config.Timer) * int64(time.Second)))
	}
}
