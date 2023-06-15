package main

import (
	"github.com/gwos/tcg/connectors"
	"github.com/gwos/tcg/connectors/events-connector/helpers"
	"github.com/gwos/tcg/services"
	"github.com/rs/zerolog/log"
)

var (
	extCfg = &helpers.ExtConfig{}
)

func main() {
	var (
		entries        = initializeEntrypoints()
		controller     = services.GetController()
		transitService = services.GetTransitService()
	)

	log.Info().Msg("Waiting for configuration to be delivered...")
	if err := transitService.DemandConfig(); err != nil {
		log.Err(err).Msg("Failed to demand config")
		return
	}

	controller.RegisterEntrypoints(entries)

	if err := connectors.Start(); err != nil {
		log.Err(err).Msg("Failed to start connector")
		return
	}

	/* return on quit signal */
	<-transitService.Quit()
}
