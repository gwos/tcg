package main

import (
	"encoding/json"
	"fmt"
	"net/url"

	"github.com/gwos/tcg/connectors"
	"github.com/gwos/tcg/transit"
)

const baseGraphUri = "https://graph.microsoft.com/v1.0/users/%s/messages?"

// Emails built in
func Emails(service *transit.DynamicMonitoredService, token, outlookEmailAddress string) (err error) {
	var (
		c    int
		body []byte
		v    interface{}
	)

	params := url.Values{
		"$filter": []string{"isRead ne true"},
		"$select": []string{"receivedDateTime"},
	}

	graphUri := fmt.Sprintf(baseGraphUri, outlookEmailAddress) + params.Encode()

	if body, err = ExecuteRequest(graphUri, token); err == nil {
		_ = json.Unmarshal(body, &v)
	} else {
		return
	}

	if c, err = getCount(v); err == nil {
		if definition, ok := contains(metricsProfile.Metrics, "unread.emails"); ok {
			metric := createMetricWithThresholds(
				"unread",
				".emails",
				float64(c),
				float64(definition.WarningThreshold),
				float64(definition.CriticalThreshold),
			)
			service.Metrics = append(service.Metrics, *metric)
			service.Status, _ = connectors.CalculateServiceStatus(&service.Metrics)
		}
	}

	return
}
