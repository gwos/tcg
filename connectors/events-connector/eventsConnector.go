package main

import (
	"encoding/json"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/gwos/tcg/connectors"
	"github.com/gwos/tcg/connectors/events-connector/helpers"
	"github.com/gwos/tcg/sdk/transit"
	"github.com/gwos/tcg/services"
	"github.com/prometheus/alertmanager/template"
)

// receiver godoc
// @Description Receive, filter, transform and send alerts/events to Groundwork
// @Tags events, connector
// @Produce json
// @Accept json
// @Param body body request true "request body" Format(template.Data)
// @Success 200 {string} string "OK"
// @Failure 500 {string} string "Internal server error"
// @Failure 400 {string} string "Bad request"
// @Router /events [post]
func receiver(c *gin.Context) {
	var data template.Data
	if err := json.NewDecoder(c.Request.Body).Decode(&data); err != nil {
		c.JSON(http.StatusBadRequest, err.Error())
		return
	}

	host, group, mb, err := helpers.GetMetricBuildersFromPrometheusData(data, helpers.GetExtConfig())
	if err != nil {
		c.JSON(http.StatusInternalServerError, err.Error())
		return
	}

	monitoredServices := make([]transit.MonitoredService, 0, len(mb))
	for _, m := range mb {
		service, err := connectors.BuildServiceForMetric(host, m)
		if err != nil {
			c.JSON(http.StatusInternalServerError, err.Error())
		}
		service.Status = transit.ServiceWarning
		service.LastPluginOutput = helpers.GetLastPluginOutput(m.Tags)
		monitoredServices = append(monitoredServices, *service)
	}

	resource, err := connectors.CreateResource(host, monitoredServices)
	if err != nil {
		c.JSON(http.StatusInternalServerError, err.Error())
		return
	}

	groups := make([]transit.ResourceGroup, 0)
	if group != "" {
		resourceRef := connectors.CreateResourceRef(host, "", transit.ResourceTypeHost)
		resourceGroup := connectors.CreateResourceGroup(group, group, transit.HostGroup, []transit.ResourceRef{resourceRef})
		groups = append(groups, resourceGroup)
	}

	if err = connectors.SendMetrics(c.Request.Context(), []transit.MonitoredResource{*resource}, &groups); err != nil {
		c.JSON(http.StatusInternalServerError, err.Error())
		return
	}
}

func initializeEntrypoints() []services.Entrypoint {
	return []services.Entrypoint{
		{
			URL:     "/receive/events",
			Method:  http.MethodPost,
			Handler: receiver,
		},
	}
}
