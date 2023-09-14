package helpers

import (
	"github.com/gwos/tcg/connectors"
	"github.com/gwos/tcg/sdk/transit"
	"github.com/prometheus/alertmanager/template"
	"github.com/rs/zerolog/log"
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
		tags := make(map[string]string,
			len(data.CommonAnnotations)+len(alert.Annotations)+len(data.CommonLabels)+len(alert.Labels))
		for k, v := range data.CommonAnnotations {
			tags[k] = v
		}
		for k, v := range alert.Annotations {
			tags[k] = v
		}
		for k, v := range data.CommonLabels {
			tags[k] = v
		}
		for k, v := range alert.Labels {
			tags[k] = v
		}

		hostGroupName, err := cfg.HostGroupMappings.Apply(tags)
		if err != nil {
			log.Debug().Err(err).Interface("tags", tags).Send()
			continue
		}
		hostName, err := cfg.HostMappings.Apply(tags)
		if err != nil || hostName == "" {
			log.Debug().Err(err).Interface("tags", tags).Send()
			continue
		}
		serviceName, err := cfg.ServiceMappings.Apply(tags)
		if err != nil || serviceName == "" {
			log.Debug().Err(err).Interface("tags", tags).Send()
			continue
		}

		mb := connectors.MetricBuilder{
			Name:           serviceName,
			Graphed:        false,
			Tags:           tags,
			UnitType:       transit.UnitCounter,
			ComputeType:    transit.Informational,
			StartTimestamp: transit.NewTimestamp(),
			EndTimestamp:   transit.NewTimestamp(),
			Value:          -1,
		}
		if !alert.StartsAt.IsZero() {
			mb.StartTimestamp.Time = alert.StartsAt.UTC()
			mb.EndTimestamp.Time = alert.StartsAt.UTC()
		}
		if !alert.EndsAt.IsZero() {
			mb.EndTimestamp.Time = alert.EndsAt.UTC()
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
