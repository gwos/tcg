package utils

import (
	"fmt"

	"github.com/rs/zerolog/log"

	"github.com/gwos/tcg/connectors"
	"github.com/gwos/tcg/connectors/databricks/client"
	"github.com/gwos/tcg/sdk/transit"
)

const (
	defaultResourceNameClusters = "Clusters"
)

func GetClustersResource(databricksClient *client.DatabricksClient) (*transit.MonitoredResource, error) {
	clusters, err := databricksClient.GetClusters()
	if err != nil {
		return nil, fmt.Errorf("failed to get clusters: %w", err)
	}

	services := make([]transit.MonitoredService, 0, len(clusters))
	for _, cluster := range clusters {
		service, err := connectors.CreateService(cluster.Name, defaultResourceNameClusters)
		if err != nil {
			log.Error().Err(err).Str("cluster_name", cluster.Name).Msg("failed to create service")
			continue
		}
		switch cluster.State {
		case "PENDING", "RESTARTING", "RESIZING":
			service.Status = transit.ServicePending
		case "TERMINATING", "TERMINATED", "ERROR":
			service.Status = transit.ServiceUnscheduledCritical
		case "UNKNOWN":
			service.Status = transit.ServiceUnknown
		default:
			service.Status = transit.ServiceOk
		}

		if service.Status != transit.ServiceOk {
			if cluster.StateMessage != "" {
				service.LastPluginOutput = cluster.StateMessage
			} else {
				service.LastPluginOutput = fmt.Sprintf(
					"Termination reason: %s, %s",
					cluster.TerminationReason.Code,
					cluster.TerminationReason.Type,
				)
			}
		}

		services = append(services, *service)
	}

	return connectors.CreateResource(defaultResourceNameClusters, services)
}
