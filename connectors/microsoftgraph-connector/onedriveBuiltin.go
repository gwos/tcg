package main

import (
	"encoding/json"
	"github.com/PaesslerAG/jsonpath"
	"github.com/gwos/tcg/connectors"
	"github.com/gwos/tcg/transit"
)

const oneDriveUri = "https://graph.microsoft.com/v1.0/drive"

// OneDrive built-in. Potentially not a built-in, could be data driven
func OneDrive(service *transit.DynamicMonitoredService, token string) (err error) {
	var (
		body []byte
		v    interface{}
	)
	if body, err = ExecuteRequest(oneDriveUri, token); err == nil {
		_ = json.Unmarshal(body, &v)
	} else {
		return
	}

	total, _ := jsonpath.Get("$.quota.total", v)
	metric1 := createMetric("onedrive.total", "", total.(float64))
	service.Metrics = append(service.Metrics, *metric1)

	remaining, _ := jsonpath.Get("$.quota.remaining", v)
	metric2 := createMetric("onedrive.remaining", "", remaining.(float64))
	service.Metrics = append(service.Metrics, *metric2)

	free := 100 - (total.(float64) / remaining.(float64))
	metric3 := createMetricWithThresholds("onedrive.free", "", free, 15, 5)
	service.Metrics = append(service.Metrics, *metric3)

	service.Status, _ = connectors.CalculateServiceStatus(&service.Metrics)
	return
}
