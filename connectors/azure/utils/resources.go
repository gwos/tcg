package utils

import (
	"context"
	"fmt"

	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/resources/armresources"
)

func ResourcesList(cred *azidentity.DefaultAzureCredential, subscriptionID string) ([]*armresources.GenericResourceExpanded, error) {
	client, err := armresources.NewClient(subscriptionID, cred, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to init client: %w", err)
	}
	pager := client.NewListPager(nil)

	var result []*armresources.GenericResourceExpanded
	for pager.More() {
		page, err := pager.NextPage(context.Background())
		if err != nil {
			return nil, fmt.Errorf("failed to retrieve next page: %w", err)
		}

		result = append(result, page.Value...)
	}

	return result, nil
}
