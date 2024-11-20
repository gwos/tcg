package client

import (
	"encoding/json"
	"fmt"
)

func (d *DatabricksClient) GetClusters() ([]Cluster, error) {
	var (
		result        = make([]Cluster, 0)
		hasMore       = true
		nextPageToken = ""
	)

	for hasMore {
		resp, err := d.cl.R().Get(
			fmt.Sprintf(
				"%s?next_page_token=%s",
				defaultPathClustersList,
				nextPageToken,
			),
		)
		if err != nil {
			return nil, err
		}
		if resp.StatusCode() != 200 {
			return nil, fmt.Errorf("bad status code: %d", resp.StatusCode())
		}

		var clustersResp ClustersResponse
		if err = json.Unmarshal(resp.Body(), &clustersResp); err != nil {
			return nil, err
		}

		if clustersResp.Clusters != nil {
			result = append(result, clustersResp.Clusters...)
		}

		hasMore = clustersResp.HasMore
		nextPageToken = clustersResp.NextPageToken
	}

	return result, nil
}
