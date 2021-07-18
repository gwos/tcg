package main

import (
	"encoding/json"
	"github.com/gwos/tcg/connectors"
	"github.com/gwos/tcg/transit"
)

const (
	securityUri = "https://graph.microsoft.com/beta/security/tiIndicators"
)
func SecurityAccessments(service *transit.DynamicMonitoredService, token string) error {
	body, err := ExecuteRequest(securityUri, token)
	if err != nil {
		return err
	}
	v := interface{}(nil)
	json.Unmarshal(body, &v)
	count, err := getCount(v)
	if err != nil {
		return err
	}
	metric := createMetricWithThresholds("security-indicators", "", float64(count), 2, 4)
	service.Metrics = append(service.Metrics, *metric)
	service.Status, _ = connectors.CalculateServiceStatus(&service.Metrics)
	return nil
}

