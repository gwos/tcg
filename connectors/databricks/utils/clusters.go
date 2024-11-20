package utils

import (
	"fmt"
	"regexp"

	"github.com/rs/zerolog/log"

	"github.com/gwos/tcg/connectors"
	"github.com/gwos/tcg/connectors/databricks/client"
	"github.com/gwos/tcg/sdk/transit"
)

const (
	defaultServiceNameClusterState = "cluster.state"
)

func GetClustersResource(databricksClient *client.DatabricksClient) ([]transit.MonitoredResource, error) {
	clusters, err := databricksClient.GetClusters()
	if err != nil {
		return nil, fmt.Errorf("failed to get clusters: %w", err)
	}

	result := make([]transit.MonitoredResource, 0, len(clusters))
	for _, cluster := range clusters {
		cluster.Name = regexp.MustCompile(`[^a-zA-Z0-9]+`).ReplaceAllString(cluster.Name, "_")

		service, err := connectors.CreateService(defaultServiceNameClusterState, cluster.Name)
		if err != nil {
			log.Error().Err(err).Str("cluster_name", cluster.Name).Msg("failed to create service")
			continue
		}
		switch cluster.State {
		case "PENDING", "RESTARTING", "RESIZING":
			service.Status = transit.ServicePending
		case "TERMINATING", "TERMINATED", "ERROR": // nolint:goconst
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

		clusterResource, err := connectors.CreateResource(cluster.Name, []transit.MonitoredService{*service})
		if err != nil {
			log.Error().Err(err).Str("cluster_name", cluster.Name).Msgf("failed to create resource for cluster")
			continue
		}
		result = append(result, *clusterResource)
	}

	return result, nil
}
