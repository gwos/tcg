package main

import (
	"encoding/json"
	"github.com/PaesslerAG/jsonpath"
	"github.com/gwos/tcg/connectors"
	"github.com/gwos/tcg/log"
	"github.com/gwos/tcg/transit"
	"io/ioutil"
	"net/http"
)

func MicrosoftDrive(service *transit.DynamicMonitoredService, connector *MicrosoftGraphConnector, cfg *ExtConfig, token string) error {
	graphUri := "https://graph.microsoft.com/v1.0/drive"
	request, _ := http.NewRequest("GET", graphUri, nil)
	request.Header.Set("accept", "application/json; odata.metadata=full")
	request.Header.Set("Authorization", "Bearer "+token)
	response, error := httpClient.Do(request)
	if error != nil {
		return error
	}
	if response.StatusCode != 200 { // TODO: refactor, push this down, make it reusable
		log.Error("[MSGraph Connector]:  Retrying Authentication...")
		connector.officeToken = ""
		connector.graphToken = ""
		connector.Initialize(*cfg)
		request.Header.Set("Authorization", "Bearer "+connector.officeToken)
		response, error = httpClient.Do(request)
		if error != nil {
			return error
		}
	}
	body, _ := ioutil.ReadAll(response.Body)
	defer response.Body.Close()
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
	return nil
}

func AddonLicenseMetrics(service *transit.DynamicMonitoredService, connector *MicrosoftGraphConnector, cfg *ExtConfig, token string) error {
	graphUri := "https://graph.microsoft.com/v1.0/subscribedSkus"
	request, _ := http.NewRequest("GET", graphUri, nil)
	request.Header.Set(	"accept", "application/json; odata.metadata=full")
	request.Header.Set("Authorization", "Bearer " + token)
	response, error := httpClient.Do(request)
	if error != nil {
		return error
 	}
	if response.StatusCode != 200 {
		log.Error("[MSGraph Connector]:  Retrying Authentication...")
		connector.officeToken = ""
		connector.graphToken = ""
		connector.Initialize(*cfg)
		request.Header.Set("Authorization", "Bearer " + connector.officeToken)
		response, error = httpClient.Do(request)
		if error != nil {
			return error
		}
	}
	body, _ := ioutil.ReadAll(response.Body)
	defer response.Body.Close()
	v := interface{}(nil)
	json.Unmarshal(body, &v)
	sku, _ := jsonpath.Get("$.value[0].skuPartNumber", v)
	consumed, _ := jsonpath.Get("$.value[0].consumedUnits", v)
	prepaid, _ := jsonpath.Get("$.value[0].prepaidUnits.enabled", v)
	metric1 := createMetric(sku.(string), "-licences-prepaid", prepaid.(float64))
	service.Metrics = append(service.Metrics, *metric1)
	metric2 := createMetric(sku.(string), "-licences-consumed", consumed.(float64))
	service.Metrics = append(service.Metrics, *metric2)

	sku, _ = jsonpath.Get("$.value[1].skuPartNumber", v)
	consumed, _ = jsonpath.Get("$.value[1].consumedUnits", v)
	prepaid, _ = jsonpath.Get("$.value[1].prepaidUnits.enabled", v)
	metric3 := createMetric(sku.(string), "-licences-prepaid", prepaid.(float64))
	service.Metrics = append(service.Metrics, *metric3)
	metric4 := createMetric(sku.(string), "-licences-consumed", consumed.(float64))
	service.Metrics = append(service.Metrics, *metric4)

	// fmt.Printf("sku = %s, consumed = %f, prepaid =%f\n", sku.(string), consumed.(float64), prepaid.(float64))
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
