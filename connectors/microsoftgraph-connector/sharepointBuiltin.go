package main

import (
	"encoding/json"
	"fmt"
	"github.com/PaesslerAG/jsonpath"
	"github.com/gwos/tcg/connectors"
	"github.com/gwos/tcg/transit"
)

const (
	siteGraphUri        = "https://graph.microsoft.com/v1.0/sites/%s:/drives"
	subSiteGraphUri = "https://graph.microsoft.com/v1.0/sites/%s:/sites/%s:/drives"
)

// SharePoint Drives built-in. Requires Parameters:
// Site
// Subsite
// Challenge: where to store parameters
func SharePoint(
	service *transit.DynamicMonitoredService,
	token string,
	sharePointSite string,
	sharePointSubSite string,
) error {
	var baseGraphUri string
	if len(sharePointSubSite) > 0 {
		baseGraphUri = subSiteGraphUri
	} else {
		baseGraphUri = siteGraphUri
	}
	graphUri := fmt.Sprintf(baseGraphUri, sharePointSite, sharePointSubSite)
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
