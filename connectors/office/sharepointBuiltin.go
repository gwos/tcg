package office

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/PaesslerAG/jsonpath"
	"github.com/gwos/tcg/sdk/transit"
	"github.com/rs/zerolog/log"
)

const (
	siteGraphURI    = "https://graph.microsoft.com/v1.0/sites/%s:/drives"
	subSiteGraphURI = "https://graph.microsoft.com/v1.0/sites/%s:/sites/%s:/drives"
)

// SharePoint Drives built-in. Requires Parameters:
// Site
// Subsite
// Challenge: where to store parameters
func SharePoint(service *transit.MonitoredService, token, sharePointSite, sharePointSubSite string) (err error) {
	var (
		c       int
		body    []byte
		baseURI string
		v       any
	)
	if len(sharePointSubSite) > 0 {
		baseURI = subSiteGraphURI
	} else {
		baseURI = siteGraphURI
	}
	graphURI := fmt.Sprintf(baseURI, sharePointSite, sharePointSubSite)

	if body, err = ExecuteRequest(graphURI, token); err == nil {
		_ = json.Unmarshal(body, &v)
	} else {
		log.Error().Msgf("%v", err)
		return
	}

	if c, err = getCount(v); err == nil {
		for i := 0; i < c; i++ {
			sku1, _ := jsonpath.Get(fmt.Sprintf("$.value[%d].name", i), v)
			totalValue, _ := jsonpath.Get(fmt.Sprintf("$.value[%d].quota.total", i), v)
			remainingValue, _ := jsonpath.Get(fmt.Sprintf("$.value[%d].quota.remaining", i), v)
			freeValue := 100 - (totalValue.(float64) / remainingValue.(float64))

			if definition, ok := containsMetric(metricsProfile.Metrics, "sharepoint.total"); ok {
				total := createMetricWithThresholds(
					strings.ToLower(strings.ReplaceAll(sku1.(string), " ", ".")),
					".total",
					totalValue.(float64),
					float64(definition.WarningThreshold),
					float64(definition.CriticalThreshold),
				)
				service.Metrics = append(service.Metrics, *total)
			}

			if definition, ok := containsMetric(metricsProfile.Metrics, "sharepoint.remaining"); ok {
				remaining := createMetricWithThresholds(
					strings.ToLower(strings.ReplaceAll(sku1.(string), " ", ".")),
					".remaining",
					remainingValue.(float64),
					float64(definition.WarningThreshold),
					float64(definition.CriticalThreshold),
				)
				service.Metrics = append(service.Metrics, *remaining)
			}

			if definition, ok := containsMetric(metricsProfile.Metrics, "sharepoint.free"); ok {
				free := createMetricWithThresholds(
					strings.ToLower(strings.ReplaceAll(sku1.(string), " ", ".")),
					".free",
					freeValue,
					float64(definition.WarningThreshold),
					float64(definition.CriticalThreshold),
				)
				service.Metrics = append(service.Metrics, *free)
			}

			service.Status, _ = transit.CalculateServiceStatus(&service.Metrics)
		}
	}

	return
}
