package elastic

import (
	"bytes"
	"context"
	"strings"

	"github.com/gwos/tcg/config"
	"github.com/gwos/tcg/connectors"
	"github.com/gwos/tcg/connectors/elastic/clients"
	_ "github.com/gwos/tcg/docs"
	"github.com/gwos/tcg/sdk/transit"
	"github.com/gwos/tcg/services"
	"github.com/rs/zerolog/log"
)

var (
	extConfig         = &ExtConfig{}
	monitorConnection = &transit.MonitorConnection{
		Extensions: extConfig,
	}
	invChksum         []byte
	connector         ElasticConnector
	ctxCancel, cancel = context.WithCancel(context.Background())
)

// temporary solution, will be removed
const templateMetricName = "$view_Template#"

func Run() {
	services.GetController().RegisterEntrypoints(initializeEntrypoints())

	transitService := services.GetTransitService()
	transitService.RegisterConfigHandler(configHandler)
	services.GetTransitService().RegisterExitHandler(cancel)

	log.Info().Msg("waiting for configuration to be delivered ...")
	if err := transitService.DemandConfig(); err != nil {
		log.Err(err).Msg("could not demand config")
		return
	}

	if err := connectors.Start(); err != nil {
		log.Err(err).Msg("could not start connector")
		return
	}

	connectors.StartPeriodic(ctxCancel, connectors.CheckInterval, periodicHandler)

	/* return on quit signal */
	<-transitService.Quit()
}

func configHandler(data []byte) {
	log.Info().Msg("configuration received")
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
		log.Err(err).Msg("could not parse config")
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
	extConfig, _, monitorConnection = tExt, tMetProf, tMonConn
	monitorConnection.Extensions = extConfig

	if err := connector.LoadConfig(*extConfig); err != nil {
		log.Err(err).Msg("could not reload config")
	}

	/* Restart periodic loop */
	cancel()
	ctxCancel, cancel = context.WithCancel(context.Background())
	services.GetTransitService().RegisterExitHandler(cancel)
	connectors.StartPeriodic(ctxCancel, connectors.CheckInterval, periodicHandler)
}

func periodicHandler() {
	if len(connector.monitoringState.Metrics) > 0 {
		metrics, inventory, groups := connector.CollectMetrics()

		chk, chkErr := connector.getInventoryHashSum()
		if chkErr != nil || !bytes.Equal(invChksum, chk) {
			err := connectors.SendInventory(context.Background(), inventory, groups, connector.config.Ownership)
			log.Err(err).Msg("inventory changed: sending inventory")
		}
		if chkErr == nil {
			invChksum = chk
		}

		err := connectors.SendMetrics(context.Background(), metrics, nil)
		log.Err(err).Msg("sending metrics")
	}
}
