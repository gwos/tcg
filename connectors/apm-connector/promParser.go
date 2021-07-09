package main

import (
	"fmt"
	"github.com/gogo/protobuf/proto"
	"github.com/gwos/tcg/log"
	"github.com/pkg/errors"
	dto "github.com/prometheus/client_model/go"
	"github.com/prometheus/common/model"
	"github.com/prometheus/prometheus/prompb"
	"math"
	"time"
)

type PromParser struct {
	DefaultTags map[string]string
}

func (p *PromParser) Parse(buf []byte) (map[string]*dto.MetricFamily, error) {
	var err error
	var metrics = map[string]*dto.MetricFamily{}
	var req prompb.WriteRequest

	if err := proto.Unmarshal(buf, &req); err != nil {
		return nil, fmt.Errorf("unable to unmarshal request body: %s", err)
	}

	now := time.Now()

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

		mf, ok := metrics[metricName]
		if !ok {
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
				log.Error("Nan Value ignored for ", metricName)
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

func (p *PromParser) parseDebug(buf []byte) (interface{}, error) {
	var req prompb.WriteRequest
	if err := proto.Unmarshal(buf, &req); err != nil {
		return nil, fmt.Errorf("unable to unmarshal request body: %s", err)
	}
	for _, ts := range req.Timeseries {
		log.Debug("----- time series ----")
		for _, l := range ts.Labels {
			log.Debug(fmt.Sprintf("\t%s = %s\n", l.GetName(), l.GetValue()))
		}
		for _, s := range ts.Samples {
			log.Debug(fmt.Sprintf("\t %f, %d\n", s.GetValue(), s.GetTimestamp()))
		}
	}
	return nil, errors.New("testing")
}
