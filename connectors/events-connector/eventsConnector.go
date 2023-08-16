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
	"github.com/rs/zerolog/log"
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
		log.Err(err).
			Interface("body", c.Request.Body).
			Msg("could not decode incomings")
		c.JSON(http.StatusBadRequest, err.Error())
		return
	}
	log.Debug().Interface("data", data).Msg("receive data")

	results, err := helpers.ParsePrometheusData(data, helpers.GetExtConfig())
	if err != nil {
		log.Debug().Err(err).
			Msg("could not parse prometheus data")
		c.JSON(http.StatusInternalServerError, err.Error())
		return
	}

	groups := make([]transit.ResourceGroup, 0)
	hostToServiceMap := make(map[string][]*transit.MonitoredService)
	hostToHostGroupMap := make(map[string]string)

	for _, r := range results {
		service, err := connectors.BuildServiceForMetric(r.HostName, r.MetricBuilder)
		if err != nil {
			c.JSON(http.StatusInternalServerError, err.Error())
		}
		service.Status = transit.ServiceWarning
		service.LastPluginOutput = helpers.GetLastPluginOutput(r.MetricBuilder.Tags)

		hostToServiceMap[r.HostName] = append(hostToServiceMap[r.HostName], service)
		if r.HostGroupName != "" {
			hostToHostGroupMap[r.HostName] = r.HostGroupName
		}
	}

	monitoredResources := make([]transit.MonitoredResource, 0, len(hostToServiceMap))
	for h, s := range hostToServiceMap {
		resource, err := connectors.CreateResource(h, s)
		if err != nil {
			c.JSON(http.StatusInternalServerError, err.Error())
			return
		}
		monitoredResources = append(monitoredResources, *resource)
	}

	for h, hg := range hostToHostGroupMap {
		resourceRef := connectors.CreateResourceRef(h, "", transit.ResourceTypeHost)
		resourceGroup := connectors.CreateResourceGroup(hg, hg, transit.HostGroup, []transit.ResourceRef{resourceRef})
		groups = append(groups, resourceGroup)
	}

	if err = connectors.SendMetrics(c.Request.Context(), monitoredResources, &groups); err != nil {
		log.Err(err).
			Msg("could not send metrics")
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
