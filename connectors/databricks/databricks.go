package databricks

import (
	"context"
	"fmt"
	"slices"
	"time"

	"github.com/gwos/tcg/connectors"
	"github.com/gwos/tcg/connectors/databricks/client"
	"github.com/gwos/tcg/sdk/transit"
	"github.com/rs/zerolog/log"
)

const (
	defaultHostGroupName        = "DATABRICKS"
	defaultHostGroupDescription = ""
)

func collectMetrics() {
	if extConfig.DatabricksURL == "" || extConfig.DatabricksAccessToken == "" {
		log.Error().
			Str("databricks_url", extConfig.DatabricksURL).
			Str("databricks_access_token", extConfig.DatabricksAccessToken).
			Msg("databricks auth data is missing")
		return
	}

	databricksClient := client.New(extConfig.DatabricksURL, extConfig.DatabricksAccessToken)

	from := lastRunTimeTo
	to := time.Now()

	monitoredResources := make([]transit.MonitoredResource, 0)
	monitoredResourcesRef := make([]transit.ResourceRef, 0)

	jobsResource, err := getJobsResources(databricksClient, from, to)
	if err != nil {
		log.Error().Err(err).
			Str("databricks_url", extConfig.DatabricksURL).
			Str("databricks_access_token", extConfig.DatabricksAccessToken).
			Msg("failed to get jobs resource")
	}

	monitoredResources = append(monitoredResources, *jobsResource)
	monitoredResourcesRef = append(
		monitoredResourcesRef,
		connectors.CreateResourceRef(jobsResource.Name, "", transit.ResourceTypeHost),
	)

	lastRunTimeTo = to

	if extConfig.HostGroup == "" {
		extConfig.HostGroup = defaultHostGroupName
	}

	resourceGroups := []transit.ResourceGroup{
		connectors.CreateResourceGroup(extConfig.HostGroup, defaultHostGroupDescription, transit.HostGroup, monitoredResourcesRef),
	}

	if err = connectors.SendMetrics(context.Background(), monitoredResources, &resourceGroups); err != nil {
		log.Error().Err(err).Msg("failed to send metrics")
	}
}

func getJobsResources(databricksClient *client.DatabricksClient, from time.Time, to time.Time) (*transit.MonitoredResource, error) {
	var (
		hostName              = "jobs"
		currentActiveJobsRuns = make(map[string]string)
	)

	jobsRuns, err := databricksClient.GetJobsLatency(from, to)
	if err != nil {
		return nil, fmt.Errorf("failed to get jobs latencies: %w", err)
	}

	var services []transit.MonitoredService
	for _, runs := range jobsRuns {
		if len(runs) > 0 {
			var metricBuilders []connectors.MetricBuilder
			for _, run := range runs {
				if !slices.Contains([]string{"QUEUED", "PENDING", "RUNNING", "TERMINATING"}, run.Status.State) {
					metricBuilders = append(metricBuilders, connectors.MetricBuilder{
						Name:           "run_latency_seconds",
						CustomName:     "run_latency_seconds",
						Value:          run.RunDuration,
						UnitType:       transit.UnitCounter,
						StartTimestamp: &transit.Timestamp{Time: time.UnixMilli(run.StartTime)},
						EndTimestamp:   &transit.Timestamp{Time: time.UnixMilli(run.EndTime)},
					})
				} else {
					currentActiveJobsRuns[run.JobID] = run.RunID
				}
			}
			service, err := connectors.BuildServiceForMultiMetric(hostName, runs[0].RunName, runs[0].RunName, metricBuilders)
			if err != nil {
				log.Error().Err(err).Str("job_name", runs[0].RunName).Msg("failed to build service for multi metric")
				continue
			}
			services = append(services, *service)
		}
	}

	activeJobsRuns = currentActiveJobsRuns

	return connectors.CreateResource(hostName, services)
}
