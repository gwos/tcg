package main

import (
	"bytes"
	"context"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/gwos/tcg/config"
	"github.com/gwos/tcg/connectors"
	"github.com/gwos/tcg/connectors/elastic-connector/clients"
	_ "github.com/gwos/tcg/docs"
	"github.com/gwos/tcg/log"
	"github.com/gwos/tcg/services"
	"github.com/gwos/tcg/transit"
	"net/http"
	"strings"
)

// Variables to control connector version and build time.
// Can be overridden during the build step.
// See README for details.
var (
	buildTime = "Build time not provided"
	buildTag  = "8.x.x"

	extConfig         = &ExtConfig{}
	metricsProfile    = &transit.MetricsProfile{}
	monitorConnection = &connectors.MonitorConnection{
		Extensions: extConfig,
	}
	cfgChksum []byte
	invChksum []byte
	connector ElasticConnector
)

// temporary solution, will be removed
const templateMetricName = "$view_Template#"

func main() {
	config.Version.Tag = buildTag
	config.Version.Time = buildTime
	log.Info(fmt.Sprintf("[Elastic Connector]: BuildVersion: %s   /   Build time: %s", config.Version.Tag, config.Version.Time))

	ctxExit, exitHandler := context.WithCancel(context.Background())
	transitService := services.GetTransitService()
	transitService.RegisterConfigHandler(configHandler)
	transitService.RegisterExitHandler(exitHandler)

	log.Info("[Elastic Connector]: Waiting for configuration to be delivered ...")
	if err := transitService.DemandConfig(
		services.Entrypoint{
			Url:    "/suggest/:viewName",
			Method: "Get",
			Handler: func(c *gin.Context) {
				c.JSON(http.StatusOK, connector.ListSuggestions(c.Param("viewName"), ""))
			},
		},
		services.Entrypoint{
			Url:    "/suggest/:viewName/:name",
			Method: "Get",
			Handler: func(c *gin.Context) {
				c.JSON(http.StatusOK, connector.ListSuggestions(c.Param("viewName"), c.Param("name")))
			},
		},
		services.Entrypoint{
			Url:    "/expressions/suggest/:name",
			Method: "Get",
			Handler: func(c *gin.Context) {
				c.JSON(http.StatusOK, connectors.ListExpressions(c.Param("name")))
			},
		},
		services.Entrypoint{
			Url:     "/expressions/evaluate",
			Method:  "Post",
			Handler: connectors.EvaluateExpressionHandler,
		},
	); err != nil {
		log.Error("[Elastic Connector]: ", err)
		return
	}

	if err := connectors.Start(); err != nil {
		log.Error("[Elastic Connector]: ", err)
		return
	}

	connectors.StartPeriodic(ctxExit, extConfig.Timer, func() {
		if len(connector.monitoringState.Metrics) > 0 {
			metrics, inventory, groups := connector.CollectMetrics()

			chk, chkErr := connector.getInventoryHashSum()
			if chkErr != nil || !bytes.Equal(invChksum, chk) {
				log.Info("[Elastic Connector]: Inventory changed. Sending inventory ...")
				err := connectors.SendInventory(inventory, groups, connector.config.Ownership)
				if err != nil {
					log.Error("[Elastic Connector]: ", err.Error())
				}
			}
			if chkErr == nil {
				invChksum = chk
			}

			log.Info("[Elastic Connector]: Monitoring resources ...")
			err := connectors.SendMetrics(metrics)
			if err != nil {
				log.Error("[Elastic Connector]: ", err.Error())
			}
		}
	})
}

func configHandler(data []byte) {
	log.Info("[Elastic Connector]: Configuration received")
	/* Init config with default values */
	tExt := &ExtConfig{
		Kibana: Kibana{
			ServerName: defaultKibanaServerName,
			Username:   defaultKibanaUsername,
			Password:   defaultKibanaPassword,
		},
		Servers: []string{defaultElasticServer},
		CustomTimeFilter: clients.KTimeFilter{
			From: defaultTimeFilterFrom,
			To:   defaultTimeFilterTo,
		},
		OverrideTimeFilter: defaultAlwaysOverrideTimeFilter,
		HostNameField:      defaultHostNameLabel,
		HostGroupField:     defaultHostGroupLabel,
		GroupNameByUser:    defaultGroupNameByUser,
		Timer:              connectors.DefaultTimer,
		AppType:            config.GetConfig().Connector.AppType,
		AgentID:            config.GetConfig().Connector.AgentID,
		GWConnections:      config.GetConfig().GWConnections,
		Ownership:          transit.Yield,
		Views:              make(map[string]map[string]transit.MetricDefinition),
	}
	tMonConn := &connectors.MonitorConnection{Extensions: tExt}
	tMetProf := &transit.MetricsProfile{}
	if err := connectors.UnmarshalConfig(data, tMetProf, tMonConn); err != nil {
		log.Error("[Elastic Connector]: Error during parsing config.", err.Error())
		return
	}
	/* Update config with received values */
	if tMonConn.Server != "" {
		servers := strings.Split(monitorConnection.Server, ",")
		for i, server := range servers {
			if !strings.HasPrefix(server, defaultProtocol) {
				servers[i] = defaultProtocol + ":" + "//" + server
			}
		}
		tExt.Servers = servers
	}
	if metricsProfile != nil {
		for _, metric := range metricsProfile.Metrics {
			// temporary solution, will be removed
			if templateMetricName == metric.Name || !metric.Monitored {
				continue
			}
			if tExt.Views[metric.ServiceType] != nil {
				tExt.Views[metric.ServiceType][metric.Name] = metric
			} else {
				metrics := make(map[string]transit.MetricDefinition)
				metrics[metric.Name] = metric
				tExt.Views[metric.ServiceType] = metrics
			}
		}
	}
	if len(tExt.GWConnections) > 0 {
		tExt.Ownership = transit.HostOwnershipType(tExt.GWConnections[0].DeferOwnership)
	}
	tExt.replaceIntervalTemplates()
	extConfig, metricsProfile, monitorConnection = tExt, tMetProf, tMonConn
	monitorConnection.Extensions = extConfig
	/* process checksums */
	chk, err := connectors.Hashsum(extConfig)
	if err != nil || !bytes.Equal(cfgChksum, chk) {
		if err := connector.LoadConfig(*extConfig); err != nil {
			log.Error("[Elastic Connector]: Cannot reload ElasticConnector config: ", err)
		} else {
			_, inventory, groups := connector.CollectMetrics()
			log.Info("[Elastic Connector]: Sending inventory ...")
			if err := connectors.SendInventory(inventory, groups, connector.config.Ownership); err != nil {
				log.Error("[Elastic Connector]: ", err.Error())
			}
			if invChk, err := connector.getInventoryHashSum(); err == nil {
				invChksum = invChk
			}
		}
	}
	if err == nil {
		cfgChksum = chk
	}
}
