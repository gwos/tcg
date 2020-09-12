package main

import (
	"bytes"
	"context"
	"github.com/gwos/tcg/config"
	"github.com/gwos/tcg/connectors"
	"github.com/gwos/tcg/log"
	"github.com/gwos/tcg/services"
	"github.com/gwos/tcg/transit"
)

var (
	extConfig         = &ExtConfig{}
	monitorConnection = &transit.MonitorConnection{
		Extensions: extConfig,
	}
	chksum            []byte
	ctxCancel, cancel = context.WithCancel(context.Background())
)

func main() {
	services.GetController().RegisterEntrypoints(initializeEntrypoints())

	transitService := services.GetTransitService()
	transitService.RegisterConfigHandler(configHandler)
	transitService.RegisterExitHandler(cancel)

	log.Info("[Prometheus Connector]: Waiting for configuration to be delivered ...")
	if err := transitService.DemandConfig(); err != nil {
		log.Error("[Prometheus Connector]: ", err)
		return
	}

	if err := connectors.Start(); err != nil {
		log.Error("[Prometheus Connector]: ", err)
		return
	}

	connectors.StartPeriodic(ctxCancel, connectors.CheckInterval, periodicHandler)

	/* prevent return */
	<-make(chan bool, 1)
}

func configHandler(data []byte) {
	log.Info("[Prometheus Connector]: Configuration received")
	/* Init config with default values */
	tExt := &ExtConfig{
		Groups: []transit.ResourceGroup{{
			GroupName: defaultHostGroupName,
			Type:      transit.HostGroup,
		}},
		Resources:        []Resource{},
		Services:         []string{},
		CheckInterval:    connectors.DefaultCheckInterval,
		Ownership:        transit.Yield,
	}
	tMonConn := &transit.MonitorConnection{Extensions: tExt}
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
	extConfig, _, monitorConnection = tExt, tMetProf, tMonConn
	monitorConnection.Extensions = extConfig
	/* Process checksums */
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
	/* Restart periodic loop */
	cancel()
	ctxCancel, cancel = context.WithCancel(context.Background())
	services.GetTransitService().RegisterExitHandler(cancel)
	connectors.StartPeriodic(ctxCancel, connectors.CheckInterval, periodicHandler)
}

func periodicHandler() {
	pull(extConfig.Resources)
}
