package utils

import (
	"context"
	"fmt"

	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/monitor/armmonitor"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/resources/armresources"
)

func MetricsDefinitionsList(
	cred *azidentity.DefaultAzureCredential,
	subscriptionID string,
	resource *armresources.GenericResourceExpanded,
) ([]string, error) {
	client, err := armmonitor.NewMetricDefinitionsClient(subscriptionID, cred, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to init client: %w", err)
	}
	pager := client.NewListPager(*resource.ID, nil)

	var result []string
	for pager.More() {
		page, err := pager.NextPage(context.Background())
		if err != nil {
			break
		}

		for _, definition := range page.Value {
			result = append(result, *definition.Name.Value)
		}
	}

	return result, nil
}
