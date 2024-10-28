package main

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/go-resty/resty/v2"
	"github.com/rs/zerolog/log"
)

const (
	defaultDatabricksURL         = "https://adb-2455086776154666.6.azuredatabricks.net"
	defaultDatabricksAccessToken = "******"
)

type JobRun struct {
	StartTime int64 `json:"start_time"`
	EndTime   int64 `json:"end_time"`
}

func GetJobLatency(token, jobID string) (time.Duration, error) {
	resp, err := resty.New().R().
		SetHeader("Authorization", fmt.Sprintf("Bearer %s", token)).
		Get(fmt.Sprintf("%s/api/2.1/jobs/runs/get?run_id=%s", defaultDatabricksURL, jobID))
	if err != nil {
		return -1, err
	}

	var job JobRun
	if err = json.Unmarshal(resp.Body(), &job); err != nil {
		return -1, err
	}

	return time.Duration(job.EndTime-job.StartTime) * time.Millisecond, nil
}

func main() {
	jobLatency, err := GetJobLatency(defaultDatabricksAccessToken, "144114860174979")
	if err != nil {
		log.Error().Err(err).Msg("failed to get job latency")
	}

	log.Info().Float64("job_latency_seconds", jobLatency.Seconds()).Msg("successfully calculated job latency")
}
