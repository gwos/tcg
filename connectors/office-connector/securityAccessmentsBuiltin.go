package main

import (
	"encoding/json"

	"github.com/gwos/tcg/sdk/transit"
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
		if definition, ok := containsMetric(metricsProfile.Metrics, "security.indicators"); ok {
			metric := createMetricWithThresholds(
				"security",
				".indicators",
				float64(c),
				float64(definition.WarningThreshold),
				float64(definition.CriticalThreshold),
			)
			service.Metrics = append(service.Metrics, *metric)
			service.Status, _ = transit.CalculateServiceStatus(&service.Metrics)
		}
	}

	return
}
