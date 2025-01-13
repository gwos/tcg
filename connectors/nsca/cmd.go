//go:build !codeanalysis

package nsca

import (
	"context"

	"github.com/gwos/tcg/connectors"
	"github.com/gwos/tcg/connectors/nsca/nsca"
	"github.com/gwos/tcg/services"
	"github.com/rs/zerolog/log"
)

var (
	nscaCancel context.CancelFunc
)

// @title TCG API Documentation
// @version 1.0

// @host localhost:8099
// @BasePath /api/v1
func Run() {
	services.GetController().RegisterEntrypoints(initializeEntrypoints())

	transitService := services.GetTransitService()
	transitService.RegisterExitHandler(func() {
		if nscaCancel != nil {
			nscaCancel()
		}
	})

	log.Info().Msg("waiting for configuration to be delivered ...")
	if err := transitService.DemandConfig(); err != nil {
		log.Err(err).Msg("could not demand config")
		return
	}
	if err := connectors.Start(); err != nil {
		log.Err(err).Msg("could not start connector")
		return
	}

	ctx, cancel := context.WithCancel(context.Background())
	nscaCancel = cancel
	nsca.Start(ctx, makeNSCAHandler())

	/* return on quit signal */
	<-transitService.Quit()
}
