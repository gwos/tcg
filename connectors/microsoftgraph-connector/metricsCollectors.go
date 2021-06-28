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

func SharePoint(service *transit.DynamicMonitoredService, connector *MicrosoftGraphConnector, cfg *ExtConfig, token string) error {
	graphUri := "https://graph.microsoft.com/v1.0/sites/gwosjoey.sharepoint.com:/sites/GWOS:/drives"
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

	sku1, _ := jsonpath.Get("$.value[0].name", v)
	total1, _ := jsonpath.Get("$.value[0].quota.total", v)
	remaining1, _ := jsonpath.Get("$.value[0].quota.remaining", v)
	total := createMetric(sku1.(string), "-total", total1.(float64))
	remaining := createMetric(sku1.(string), "-remaining", remaining1.(float64))
	free := 100 - (total1.(float64) / remaining1.(float64))
	free1 := createMetricWithThresholds(sku1.(string), "-free", free, 15, 5)
	service.Metrics = append(service.Metrics, *total)
	service.Metrics = append(service.Metrics, *remaining)
	service.Metrics = append(service.Metrics, *free1)

	sku2, _ := jsonpath.Get("$.value[1].name", v)
	total2, _ := jsonpath.Get("$.value[1].quota.total", v)
	remaining2, _ := jsonpath.Get("$.value[1].quota.remaining", v)
	total_ := createMetric(sku2.(string), "-total", total2.(float64))
	remaining_ := createMetric(sku2.(string), "-remaining", remaining2.(float64))
	free_ := 100 - (total2.(float64) / remaining2.(float64))
	free2 := createMetricWithThresholds(sku2.(string), "-free", free_, 15, 5)
	service.Metrics = append(service.Metrics, *total_)
	service.Metrics = append(service.Metrics, *remaining_)
	service.Metrics = append(service.Metrics, *free2)
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
