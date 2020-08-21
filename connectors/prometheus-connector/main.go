package main

import (
	"bytes"
	"fmt"
	"github.com/gwos/tcg/config"
	"github.com/gwos/tcg/connectors"
	"github.com/gwos/tcg/log"
	"github.com/gwos/tcg/services"
	"github.com/gwos/tcg/transit"
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
	chksum []byte
)

func main() {
	config.Version.Tag = buildTag
	config.Version.Time = buildTime
	log.Info(fmt.Sprintf("[Prometheus Connector]: Version: %s   /   Build time: %s", config.Version.Tag, config.Version.Time))

	transitService := services.GetTransitService()
	transitService.RegisterConfigHandler(configHandler)

	log.Info("[Prometheus Connector]: Waiting for configuration to be delivered ...")
	if err := transitService.DemandConfig(initializeEntrypoints()...); err != nil {
		log.Error("[Prometheus Connector]: ", err)
		return
	}

	if err := connectors.Start(); err != nil {
		log.Error("[Prometheus Connector]: ", err)
		return
	}

	log.Info("[Prometheus Connector]: Waiting for configuration ...")
	connectors.StartPeriodic(nil, extConfig.Timer, func() {
		pull(extConfig.Resources)
	})
}

func configHandler(data []byte) {
	log.Info("[Prometheus Connector]: Configuration received")
	/* Init config with default values */
	tExt := &ExtConfig{
		Groups: []transit.ResourceGroup{{
			GroupName: defaultHostGroupName,
			Type:      transit.HostGroup,
		}},
		Resources: []Resource{},
		Services:  []string{},
		Timer:     connectors.DefaultTimer,
		Ownership: transit.Yield,
	}
	tMonConn := &connectors.MonitorConnection{Extensions: tExt}
	tMetProf := &transit.MetricsProfile{}
	if err := connectors.UnmarshalConfig(data, tMetProf, tMonConn); err != nil {
		log.Error("[Prometheus Connector]: Error during parsing config.", err.Error())
		return
	}
	/* Update config with received values */
	gwConnections := config.GetConfig().GWConnections
	if len(gwConnections) > 0 {
		tExt.Ownership = transit.HostOwnershipType(gwConnections[0].DeferOwnership)
	}
	extConfig, metricsProfile, monitorConnection = tExt, tMetProf, tMonConn
	monitorConnection.Extensions = extConfig

	chk, err := connectors.Hashsum(
		config.GetConfig().Connector.AgentID,
		config.GetConfig().GWConnections,
		extConfig,
	)
	if err != nil || !bytes.Equal(chksum, chk) {
		log.Info("[Prometheus Connector]: Sending inventory ...")
		_ = connectors.SendInventory(
			*Synchronize(),
			extConfig.Groups,
			extConfig.Ownership,
		)
	}
	if err == nil {
		chksum = chk
	}
}
