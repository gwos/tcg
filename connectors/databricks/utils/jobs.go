package utils

import (
	"fmt"
	"maps"
	"regexp"
	"slices"
	"time"

	"github.com/rs/zerolog/log"

	"github.com/gwos/tcg/connectors"
	"github.com/gwos/tcg/connectors/databricks/client"
	"github.com/gwos/tcg/sdk/transit"
)

const (
	defaultServiceNameJobRunDuration  = "job.runs.duration"
	defaultServiceNameJobTaskDuration = "job.tasks.duration"
)

func GetJobsResources(databricksClient *client.DatabricksClient, from time.Time, to time.Time) ([]transit.MonitoredResource, error) {
	result := make([]transit.MonitoredResource, 0)

	jobsRuns, err := databricksClient.GetJobsRuns(from, to)
	if err != nil {
		return nil, fmt.Errorf("failed to get jobs runs: %w", err)
	}

	jobsTasks, err := databricksClient.GetJobsTasks(getJobsRunsIDs(jobsRuns))
	if err != nil {
		return nil, fmt.Errorf("failed to get jobs tasks: %w", err)
	}

	for _, jobID := range slices.Collect(maps.Keys(jobsRuns)) {
		runs, ok := jobsRuns[jobID]
		if !ok || len(runs) == 0 {
			continue
		}

		resourceName := regexp.MustCompile(`[^a-zA-Z0-9]+`).ReplaceAllString(runs[0].RunName, "_")

		services := make([]transit.MonitoredService, 0)

		runsServices, err := getRunsServices(runs)
		if err != nil {
			return nil, fmt.Errorf("failed to init runs services: %w", err)
		}
		services = append(services, runsServices...)

		tasksServices, err := getTasksServices(resourceName, jobsTasks[jobID])
		if err != nil {
			return nil, fmt.Errorf("failed to init tasks services: %w", err)
		}
		services = append(services, tasksServices...)

		jobResource, err := connectors.CreateResource(resourceName, services)
		if err != nil {
			log.Error().Err(err).Str("job_name", resourceName).Msgf("failed to create resource for job")
			continue
		}
		result = append(result, *jobResource)
	}

	return result, nil
}

func getRunsServices(runs []client.Run) ([]transit.MonitoredService, error) {
	services := make([]transit.MonitoredService, 0)

	for _, run := range runs {
		run.RunName = regexp.MustCompile(`[^a-zA-Z0-9]+`).ReplaceAllString(run.RunName, "_")

		output := run.Status.TerminationDetails.Message
		status := transit.ServicePending
		if run.Status.State == "TERMINATED" {
			if run.Status.TerminationDetails.Code != "SUCCESS" {
				status = transit.ServiceUnscheduledCritical
			}
		}

		service0, err := connectors.BuildServiceForMetric(run.RunName, connectors.MetricBuilder{
			Name:           defaultServiceNameJobRunDuration,
			CustomName:     defaultServiceNameJobRunDuration,
			Value:          0,
			Graphed:        true,
			StartTimestamp: &transit.Timestamp{Time: time.UnixMilli(run.StartTime).Add(-2 * time.Second)},
			EndTimestamp:   &transit.Timestamp{Time: time.UnixMilli(run.StartTime).Add(-time.Second)},
		})
		if err != nil {
			log.Error().Err(err).Str("job_name", run.RunName).Msg("failed to build service for metric")
			continue
		}
		service0.Status = status
		service0.LastPluginOutput = output

		service1, err := connectors.BuildServiceForMetric(run.RunName, connectors.MetricBuilder{
			Name:           defaultServiceNameJobRunDuration,
			CustomName:     defaultServiceNameJobRunDuration,
			Value:          run.RunDuration / 1000,
			Graphed:        true,
			StartTimestamp: &transit.Timestamp{Time: time.UnixMilli(run.StartTime)},
			EndTimestamp:   &transit.Timestamp{Time: time.UnixMilli(run.EndTime)},
		})
		if err != nil {
			log.Error().Err(err).Str("job_name", run.RunName).Msg("failed to build service for metric")
			continue
		}
		service1.Status = status
		service1.LastPluginOutput = output

		services = append(services, *service0, *service1)

		if !slices.Contains([]string{"QUEUED", "PENDING", "RUNNING", "TERMINATING"}, run.Status.State) {
			service2, err := connectors.BuildServiceForMetric(run.RunName, connectors.MetricBuilder{
				Name:           defaultServiceNameJobRunDuration,
				CustomName:     defaultServiceNameJobRunDuration,
				Value:          0,
				Graphed:        true,
				StartTimestamp: &transit.Timestamp{Time: time.UnixMilli(run.EndTime).Add(time.Second)},
				EndTimestamp:   &transit.Timestamp{Time: time.UnixMilli(run.EndTime).Add(2 * time.Second)},
			})
			if err != nil {
				log.Error().Err(err).Str("job_name", run.RunName).Msg("failed to build service for metric")
				continue
			}
			service2.Status = status
			service0.LastPluginOutput = output

			services = append(services, *service2)
		}
	}

	return services, nil
}

func getTasksServices(jobName string, tasks []client.Task) ([]transit.MonitoredService, error) {
	var services []transit.MonitoredService

	for _, task := range tasks {
		task.TaskKey = regexp.MustCompile(`[^a-zA-Z0-9]+`).ReplaceAllString(task.TaskKey, "_")

		output := task.Status.TerminationDetails.Message
		status := transit.ServicePending
		if task.Status.State == "TERMINATED" {
			if task.Status.TerminationDetails.Code != "SUCCESS" {
				status = transit.ServiceUnscheduledCritical
			}
		}

		service0, err := connectors.BuildServiceForMetric(jobName, connectors.MetricBuilder{
			Name:           task.TaskKey,
			CustomName:     task.TaskKey,
			Value:          0,
			Graphed:        true,
			StartTimestamp: &transit.Timestamp{Time: time.UnixMilli(task.StartTime).Add(-2 * time.Second)},
			EndTimestamp:   &transit.Timestamp{Time: time.UnixMilli(task.EndTime).Add(-time.Second)},
		})
		if err != nil {
			log.Error().Err(err).Str("task_name", task.TaskKey).Msg("failed to build service for metric")
			continue
		}
		service0.Name = defaultServiceNameJobTaskDuration
		service0.Status = status
		service0.LastPluginOutput = output

		service1, err := connectors.BuildServiceForMetric(jobName, connectors.MetricBuilder{
			Name:           task.TaskKey,
			CustomName:     task.TaskKey,
			Value:          (task.EndTime - task.StartTime) / 1000,
			Graphed:        true,
			StartTimestamp: &transit.Timestamp{Time: time.UnixMilli(task.StartTime)},
			EndTimestamp:   &transit.Timestamp{Time: time.UnixMilli(task.EndTime)},
		})
		if err != nil {
			log.Error().Err(err).Str("task_name", task.TaskKey).Msg("failed to build service for metric")
			continue
		}
		service1.Name = defaultServiceNameJobTaskDuration
		service1.Status = status
		service1.LastPluginOutput = output

		if !slices.Contains([]string{"QUEUED", "PENDING", "RUNNING", "TERMINATING"}, task.Status.State) {
			service2, err := connectors.BuildServiceForMetric(jobName, connectors.MetricBuilder{
				Name:           task.TaskKey,
				CustomName:     task.TaskKey,
				Value:          0,
				Graphed:        true,
				StartTimestamp: &transit.Timestamp{Time: time.UnixMilli(task.EndTime).Add(time.Second)},
				EndTimestamp:   &transit.Timestamp{Time: time.UnixMilli(task.EndTime).Add(2 * time.Second)},
			})
			if err != nil {
				log.Error().Err(err).Str("task_name", task.TaskKey).Msg("failed to build service for metric")
				continue
			}
			service2.Name = defaultServiceNameJobTaskDuration
			service2.Status = status
			service2.LastPluginOutput = output

			services = append(services, *service2)
		}
	}

	return services, nil
}

func getJobsRunsIDs(jobsRuns map[int64][]client.Run) []int64 {
	result := make([]int64, 0)
	for _, jobRuns := range jobsRuns {
		for _, run := range jobRuns {
			result = append(result, run.RunID)
		}
	}
	return result
}
