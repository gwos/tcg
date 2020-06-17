package main

import (
	"github.com/gwos/tcg/connectors"
	"github.com/gwos/tcg/log"
	"github.com/gwos/tcg/services"
	"time"
)

func main() {
	connectors.ControlCHandler()

	var transitService = services.GetTransitService()

	err := transitService.StartController(initializeEntrypoints()...)

	if err != nil {
		log.Error(err.Error())
		return
	}

	for {
		time.Sleep(10 * time.Second)
	}
}
