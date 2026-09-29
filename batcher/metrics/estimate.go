package metrics

import (
	"github.com/gwos/tcg/sdk/transit"
)

// Approximate JSON overhead of fixed parts, in bytes.
// Keys, punctuation and formatted numbers are counted here,
// variable-length strings are added separately.
const (
	sizeRequest    = 30 // context, resources keys and brackets
	sizeContext    = 80 // context keys and timeStamp
	sizeGroup      = 45 // groupName, type, resources keys
	sizeRef        = 22 // name, type keys
	sizeBaseInfo   = 22 // name, type keys
	sizeProperties = 16 // properties key and brackets
	sizeServices   = 14 // services key and brackets
	sizeMonInfo    = 12 // status key
	sizeTimestamp  = 32 // key and quoted millis
	sizeTypedValue = 30 // valueType and value keys
	sizeProperty   = 4  // key quotes and separators
	sizeMetric     = 32 // metricName, sampleType keys
	sizeInterval   = 67 // interval with endTime and startTime
	sizeValue      = 9  // value key
	sizeThresholds = 15 // thresholds key and brackets
	sizeThreshold  = 36 // sampleType, label, value keys
	sizeTag        = 6  // quotes and separators
	sizeFieldQuote = 14 // key and quotes for an optional string field
)

// EstimateSize returns the approximate size of the JSON-serialized request.
// It is cheap enough to be called on the caller's thread, no serialization is done.
func EstimateSize(q *transit.ResourcesWithServicesRequest) int {
	if q == nil {
		return 0
	}
	n := sizeRequest + sizeContext + len(q.Context.AppType) + len(q.Context.AgentID) +
		len(q.Context.TraceToken) + len(q.Context.Version)
	for i := range q.Groups {
		g := &q.Groups[i]
		n += sizeGroup + len(g.GroupName) + len(g.Type) + len(g.Description)
		for j := range g.Resources {
			r := &g.Resources[j]
			n += sizeRef + len(r.Name) + len(r.Type) + len(r.Owner)
		}
	}
	for i := range q.Resources {
		res := &q.Resources[i]
		n += estimateBaseInfo(&res.BaseInfo) + estimateMonInfo(&res.MonitoredInfo)
		n += optStr(res.Device) + sizeServices
		for j := range res.Services {
			svc := &res.Services[j]
			n += estimateBaseInfo(&svc.BaseInfo) + estimateMonInfo(&svc.MonitoredInfo)
			for k := range svc.Metrics {
				n += estimateMetric(&svc.Metrics[k])
			}
		}
	}
	return n
}

func estimateBaseInfo(p *transit.BaseInfo) int {
	n := sizeBaseInfo + len(p.Name) + len(p.Type) +
		optStr(p.Owner) + optStr(p.Category) + optStr(p.Description)
	if len(p.Properties) > 0 {
		n += sizeProperties
	}
	for k, v := range p.Properties {
		n += sizeProperty + len(k) + estimateTypedValue(&v)
	}
	return n
}

func estimateMonInfo(p *transit.MonitoredInfo) int {
	n := sizeMonInfo + len(p.Status) + optStr(p.LastPluginOutput)
	if p.LastCheckTime != nil {
		n += sizeTimestamp
	}
	if p.NextCheckTime != nil {
		n += sizeTimestamp
	}
	return n
}

func estimateMetric(p *transit.TimeSeries) int {
	n := sizeMetric + len(p.MetricName) + len(p.SampleType) + optStr(string(p.Unit))
	if p.Interval != nil {
		n += sizeInterval
	}
	if p.Value != nil {
		n += sizeValue + estimateTypedValue(p.Value)
	}
	for k, v := range p.Tags {
		n += sizeTag + len(k) + len(v)
	}
	if len(p.Thresholds) > 0 {
		n += sizeThresholds
	}
	for i := range p.Thresholds {
		t := &p.Thresholds[i]
		n += sizeThreshold + len(t.Label) + len(t.SampleType)
		if t.Value != nil {
			n += estimateTypedValue(t.Value)
		}
	}
	return n
}

func estimateTypedValue(p *transit.TypedValue) int {
	n := sizeTypedValue + len(p.ValueType)
	switch {
	case p.StringValue != nil:
		n += len(*p.StringValue) + 2
	case p.TimeValue != nil:
		n += 15
	case p.DoubleValue != nil:
		n += 8
	case p.IntegerValue != nil:
		n += 3
	case p.BoolValue != nil:
		n += 5
	}
	return n
}

func optStr(s string) int {
	if s == "" {
		return 0
	}
	return sizeFieldQuote + len(s)
}
