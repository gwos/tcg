package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/rs/zerolog/log"

	"github.com/gwos/tcg/connectors/azure/utils"
)

const (
	defaultAzureTenantID       = "***"
	defaultAzureClientID       = "***"
	defaultAzureClientSecret   = "***"
	defaultAzureSubscriptionID = "***"
)

func main() {
	_ = os.Setenv("AZURE_TENANT_ID", defaultAzureTenantID)
	_ = os.Setenv("AZURE_CLIENT_ID", defaultAzureClientID)
	_ = os.Setenv("AZURE_CLIENT_SECRET", defaultAzureClientSecret)

	cred, err := azidentity.NewDefaultAzureCredential(nil)
	if err != nil {
		log.Error().Err(err).Msg("failed to create default azure credential")
		return
	}

	resources, err := utils.ResourcesList(cred, defaultAzureSubscriptionID)
	if err != nil {
		log.Error().Err(err).Msg("failed to get azure resources")
		return
	}

	for _, resource := range resources {
		definitions, err := utils.MetricsDefinitionsList(cred, defaultAzureSubscriptionID, resource)
		if err != nil {
			log.Error().Err(err).
				Str("resource_id", *resource.ID).
				Str("resource_name", *resource.Name).
				Msg("failed to get metrics definitions")
			continue
		}
		metrics, err := utils.MetricsList(cred, defaultAzureSubscriptionID, resource, definitions)
		if err != nil {
			log.Error().Err(err).Msg("failed to get metrics list")
			continue
		}

		b, err := json.Marshal(metrics)
		if err != nil {
			log.Error().Err(err).Msg("failed to marshal metrics")
			return
		}

		fmt.Println(string(b))
	}
}
