package databricks

import (
	"context"
	"time"

	"github.com/gwos/tcg/connectors/databricks/utils"
	"github.com/rs/zerolog/log"

	"github.com/gwos/tcg/connectors"
	"github.com/gwos/tcg/connectors/databricks/client"
	"github.com/gwos/tcg/sdk/transit"
)

const (
	defaultHostGroupName        = "DATABRICKS"
	defaultHostGroupDescription = ""
)

func collectMetrics() {
	if extConfig.DatabricksURL == "" || extConfig.DatabricksAccessToken == "" {
		log.Debug().
			Str("databricks_url", extConfig.DatabricksURL).
			Str("databricks_access_token", extConfig.DatabricksAccessToken).
			Msg("databricks auth data is missing")
		return
	}

	databricksClient := client.New(extConfig.DatabricksURL, extConfig.DatabricksAccessToken)

	from := lastRunTimeTo
	to := time.Now()

	monitoredResources := make([]transit.MonitoredResource, 0)
	monitoredResourcesRef := make([]transit.ResourceRef, 0)

	jobsResource, err := utils.GetJobsResource(databricksClient, from, to)
	if err != nil {
		log.Error().Err(err).
			Str("databricks_url", extConfig.DatabricksURL).
			Str("databricks_access_token", extConfig.DatabricksAccessToken).
			Msg("failed to get jobs resource")
		return
	}

	clusterResource, err := utils.GetClustersResource(databricksClient)
	if err != nil {
		log.Error().Err(err).
			Str("databricks_url", extConfig.DatabricksURL).
			Str("databricks_access_token", extConfig.DatabricksAccessToken).
			Msg("failed to get clusters resource")
		return
	}

	monitoredResources = append(monitoredResources, *jobsResource, *clusterResource)
	monitoredResourcesRef = append(
		monitoredResourcesRef,
		connectors.CreateResourceRef(jobsResource.Name, "", transit.ResourceTypeHost),
		connectors.CreateResourceRef(clusterResource.Name, "", transit.ResourceTypeHost),
	)

	lastRunTimeTo = to

	if extConfig.HostGroup == "" {
		extConfig.HostGroup = defaultHostGroupName
	}

	resourceGroups := []transit.ResourceGroup{
		connectors.CreateResourceGroup(extConfig.HostGroup, defaultHostGroupDescription, transit.HostGroup, monitoredResourcesRef),
	}

	if err = connectors.SendMetrics(context.Background(), monitoredResources, &resourceGroups); err != nil {
		log.Error().Err(err).Msg("failed to send metrics")
	}
}
