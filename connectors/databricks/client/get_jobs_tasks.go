package client

import (
	"encoding/json"
	"fmt"
)

func (d *DatabricksClient) GetJobsTasks(runsIDs []int64) (map[int64][]Task, error) {
	result := make(map[int64][]Task)

	for _, runID := range runsIDs {
		resp, err := d.cl.R().Get(
			fmt.Sprintf(
				"%s?run_id=%d",
				defaultPathJobsRunsGet,
				runID,
			),
		)
		if err != nil {
			return nil, err
		}
		if resp.StatusCode() != 200 {
			return nil, fmt.Errorf("bad status code: %d", resp.StatusCode())
		}

		var runResp RunResponse
		if err = json.Unmarshal(resp.Body(), &runResp); err != nil {
			return nil, err
		}

		result[runResp.JobID] = append(result[runResp.JobID], runResp.Tasks...)
	}

	return result, nil
}
