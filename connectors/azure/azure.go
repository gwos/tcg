package azure

import (
	"context"
	"encoding/json"
	"os"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/gwos/tcg/connectors"
	"github.com/gwos/tcg/connectors/azure/utils"
	"github.com/gwos/tcg/sdk/transit"
	"github.com/rs/zerolog/log"
)

const (
	envAzureTenantID     = "AZURE_TENANT_ID"
	envAzureClientID     = "AZURE_CLIENT_ID"
	envAzureClientSecret = "AZURE_CLIENT_SECRET"
)

type ExtConfig struct {
	AzureTenantID       string `json:"azureTenantId"`
	AzureClientID       string `json:"azureClientId"`
	AzureClientSecret   string `json:"azureClientSecret"`
	AzureSubscriptionID string `json:"azureSubscriptionId"`

	Ownership     transit.HostOwnershipType `json:"ownership,omitempty"`
	CheckInterval time.Duration             `json:"checkIntervalMinutes"`
}

// UnmarshalJSON implements json.Unmarshaler.
func (cfg *ExtConfig) UnmarshalJSON(input []byte) error {
	type plain ExtConfig
	c := plain(*cfg)
	if err := json.Unmarshal(input, &c); err != nil {
		return err
	}
	if c.CheckInterval != cfg.CheckInterval {
		c.CheckInterval = c.CheckInterval * time.Minute
	}
	*cfg = ExtConfig(c)
	return nil
}

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

	resources, err := utils.ResourcesList(cred, extConfig.AzureSubscriptionID)
	if err != nil {
		log.Error().Err(err).Msg("failed to get azure resources")
		return
	}

	monitoredResources := make([]transit.MonitoredResource, 0)
	for _, resource := range resources {
		definitions, err := utils.MetricsDefinitionsList(cred, extConfig.AzureSubscriptionID, resource)
		if err != nil {
			log.Error().Err(err).
				Str("resource_id", *resource.ID).
				Str("resource_name", *resource.Name).
				Msg("failed to get metrics definitions")
			continue
		}

		metrics, err := utils.MetricsList(cred, extConfig.AzureSubscriptionID, resource, definitions)
		if err != nil {
			continue
		}

		var services []transit.MonitoredService
		for _, metric := range metrics {
			value := metric.Value[0].Timeseries[0].Data[len(metric.Value[0].Timeseries[0].Data)-1].Average
			if value == nil {
				var zero float64 = 0
				value = &zero
			}
			mb := connectors.MetricBuilder{
				Name:       *metric.Value[0].Name.Value,
				CustomName: *metric.Value[0].Name.Value,
				Value:      value,
				UnitType:   transit.UnitCounter,
			}
			service, err := connectors.BuildServiceForMetric(*resource.Name, mb)
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
	}

	if err = connectors.SendMetrics(context.Background(), monitoredResources, nil); err != nil {
		log.Error().Err(err).Msg("failed to send metrics")
	}
}
