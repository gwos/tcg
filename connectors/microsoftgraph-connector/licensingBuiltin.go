package main

import (
	"encoding/json"
	"fmt"
	"github.com/PaesslerAG/jsonpath"
	"github.com/gwos/tcg/transit"
)

const (
	licenseUri = "https://graph.microsoft.com/v1.0/subscribedSkus"
)

// Licensing Built-in - could be data driven
func AddonLicenseMetrics(service *transit.DynamicMonitoredService, token string) error {
	body, err := ExecuteRequest(licenseUri, token)
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

