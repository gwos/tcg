package office

import (
	"encoding/json"

	"github.com/PaesslerAG/jsonpath"
	"github.com/gwos/tcg/sdk/transit"
)

const oneDriveURI = "https://graph.microsoft.com/v1.0/drive"

// OneDrive built-in. Potentially not a built-in, could be data driven
func OneDrive(service *transit.MonitoredService, token string) (err error) {
	var (
		body []byte
		v    any
	)
	if body, err = ExecuteRequest(oneDriveURI, token); err == nil {
		_ = json.Unmarshal(body, &v)
	} else {
		return
	}

	total, _ := jsonpath.Get("$.quota.total", v)

	if definition, ok := containsMetric(metricsProfile.Metrics, "onedrive.total"); ok {
		metric1 := createMetricWithThresholds(
			"onedrive",
			".total",
			total.(float64),
			float64(definition.WarningThreshold),
			float64(definition.CriticalThreshold),
		)
		service.Metrics = append(service.Metrics, *metric1)
	}

	remaining, _ := jsonpath.Get("$.quota.remaining", v)
	if definition, ok := containsMetric(metricsProfile.Metrics, "onedrive.remaining"); ok {
		metric2 := createMetricWithThresholds(
			"onedrive",
			".remaining",
			remaining.(float64),
			float64(definition.WarningThreshold),
			float64(definition.CriticalThreshold),
		)
		service.Metrics = append(service.Metrics, *metric2)
	}

	if definition, ok := containsMetric(metricsProfile.Metrics, "onedrive"); ok {
		free := 100 - (total.(float64) / remaining.(float64))
		metric3 := createMetricWithThresholds(
			"onedrive",
			".free",
			free,
			float64(definition.WarningThreshold),
			float64(definition.CriticalThreshold),
		)
		service.Metrics = append(service.Metrics, *metric3)
	}

	service.Status, _ = transit.CalculateServiceStatus(&service.Metrics)
	return
}
