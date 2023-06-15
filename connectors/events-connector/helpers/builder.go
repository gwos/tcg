package helpers

import (
	"github.com/gwos/tcg/connectors"
	"github.com/gwos/tcg/sdk/transit"
	"github.com/prometheus/alertmanager/template"
)

const (
	tagKeySummary     = "summary"
	tagKeyDescription = "description"
)

func GetMetricBuildersFromPrometheusData(data template.Data, cfg *ExtConfig) (string, []connectors.MetricBuilder, error) {
	var (
		err            error
		hostName       string
		serviceName    string
		metricBuilders = make([]connectors.MetricBuilder, 0)
	)

	hostName, err = cfg.HostMappings.Apply(data.CommonLabels)
	if err != nil || hostName == "" {
		return "", nil, err
	}

	for _, alert := range data.Alerts {
		serviceName, err = cfg.ServiceMappings.Apply(alert.Labels)
		if err != nil || serviceName == "" {
			continue
		}

		mb := connectors.MetricBuilder{
			Name:           serviceName,
			Graphed:        false,
			Tags:           alert.Labels,
			UnitType:       transit.UnitCounter,
			ComputeType:    transit.Informational,
			StartTimestamp: &transit.Timestamp{Time: alert.StartsAt},
			EndTimestamp:   &transit.Timestamp{Time: alert.EndsAt},
		}

		for k, v := range alert.Annotations {
			mb.Tags[k] = v
		}

		metricBuilders = append(metricBuilders, mb)
	}

	return hostName, metricBuilders, nil
}

func GetLastPluginOutput(tag map[string]string) string {
	var result string
	if description, ok := tag[tagKeySummary]; ok {
		result += description
	}
	if summary, ok := tag[tagKeyDescription]; ok {
		if len(result) > 0 {
			result += " | "
		}
		result += summary
	}
	return result
}
