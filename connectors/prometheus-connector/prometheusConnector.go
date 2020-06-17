package main

import (
	"github.com/gin-gonic/gin"
	"github.com/gwos/tcg/services"
)

// initializeEntrypoints - function for setting entrypoints,
// that will be available through the Server Connector API
func initializeEntrypoints() []services.Entrypoint {
	return append(make([]services.Entrypoint, 2),
		services.Entrypoint{
			Url:    "/synchronize",
			Method: "Post",
			Handler: func(c *gin.Context) {
				//TODO: parse body and send to foundation
			},
		},
		services.Entrypoint{
			Url:    "/send-metrics",
			Method: "Post",
			Handler: func(c *gin.Context) {
				//TODO: parse body and send to foundation
			},
		},
	)
}
