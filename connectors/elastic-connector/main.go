package main

import (
	"bytes"
	"context"
	"github.com/gwos/tcg/config"
	"github.com/gwos/tcg/connectors"
	"github.com/gwos/tcg/connectors/elastic-connector/clients"
	_ "github.com/gwos/tcg/docs"
	"github.com/gwos/tcg/log"
	"github.com/gwos/tcg/services"
	"github.com/gwos/tcg/transit"
	"strings"
)

// Variables to control connector version and build time.
// Can be overridden during the build step.
// See README for details.
var (
	extConfig         = &ExtConfig{}
	metricsProfile    = &transit.MetricsProfile{}
	monitorConnection = &transit.MonitorConnection{
		Extensions: extConfig,
	}
	cfgChksum []byte
	invChksum []byte
	connector ElasticConnector
)

// temporary solution, will be removed
const templateMetricName = "$view_Template#"

func main() {
	services.GetController().RegisterEntrypoints(initializeEntrypoints())

	ctxExit, exitHandler := context.WithCancel(context.Background())
	transitService := services.GetTransitService()
	transitService.RegisterConfigHandler(configHandler)
	transitService.RegisterExitHandler(exitHandler)

	log.Info("[Elastic Connector]: Waiting for configuration to be delivered ...")
	if err := transitService.DemandConfig(); err != nil {
		log.Error("[Elastic Connector]: ", err)
		return
	}

	if err := connectors.Start(); err != nil {
		log.Error("[Elastic Connector]: ", err)
		return
	}

	connectors.StartPeriodic(ctxExit, extConfig.CheckInterval, func() {
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
		CheckInterval:      connectors.DefaultCheckInterval,
		AppType:            config.GetConfig().Connector.AppType,
		AgentID:            config.GetConfig().Connector.AgentID,
		GWConnections:      config.GetConfig().GWConnections,
		Ownership:          transit.Yield,
		Views:              make(map[string]map[string]transit.MetricDefinition),
	}
	tMonConn := &transit.MonitorConnection{Extensions: tExt}
	tMetProf := &transit.MetricsProfile{}
	if err := connectors.UnmarshalConfig(data, tMetProf, tMonConn); err != nil {
		log.Error("[Elastic Connector]: Error during parsing config.", err.Error())
		return
	}
	/* Update config with received values */
	if tMonConn.Server != "" {
		servers := strings.Split(tMonConn.Server, ",")
		for i, server := range servers {
			if !strings.HasPrefix(server, defaultProtocol) {
				servers[i] = defaultProtocol + ":" + "//" + server
			}
		}
		tExt.Servers = servers
	}
	if !strings.HasPrefix(tExt.Kibana.ServerName, defaultProtocol) {
		kibanaServerName := defaultProtocol + ":" + "//" + tExt.Kibana.ServerName
		tExt.Kibana.ServerName = kibanaServerName
	}
	if !strings.HasSuffix(tExt.Kibana.ServerName, "/") {
		kibanaServerName := tExt.Kibana.ServerName + "/"
		tExt.Kibana.ServerName = kibanaServerName
	}
	if tExt.GroupNameByUser && tExt.HostGroupField == defaultHostGroupLabel {
		tExt.HostGroupField = defaultHostGroupName
	}
	for _, metric := range tMetProf.Metrics {
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
	if len(tExt.GWConnections) > 0 {
		for _, conn := range tExt.GWConnections {
			if conn.DeferOwnership != "" {
				ownership := transit.HostOwnershipType(tExt.GWConnections[0].DeferOwnership)
				if ownership != "" {
					tExt.Ownership = ownership
					break
				}
			}
		}
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
