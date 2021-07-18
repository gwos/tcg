package main

import (
	"encoding/json"
	"fmt"
	"github.com/gwos/tcg/connectors"
	"github.com/gwos/tcg/transit"
	"net/url"
)

// Emails built in
func Emails(
	service *transit.DynamicMonitoredService,
	token string,
	outlookEmailAddress string,
) error {
	baseGraphUri := "https://graph.microsoft.com/v1.0/users/%s/messages?"
	params := url.Values{}
	params.Add("$filter", "isRead ne true")
	params.Add("$select", "receivedDateTime")
	graphUri := fmt.Sprintf(baseGraphUri, outlookEmailAddress) + params.Encode()
	body, err := ExecuteRequest(graphUri, token)
	if err != nil {
		return err
	}
	v := interface{}(nil)
	json.Unmarshal(body, &v)
	count, err := getCount(v)
	if err != nil {
		return err
	}
	// TODO: wire in thresholds
	metric := createMetricWithThresholds("unread-emails", "", float64(count), 2, 4)
	service.Metrics = append(service.Metrics, *metric)
	service.Status, _ = connectors.CalculateServiceStatus(&service.Metrics)
	return nil
}

