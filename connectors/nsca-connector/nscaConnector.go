package main

import (
	"context"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/gwos/tcg/connectors/nsca-connector/parser"
	"github.com/gwos/tcg/services"
	"github.com/rs/zerolog/log"
)

func initializeEntrypoints() []services.Entrypoint {
	rv := make([]services.Entrypoint, 6)
	for _, dataFormat := range []parser.DataFormat{parser.Bronx, parser.NSCA, parser.NSCAAlt} {
		rv = append(rv, services.Entrypoint{
			Handler: makeEntrypointHandler(dataFormat),
			Method:  http.MethodPost,
			URL:     fmt.Sprintf("check/%s", dataFormat),
		})
	}
	return rv
}

func makeEntrypointHandler(dataFormat parser.DataFormat) func(*gin.Context) {
	return func(c *gin.Context) {
		var (
			err     error
			payload []byte
		)
		ctx, span := services.StartTraceSpan(context.Background(), "connectors", "EntrypointHandler")
		defer func() {
			services.EndTraceSpan(span,
				services.TraceAttrError(err),
				services.TraceAttrPayloadLen(payload),
				services.TraceAttrEntrypoint(c.FullPath()),
			)
		}()

		payload, err = c.GetRawData()
		if err != nil {
			log.Warn().Err(err).
				Str("entrypoint", c.FullPath()).
				Msg("could not process incoming request")
			c.JSON(http.StatusBadRequest, err.Error())
			return
		}
		if _, err = parser.ProcessMetrics(ctx, payload, dataFormat); err != nil {
			log.Warn().Err(err).
				Str("entrypoint", c.FullPath()).
				Str("dataFormat", string(dataFormat)).
				Bytes("payload", payload).
				Msg("could not process metrics")
			c.JSON(http.StatusBadRequest, err.Error())
			return
		}
		c.JSON(http.StatusOK, nil)
	}
}
