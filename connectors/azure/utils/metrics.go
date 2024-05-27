package utils

import (
	"context"
	"fmt"

	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/monitor/armmonitor"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/resources/armresources"
)

func MetricsList(
	cred *azidentity.DefaultAzureCredential,
	subscriptionID string,
	resource *armresources.GenericResourceExpanded,
	definitions []string,
) ([]armmonitor.MetricsClientListResponse, error) {
	client, err := armmonitor.NewMetricsClient(subscriptionID, cred, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to init client: %w", err)
	}

	result := make([]armmonitor.MetricsClientListResponse, 0)
	for _, definition := range definitions {
		metric, err := client.List(context.Background(), *resource.ID, &armmonitor.MetricsClientListOptions{
			Metricnames: &definition,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to list metrics: %w", err)
		}
		if len(metric.Value) == 0 || len(metric.Value[0].Timeseries) == 0 ||
			len(metric.Value[0].Timeseries[0].Data) == 0 {
			continue
		}
		result = append(result, metric)
	}

	return result, nil
}
