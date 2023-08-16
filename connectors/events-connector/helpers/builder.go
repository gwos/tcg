package helpers

import (
	"github.com/gwos/tcg/connectors"
	"github.com/gwos/tcg/sdk/transit"
	"github.com/prometheus/alertmanager/template"
)

type ParseResult struct {
	HostName      string
	HostGroupName string
	MetricBuilder connectors.MetricBuilder
}

const (
	tagKeySummary     = "summary"
	tagKeyDescription = "description"
)

func ParsePrometheusData(data template.Data, cfg *ExtConfig) ([]ParseResult, error) {
	results := make([]ParseResult, 0)

	for _, alert := range data.Alerts {
		hostName, err := cfg.HostMappings.Apply(alert.Labels)
		if err != nil || hostName == "" {
			return nil, err
		}

		hostGroupName, err := cfg.HostGroupMappings.Apply(data.CommonLabels)
		if err != nil {
			return nil, err
		}

		serviceName, err := cfg.ServiceMappings.Apply(alert.Labels)
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
			Value:          -1,
		}

		for k, v := range alert.Annotations {
			mb.Tags[k] = v
		}

		results = append(results, ParseResult{
			HostName:      hostName,
			HostGroupName: hostGroupName,
			MetricBuilder: mb,
		})
	}

	return results, nil
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
