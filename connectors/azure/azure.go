package azure

import (
	"context"
	"os"

	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/rs/zerolog/log"

	"github.com/gwos/tcg/connectors"
	"github.com/gwos/tcg/connectors/azure/utils"
	"github.com/gwos/tcg/sdk/clients"
	"github.com/gwos/tcg/sdk/transit"
)

const (
	defaultHostGroupName        = "TCG-AZURE"
	defaultHostGroupDescription = ""

	envAzureTenantID     = "AZURE_TENANT_ID"
	envAzureClientID     = "AZURE_CLIENT_ID"
	envAzureClientSecret = "AZURE_CLIENT_SECRET"
)

func collectMetrics() {
	if extConfig.AzureTenantID == "" || extConfig.AzureClientID == "" ||
		extConfig.AzureClientSecret == "" || extConfig.AzureSubscriptionID == "" {
		return
	}

	_ = os.Setenv(envAzureTenantID, extConfig.AzureTenantID)
	_ = os.Setenv(envAzureClientID, extConfig.AzureClientID)
	_ = os.Setenv(envAzureClientSecret, extConfig.AzureClientSecret)

	cred, err := azidentity.NewDefaultAzureCredential(nil)
	if err != nil {
		log.Error().Err(err).Msg("failed to create default azure credential")
		return
	}

	allResources, err := utils.ResourcesList(cred, extConfig.AzureSubscriptionID)
	if err != nil {
		log.Error().Err(err).Msg("failed to get azure resources")
		return
	}
	targetResources := utils.FilterResources(allResources, extConfig.GWMapping.Host)

	monitoredResources := make([]transit.MonitoredResource, 0)
	monitoredResourcesRef := make([]transit.ResourceRef, 0)
	for _, resource := range targetResources {
		allDefinitions, err := utils.MetricsDefinitionsList(cred, extConfig.AzureSubscriptionID, resource)
		if err != nil {
			log.Error().Err(err).
				Str("resource_id", *resource.ID).
				Str("resource_name", *resource.Name).
				Msg("failed to get metrics definitions")
			continue
		}
		targetDefinitions := utils.FilterDefinitions(allDefinitions, extConfig.GWMapping.Service)

		metrics, err := utils.MetricsList(cred, extConfig.AzureSubscriptionID, resource, targetDefinitions)
		if err != nil {
			continue
		}

		var services []transit.MonitoredService
		for _, metric := range metrics {
			metricBuilder := utils.CreateMetricBuilder(metric)

			service, err := connectors.BuildServiceForMetric(*resource.Name, metricBuilder)
			if err != nil {
				log.Error().Err(err).Msg("failed to build service for metric")
				continue
			}
			services = append(services, *service)
		}

		if len(services) == 0 {
			continue
		}

		monitoredResource, err := connectors.CreateResource(*resource.Name, services)
		if err != nil {
			log.Error().Err(err).Msg("failed to create resource")
			continue
		}

		monitoredResources = append(monitoredResources, *monitoredResource)
		monitoredResourcesRef = append(
			monitoredResourcesRef,
			connectors.CreateResourceRef(*resource.Name, "", transit.ResourceTypeHost),
		)
	}

	if extConfig.HostGroup == "" {
		extConfig.HostGroup = defaultHostGroupName
	}

	resourceGroups := []transit.ResourceGroup{
		connectors.CreateResourceGroup(extConfig.HostGroup, defaultHostGroupDescription, transit.HostGroup, monitoredResourcesRef),
	}

	ctx := context.Background()
	if extConfig.HostPrefix != "" {
		ctx = clients.CtxWithHeader(ctx, map[string][]string{
			"HostNamePrefix": {extConfig.HostPrefix},
		})
	}

	if err = connectors.SendMetrics(ctx, monitoredResources, &resourceGroups); err != nil {
		log.Error().Err(err).Msg("failed to send metrics")
	}
}
