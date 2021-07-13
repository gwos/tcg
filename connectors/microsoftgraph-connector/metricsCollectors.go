package main

import (
	"encoding/json"
	"fmt"
	"github.com/PaesslerAG/jsonpath"
	"github.com/gwos/tcg/connectors"
	"github.com/gwos/tcg/log"
	"github.com/gwos/tcg/transit"
	"net/url"
)

// OneDrive built-in. Potentially not a built-in, could be data driven
func OneDrive(service *transit.DynamicMonitoredService, token string) error {
	graphUri := "https://graph.microsoft.com/v1.0/drive"
	body, err := ExecuteRequest(graphUri, token)
	if err != nil {
		return err
	}
	v := interface{}(nil)
	json.Unmarshal(body, &v)
	total, _ := jsonpath.Get("$.quota.total", v)
	remaining, _ := jsonpath.Get("$.quota.remaining", v)
	metric1 := createMetric("onedrive.total", "", total.(float64))
	service.Metrics = append(service.Metrics, *metric1)
	metric2 := createMetric("onedrive.remaining", "", remaining.(float64))
	service.Metrics = append(service.Metrics, *metric2)
	free := 100 - (total.(float64) / remaining.(float64))
	metric3 := createMetricWithThresholds("onedrive.free", "", free, 15, 5)
	service.Metrics = append(service.Metrics, *metric3)
	service.Status, _ = connectors.CalculateServiceStatus(&service.Metrics)
	return nil
}

// SharePoint Drives built-in. Requires Parameters:
// Site
// Subsite
// Challenge: where to store parameters
func SharePoint(service *transit.DynamicMonitoredService, token string) error {
	baseGraphUri := "https://graph.microsoft.com/v1.0/sites/%s:/sites/%s:/drives"
	// graphUri := fmt.Sprintf(baseGraphUri, "gwosjoey.sharepoint.com", "GWOS") // TODO: comes from config
	graphUri := fmt.Sprintf(baseGraphUri, sharePointSite, sharePointSubSite) // TODO: comes from config
	body, err := ExecuteRequest(graphUri, token)
	if err != nil {
		return err
	}
	v := interface{}(nil)
	json.Unmarshal(body, &v)
	count := getCount(v)
	for i := 0; i < count; i++ {
		sku1, _ := jsonpath.Get(fmt.Sprintf("$.value[%d].name", i), v)
		total1, _ := jsonpath.Get(fmt.Sprintf("$.value[%d].quota.total", i), v)
		remaining1, _ := jsonpath.Get(fmt.Sprintf("$.value[%d].quota.remaining", i), v)
		total := createMetric(sku1.(string), "-total", total1.(float64))
		remaining := createMetric(sku1.(string), "-remaining", remaining1.(float64))
		free := 100 - (total1.(float64) / remaining1.(float64))
		free1 := createMetricWithThresholds(sku1.(string), "-free", free, 15, 5)
		service.Metrics = append(service.Metrics, *total)
		service.Metrics = append(service.Metrics, *remaining)
		service.Metrics = append(service.Metrics, *free1)
		service.Status, _ = connectors.CalculateServiceStatus(&service.Metrics)
	}
	return nil
}

// Licensing Built-in - could be data driven
func AddonLicenseMetrics(service *transit.DynamicMonitoredService, token string) error {
	graphUri := "https://graph.microsoft.com/v1.0/subscribedSkus"
	body, err := ExecuteRequest(graphUri, token)
	if err != nil {
		return err
	}
	v := interface{}(nil)
	json.Unmarshal(body, &v)
	count := getCount(v)
	for i := 0; i < count; i++ {
		sku, _ := jsonpath.Get(fmt.Sprintf("$.value[%d].skuPartNumber", i), v)
		consumed, _ := jsonpath.Get(fmt.Sprintf("$.value[%d].consumedUnits", i), v)
		prepaid, _ := jsonpath.Get(fmt.Sprintf("$.value[%d].prepaidUnits.enabled", i), v)
		metric1 := createMetric(sku.(string), "-licences-prepaid", prepaid.(float64))
		service.Metrics = append(service.Metrics, *metric1)
		metric2 := createMetric(sku.(string), "-licences-consumed", consumed.(float64))
		service.Metrics = append(service.Metrics, *metric2)
	}
	return nil
}

// Emails built in
func Emails(service *transit.DynamicMonitoredService, token string) error {
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
	count := getCount(v)
	metric := createMetricWithThresholds("unread-emails", "", float64(count), 2, 4)
	service.Metrics = append(service.Metrics, *metric)
	service.Status, _ = connectors.CalculateServiceStatus(&service.Metrics)
	return nil
}

func createMetric(name string, suffix string, value interface{}) *transit.TimeSeries {
	return createMetricWithThresholds(name, suffix, value, -1, -1)
}

func createMetricWithThresholds(name string, suffix string, value interface{}, warning float64, critical float64) *transit.TimeSeries {
	metricBuilder := connectors.MetricBuilder{
		Name:       name + suffix,
		Value:      value,
		UnitType:   transit.UnitCounter,
		Warning:  warning,
		Critical: critical,
		Graphed: true, // TODO: get this value from configs
	}
	metric, err := connectors.BuildMetric(metricBuilder)
	if err != nil {
		log.Error("failed to build metric " + metricBuilder.Name)
		return nil
	}
	return metric
}

func getCount(v interface{}) int {
	var count int = 0
	if v != nil {
		ix, err := jsonpath.Get("$.value[*]", v)
		if err != nil {
			log.Error(fmt.Sprintf("err = %v", err))
		}
		if ix != nil {
			count = len(ix.([]interface{}))
		}
	}
	return count
}

