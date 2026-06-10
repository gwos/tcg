package utils

import (
	"context"
	"sort"
	"strings"

	ociCom "github.com/oracle/oci-go-sdk/v65/common"
	ociSearch "github.com/oracle/oci-go-sdk/v65/resourcesearch"
)

type Resource struct {
	OCID           string
	DisplayName    string
	LifecycleState string
	CompartmentID  string
	ResourceType   string
}

func ListResources(ctx context.Context, client ociSearch.ResourceSearchClient) (map[string]Resource, error) {
	inventory := make(map[string]Resource)

	var page *string
	for {
		resp, err := client.SearchResources(ctx, ociSearch.SearchResourcesRequest{
			SearchDetails: ociSearch.StructuredSearchDetails{
				Query: ociCom.String("query all resources"),
			},
			Limit: ociCom.Int(defaultPageLimit),
			Page:  page,
		})
		if err != nil {
			return nil, err
		}

		for _, item := range resp.Items {
			if item.Identifier == nil {
				continue
			}
			ocid := strings.TrimSpace(*item.Identifier)
			if ocid == "" {
				continue
			}

			displayName := ocid
			if item.DisplayName != nil && strings.TrimSpace(*item.DisplayName) != "" {
				displayName = strings.TrimSpace(*item.DisplayName)
			}

			res := Resource{
				OCID:        ocid,
				DisplayName: displayName,
			}
			if item.LifecycleState != nil {
				res.LifecycleState = strings.TrimSpace(*item.LifecycleState)
			}
			if item.CompartmentId != nil {
				res.CompartmentID = strings.TrimSpace(*item.CompartmentId)
			}
			if item.ResourceType != nil {
				res.ResourceType = strings.TrimSpace(*item.ResourceType)
			}

			inventory[ocid] = res
		}

		if resp.OpcNextPage == nil || *resp.OpcNextPage == "" {
			break
		}
		page = resp.OpcNextPage
	}

	return inventory, nil
}

func SortedResources(inventory map[string]Resource) []Resource {
	result := make([]Resource, 0, len(inventory))
	for _, r := range inventory {
		result = append(result, r)
	}
	sort.Slice(result, func(i, j int) bool {
		if result[i].DisplayName == result[j].DisplayName {
			return result[i].OCID < result[j].OCID
		}
		return result[i].DisplayName < result[j].DisplayName
	})
	return result
}
