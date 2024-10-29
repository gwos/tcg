package client

import (
	"encoding/json"
	"fmt"
	"time"
)

func (d *DatabricksClient) GetJobsLatency(from, to time.Time) (map[int64][]Run, error) {
	var (
		result = make(map[int64][]Run)

		hasMore       = true
		nextPageToken = ""
	)

	for hasMore {
		resp, err := d.cl.R().Get(
			fmt.Sprintf(
				"%s?next_page_token=%s&start_time_from=%d&start_time_to=%d",
				defaultPathJobsRunsList,
				nextPageToken,
				from.UnixMilli(),
				to.UnixMilli(),
			),
		)
		if err != nil {
			return nil, err
		}
		if resp.StatusCode() != 200 {
			return nil, fmt.Errorf("bad status code: %d", resp.StatusCode())
		}

		var runsResp RunsResponse
		if err = json.Unmarshal(resp.Body(), &runsResp); err != nil {
			return nil, err
		}

		for _, run := range runsResp.Runs {
			result[run.JobID] = append(result[run.JobID], run)
		}

		hasMore = runsResp.HasMore
		nextPageToken = runsResp.NextPageToken
	}

	return result, nil
}
