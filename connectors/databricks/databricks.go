package databricks

import (
	"context"
	"fmt"
	"slices"
	"time"

	"github.com/rs/zerolog/log"

	"github.com/gwos/tcg/connectors"
	"github.com/gwos/tcg/connectors/databricks/client"
	"github.com/gwos/tcg/sdk/transit"
)

const (
	defaultHostGroupName        = "DATABRICKS"
	defaultHostGroupDescription = ""
)

func collectMetrics() {
	if extConfig.DatabricksURL == "" || extConfig.DatabricksAccessToken == "" {
		log.Debug().
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
		return
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
		currentActiveJobsRuns = make(map[int64]int64)
	)

	jobsRuns, err := databricksClient.GetJobsLatency(from, to)
	if err != nil {
		return nil, fmt.Errorf("failed to get jobs latencies: %w", err)
	}

	var services []transit.MonitoredService
	for _, runs := range jobsRuns {
		if len(runs) > 0 {
			for _, run := range runs {
				if !slices.Contains([]string{"QUEUED", "PENDING", "RUNNING", "TERMINATING"}, run.Status.State) {
					service0, err := connectors.BuildServiceForMetric(hostName, connectors.MetricBuilder{
						Name:           "latency_seconds",
						CustomName:     "latency_seconds",
						Value:          0,
						Graphed:        true,
						StartTimestamp: &transit.Timestamp{Time: time.UnixMilli(run.StartTime).Add(-2 * time.Minute)},
						EndTimestamp:   &transit.Timestamp{Time: time.UnixMilli(run.StartTime).Add(-time.Minute)},
					})
					if err != nil {
						log.Error().Err(err).Str("job_name", run.RunName).Msg("failed to build service for metric")
						continue
					}
					service0.Name = run.RunName

					metricBuilder := connectors.MetricBuilder{
						Name:           "latency_seconds",
						CustomName:     "latency_seconds",
						Value:          run.RunDuration / 1000,
						Graphed:        true,
						StartTimestamp: &transit.Timestamp{Time: time.UnixMilli(run.StartTime)},
						EndTimestamp:   &transit.Timestamp{Time: time.UnixMilli(run.EndTime)},
					}
					service1, err := connectors.BuildServiceForMetric(hostName, metricBuilder)
					if err != nil {
						log.Error().Err(err).Str("job_name", run.RunName).Msg("failed to build service for metric")
						continue
					}
					service1.Name = run.RunName

					service2, err := connectors.BuildServiceForMetric(hostName, connectors.MetricBuilder{
						Name:           "latency_seconds",
						CustomName:     "latency_seconds",
						Value:          0,
						Graphed:        true,
						StartTimestamp: &transit.Timestamp{Time: time.UnixMilli(run.EndTime).Add(time.Minute)},
						EndTimestamp:   &transit.Timestamp{Time: time.UnixMilli(run.EndTime).Add(2 * time.Minute)},
					})
					if err != nil {
						log.Error().Err(err).Str("job_name", run.RunName).Msg("failed to build service for metric")
						continue
					}
					service2.Name = run.RunName

					services = append(services, *service0, *service1, *service2)
				} else {
					currentActiveJobsRuns[run.JobID] = run.RunID
				}
			}
		}
	}

	activeJobsRuns = currentActiveJobsRuns

	return connectors.CreateResource(hostName, services)
}
