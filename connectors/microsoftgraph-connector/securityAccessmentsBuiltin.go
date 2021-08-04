package main

import (
	"encoding/json"

	"github.com/gwos/tcg/connectors"
	"github.com/gwos/tcg/transit"
)

const (
	securityUri = "https://graph.microsoft.com/beta/security/tiIndicators"
)

func SecurityAssessments(service *transit.DynamicMonitoredService, token string) (err error) {
	var (
		v    interface{}
		c    int
		body []byte
	)

	if body, err = ExecuteRequest(securityUri, token); err == nil {
		_ = json.Unmarshal(body, &v)
	} else {
		return
	}

	if c, err = getCount(v); err == nil {
		metric := createMetricWithThresholds("security-indicators", "", float64(c), 2, 4)
		service.Metrics = append(service.Metrics, *metric)
		service.Status, _ = connectors.CalculateServiceStatus(&service.Metrics)
	}

	return
}
