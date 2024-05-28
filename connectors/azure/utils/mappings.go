package utils

import (
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/resources/armresources"
	"github.com/gwos/tcg/sdk/mapping"
)

func FilterResources(
	resources []*armresources.GenericResourceExpanded,
	mappings mapping.Mappings,
) []*armresources.GenericResourceExpanded {
	result := make([]*armresources.GenericResourceExpanded, 0)
	for _, resource := range resources {
		if mappings.MatchString(*resource.Name) {
			result = append(result, resource)
		}
	}
	return result
}

func FilterDefinitions(definitions []string, mappings mapping.Mappings) []string {
	result := make([]string, 0)
	for _, definition := range definitions {
		if mappings.MatchString(definition) {
			result = append(result, definition)
		}
	}
	return result
}
