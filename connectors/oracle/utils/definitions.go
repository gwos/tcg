package utils

import (
	"context"
	"sort"
	"strings"

	ociCom "github.com/oracle/oci-go-sdk/v65/common"
	ociMon "github.com/oracle/oci-go-sdk/v65/monitoring"
)

const (
	defaultPageLimit = 1000
)

var excludedServiceDefinitions = map[string]struct{}{
	"maintenance_status":          {},
	"instance_status":             {},
	"instanceaccessibilitystatus": {},
	"instancefilesystemstatus":    {},
	"backupfailure":               {},
	"backupsize":                  {},
	"backuptime":                  {},
}

type definition struct {
	Namespace  string
	Name       string
	Dimensions map[string]string
}

func ListDefinitions(ctx context.Context, client ociMon.MonitoringClient, compartmentID string) ([]definition, error) {
	unique := make(map[string]definition)

	var page *string
	for {
		resp, err := client.ListMetrics(ctx, ociMon.ListMetricsRequest{
			CompartmentId: ociCom.String(compartmentID),
			Page:          page,
			Limit:         ociCom.Int(defaultPageLimit),
		})
		if err != nil {
			return nil, err
		}

		for _, item := range resp.Items {
			if item.Namespace == nil || item.Name == nil {
				continue
			}
			namespace := strings.TrimSpace(*item.Namespace)
			name := strings.TrimSpace(*item.Name)
			if namespace == "" || name == "" {
				continue
			}
			if isExcludedServiceDefinition(name) {
				continue
			}

			key := namespace + "\x00" + name
			if _, exists := unique[key]; !exists {
				unique[key] = definition{
					Namespace:  namespace,
					Name:       name,
					Dimensions: item.Dimensions,
				}
			}
		}

		if resp.OpcNextPage == nil || *resp.OpcNextPage == "" {
			break
		}
		page = resp.OpcNextPage
	}

	result := make([]definition, 0, len(unique))
	for _, item := range unique {
		result = append(result, item)
	}
	sort.Slice(result, func(i, j int) bool {
		if result[i].Namespace == result[j].Namespace {
			return result[i].Name < result[j].Name
		}
		return result[i].Namespace < result[j].Namespace
	})

	return result, nil
}

func isExcludedServiceDefinition(name string) bool {
	_, exists := excludedServiceDefinitions[strings.ToLower(strings.TrimSpace(name))]
	return exists
}
