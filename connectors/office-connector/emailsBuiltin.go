package main

import (
	"encoding/json"
	"fmt"
	"net/url"

	"github.com/gwos/tcg/sdk/transit"
)

const baseGraphURI = "https://graph.microsoft.com/v1.0/users/%s/messages?"

// Emails built in
func Emails(service *transit.MonitoredService, token, outlookEmailAddress string) (err error) {
	var (
		c    int
		body []byte
		v    interface{}
	)

	params := url.Values{
		"$filter": []string{"isRead ne true"},
		"$select": []string{"receivedDateTime"},
	}

	graphURI := fmt.Sprintf(baseGraphURI, outlookEmailAddress) + params.Encode()

	if body, err = ExecuteRequest(graphURI, token); err == nil {
		_ = json.Unmarshal(body, &v)
	} else {
		return
	}

	if c, err = getCount(v); err == nil {
		if definition, ok := containsMetric(metricsProfile.Metrics, "unread.emails"); ok {
			metric := createMetricWithThresholds(
				"unread",
				".emails",
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
