package office

import (
	"encoding/json"

	"github.com/gwos/tcg/sdk/transit"
)

const (
	securityURI = "https://graph.microsoft.com/beta/security/tiIndicators"
)

func SecurityAssessments(service *transit.MonitoredService, token string) (err error) {
	var (
		v    any
		c    int
		body []byte
	)

	if body, err = ExecuteRequest(securityURI, token); err == nil {
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
