package main

import (
	"context"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/gwos/tcg/connectors/checker-connector/parser"
	"github.com/gwos/tcg/services"
	"github.com/rs/zerolog/log"
)

// ScheduleTask defines command
type ScheduleTask struct {
	CombinedOutput bool              `json:"combinedOutput,omitempty"`
	Command        []string          `json:"command"`
	Cron           string            `json:"cron"`
	DataFormat     parser.DataFormat `json:"dataFormat"`
	Environment    []string          `json:"environment,omitempty"`
}

func (t ScheduleTask) String() string {
	return fmt.Sprintf(
		"%s [%s] %v %v",
		t.DataFormat,
		t.Cron,
		t.Command,
		t.Environment,
	)
}

// ExtConfig defines the MonitorConnection extensions configuration
type ExtConfig struct {
	Schedule []ScheduleTask `json:"schedule"`
}

// Validate validates value
func (cfg ExtConfig) Validate() error {
	for _, task := range cfg.Schedule {
		if len(task.Command) == 0 {
			return fmt.Errorf("ExtConfig Schedule item error: Command is empty")
		}
	}
	return nil
}

func initializeEntrypoints() []services.Entrypoint {
	rv := make([]services.Entrypoint, 6)
	for _, dataFormat := range []parser.DataFormat{parser.Bronx, parser.NSCA, parser.NSCAAlt} {
		rv = append(rv, services.Entrypoint{
			Handler: makeEntrypointHandler(dataFormat),
			Method:  http.MethodPost,
			URL:     fmt.Sprintf("checker/%s", dataFormat),
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
