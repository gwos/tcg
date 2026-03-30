package utils

import (
	"context"
	"errors"
	"fmt"
	"sort"

	ociCom "github.com/oracle/oci-go-sdk/v65/common"
	ociIde "github.com/oracle/oci-go-sdk/v65/identity"
)

var errEmptyRootCompartmentIDName = errors.New("failed to get root compartment ID and Name")

type compartment struct {
	ID   string
	Name string
}

func ListCompartments(ctx context.Context, client ociIde.IdentityClient, tenancyOCID string) ([]compartment, error) {
	rootCompartment, err := client.GetCompartment(ctx, ociIde.GetCompartmentRequest{
		CompartmentId: ociCom.String(tenancyOCID),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get root compartment: %w", err)
	}
	if rootCompartment.Id == nil || rootCompartment.Name == nil {
		return nil, errEmptyRootCompartmentIDName
	}

	seen := make(map[string]compartment)
	seen[tenancyOCID] = compartment{
		ID:   *rootCompartment.Id,
		Name: *rootCompartment.Name,
	}

	var page *string
	for {
		resp, err := client.ListCompartments(ctx, ociIde.ListCompartmentsRequest{
			CompartmentId:          ociCom.String(tenancyOCID),
			CompartmentIdInSubtree: ociCom.Bool(true),
			AccessLevel:            ociIde.ListCompartmentsAccessLevelAny,
			Limit:                  ociCom.Int(1000),
			Page:                   page,
		})
		if err != nil {
			return nil, err
		}

		for _, item := range resp.Items {
			if item.Id == nil || item.Name == nil {
				continue
			}
			if item.LifecycleState != ociIde.CompartmentLifecycleStateActive {
				continue
			}
			seen[*item.Id] = compartment{
				ID:   *item.Id,
				Name: *item.Name,
			}
		}

		if resp.OpcNextPage == nil || *resp.OpcNextPage == "" {
			break
		}
		page = resp.OpcNextPage
	}

	result := make([]compartment, 0, len(seen))
	for _, item := range seen {
		result = append(result, item)
	}
	sort.Slice(result, func(i, j int) bool {
		if result[i].Name == result[j].Name {
			return result[i].ID < result[j].ID
		}
		return result[i].Name < result[j].Name
	})

	return result, nil
}
