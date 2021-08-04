package main

import (
	"encoding/json"
	"fmt"
	"github.com/PaesslerAG/jsonpath"
	"github.com/gwos/tcg/connectors"
	"github.com/gwos/tcg/transit"
)

const (
	siteGraphUri    = "https://graph.microsoft.com/v1.0/sites/%s:/drives"
	subSiteGraphUri = "https://graph.microsoft.com/v1.0/sites/%s:/sites/%s:/drives"
)

// SharePoint Drives built-in. Requires Parameters:
// Site
// Subsite
// Challenge: where to store parameters
func SharePoint(service *transit.DynamicMonitoredService, token string, sharePointSite string, sharePointSubSite string) (err error) {
	var (
		c       int
		body    []byte
		baseUri string
		v       interface{}
	)
	if len(sharePointSubSite) > 0 {
		baseUri = subSiteGraphUri
	} else {
		baseUri = siteGraphUri
	}
	graphUri := fmt.Sprintf(baseUri, sharePointSite, sharePointSubSite)

	if body, err = ExecuteRequest(graphUri, token); err == nil {
		_ = json.Unmarshal(body, &v)
	} else {
		return
	}

	if c, err = getCount(v); err == nil {
		for i := 0; i < c; i++ {
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
	}

	return
}
