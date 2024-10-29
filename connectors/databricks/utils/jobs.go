package utils

import (
	"fmt"
	"slices"
	"time"

	"github.com/rs/zerolog/log"

	"github.com/gwos/tcg/connectors"
	"github.com/gwos/tcg/connectors/databricks/client"
	"github.com/gwos/tcg/sdk/transit"
)

const (
	defaultResourceNameJobs = "Jobs"
)

func GetJobsResource(databricksClient *client.DatabricksClient, from time.Time, to time.Time) (*transit.MonitoredResource, error) {
	jobsRuns, err := databricksClient.GetJobsLatency(from, to)
	if err != nil {
		return nil, fmt.Errorf("failed to get jobs latencies: %w", err)
	}

	var services []transit.MonitoredService
	for _, runs := range jobsRuns {
		if len(runs) > 0 {
			for _, run := range runs {
				output := run.Status.TerminationDetails.Message
				status := transit.ServicePending
				if run.Status.State == "TERMINATED" {
					if run.Status.TerminationDetails.Code != "SUCCESS" {
						status = transit.ServiceUnscheduledCritical
					}
				}

				service0, err := connectors.BuildServiceForMetric(defaultResourceNameJobs, connectors.MetricBuilder{
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
				service0.Status = status
				service0.LastPluginOutput = output

				metricBuilder := connectors.MetricBuilder{
					Name:           "latency_seconds",
					CustomName:     "latency_seconds",
					Value:          run.RunDuration / 1000,
					Graphed:        true,
					StartTimestamp: &transit.Timestamp{Time: time.UnixMilli(run.StartTime)},
					EndTimestamp:   &transit.Timestamp{Time: time.UnixMilli(run.EndTime)},
				}
				service1, err := connectors.BuildServiceForMetric(defaultResourceNameJobs, metricBuilder)
				if err != nil {
					log.Error().Err(err).Str("job_name", run.RunName).Msg("failed to build service for metric")
					continue
				}
				service1.Name = run.RunName
				service1.Status = status
				service1.LastPluginOutput = output

				services = append(services, *service0, *service1)

				if !slices.Contains([]string{"QUEUED", "PENDING", "RUNNING", "TERMINATING"}, run.Status.State) {
					service2, err := connectors.BuildServiceForMetric(defaultResourceNameJobs, connectors.MetricBuilder{
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
					service2.Status = status
					service0.LastPluginOutput = output

					services = append(services, *service2)
				}
			}
		}
	}

	return connectors.CreateResource(defaultResourceNameJobs, services)
}
