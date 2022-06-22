package main

import (
	"fmt"
	"math"
	"time"

	"github.com/gogo/protobuf/proto"
	"github.com/gwos/tcg/sdk/transit"
	dto "github.com/prometheus/client_model/go"
	"github.com/prometheus/common/model"
	"github.com/prometheus/prometheus/prompb"
	"github.com/rs/zerolog/log"
)

type PromParser struct {
	DefaultTags map[string]string
}

func (p *PromParser) Parse(buf []byte, withFilters bool, resource string) (map[string]*dto.MetricFamily, error) {
	var err error
	var metrics = map[string]*dto.MetricFamily{}
	var req prompb.WriteRequest

	if err = proto.Unmarshal(buf, &req); err != nil {
		return nil, fmt.Errorf("unable to unmarshal request body: %s", err)
	}

	now := time.Now()

	availableMetrics[resource] = []string{}
	for _, ts := range req.Timeseries {
		tags := map[string]string{}
		for key, value := range p.DefaultTags {
			tags[key] = value
		}

		labels := make([]*dto.LabelPair, 0)
		for _, l := range ts.Labels {
			tags[l.Name] = l.Value
			labels = append(labels, &dto.LabelPair{Name: &l.Name, Value: &l.Value})
		}

		metricName := tags[model.MetricNameLabel]
		if metricName == "" {
			return nil, fmt.Errorf("metric name %q not found in tag-set or empty", model.MetricNameLabel)
		}
		delete(tags, model.MetricNameLabel)
		availableMetrics[resource] = append(availableMetrics[resource], metricName)

		mf, ok := metrics[metricName]

		if !ok && (withFilters && profileContainsMetric(metricsProfile, metricName) || !withFilters) {
			mf = &dto.MetricFamily{
				Name:   &metricName,
				Type:   dto.MetricType_COUNTER.Enum(),
				Metric: make([]*dto.Metric, 0),
			}
			metrics[metricName] = mf
		}
		for _, s := range ts.Samples {
			stamp := now.Unix()
			if s.Timestamp > 0 {
				stamp = s.Timestamp // time.Unix(0, s.Timestamp*1000000)
			}
			if math.IsNaN(s.Value) {
				log.Error().Msgf("NaN value ignored for %s", metricName)
				continue
			}
			m := dto.Metric{
				Label: labels,
				Counter: &dto.Counter{
					Value: &s.Value,
				},
				TimestampMs: &stamp,
			}
			mf.Metric = append(mf.Metric, &m)
		}
	}

	return metrics, err
}

func profileContainsMetric(profile *transit.MetricsProfile, metric string) bool {
	for _, value := range profile.Metrics {
		if value.Name == metric {
			return true
		}
	}

	return false
}
