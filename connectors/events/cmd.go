package events

import (
	"github.com/gwos/tcg/connectors"
	"github.com/gwos/tcg/connectors/events/helpers"
	"github.com/gwos/tcg/services"
	"github.com/rs/zerolog/log"
)

func Run() {
	var (
		entries        = initializeEntrypoints()
		controller     = services.GetController()
		transitService = services.GetTransitService()
	)

	transitService.RegisterConfigHandler(helpers.ConfigHandler)
	transitService.RegisterExitHandler(*helpers.GetCancelFunc())
	controller.RegisterEntrypoints(entries)

	log.Info().Msg("Waiting for configuration to be delivered...")
	if err := transitService.DemandConfig(); err != nil {
		log.Err(err).Msg("Failed to demand config")
		return
	}

	if err := connectors.Start(); err != nil {
		log.Err(err).Msg("Failed to start connector")
		return
	}

	/* return on quit signal */
	<-transitService.Quit()
}
