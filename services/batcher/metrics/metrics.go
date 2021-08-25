package metrics

import (
	"encoding/json"
	"time"

	"github.com/gwos/tcg/milliseconds"
	"github.com/gwos/tcg/transit"
	"github.com/rs/zerolog/log"
)

// MetricsBatchBuilder implements builder
type MetricsBatchBuilder struct {
	tContexts []transit.TracerContext
	groupsMap map[string]transit.ResourceGroup
	resMap    map[string]transit.DynamicMonitoredResource
	svcMap    map[string]svcMapItem
}

// NewMetricsBatchBuilder returns new instance
func NewMetricsBatchBuilder() *MetricsBatchBuilder {
	return &MetricsBatchBuilder{
		tContexts: make([]transit.TracerContext, 0),
		groupsMap: make(map[string]transit.ResourceGroup),
		resMap:    make(map[string]transit.DynamicMonitoredResource),
		svcMap:    make(map[string]svcMapItem),
	}
}

// Add adds single transit.DynamicResourcesWithServicesRequest to batch
func (bld *MetricsBatchBuilder) Add(p []byte) {
	r := transit.DynamicResourcesWithServicesRequest{}
	if err := json.Unmarshal(p, &r); err != nil {
		log.Err(err).
			RawJSON("payload", p).
			Msg("could not unmarshal metrics payload for batch")
	} else {
		bld.tContexts = append(bld.tContexts, *r.Context)
		for _, g := range r.Groups {
			bld.groupsMap[string(g.Type)+":"+g.GroupName] = g
		}
		for _, res := range r.Resources {
			resK := string(res.Type) + ":" + res.Name
			for _, svc := range res.Services {
				applyTime(&res, &svc, r.Context.TimeStamp) // ensure time fields
				svcK := string(res.Type) + ":" + res.Name + "::" + string(svc.Type) + ":" + svc.Name
				bld.svcMap[svcK] = svcMapItem{
					resK: resK,
					svcK: svcK,
					svc:  svc,
				}
			}
			applyTime(&res, &transit.DynamicMonitoredService{}, r.Context.TimeStamp) // ensure resource time fields in case of empty services
			res.Services = []transit.DynamicMonitoredService{}
			bld.resMap[resK] = res
		}
	}
}

// Build builds the batch payload if not empty
func (bld *MetricsBatchBuilder) Build() ([]byte, error) {
	r := transit.DynamicResourcesWithServicesRequest{}
	if len(bld.tContexts) > 0 {
		r.Context = &bld.tContexts[0]
	}
	for _, g := range bld.groupsMap {
		r.Groups = append(r.Groups, g)
	}
	for _, svcItem := range bld.svcMap {
		res := bld.resMap[svcItem.resK]
		res.Services = append(res.Services, svcItem.svc)
		bld.resMap[svcItem.resK] = res
	}
	for _, res := range bld.resMap {
		r.Resources = append(r.Resources, res)
	}

	if len(r.Resources) > 0 {
		log.Debug().Msgf("batched %d resources with %d services in %d groups",
			len(r.Resources), len(bld.svcMap), len(bld.groupsMap))
		return json.Marshal(r)
	}
	return nil, nil
}

type svcMapItem struct {
	resK string
	svcK string
	svc  transit.DynamicMonitoredService
}

func applyTime(res *transit.DynamicMonitoredResource,
	svc *transit.DynamicMonitoredService,
	ts milliseconds.MillisecondTimestamp) {

	if ts.IsZero() {
		ts = milliseconds.MillisecondTimestamp{Time: time.Now()}
	}
	switch {
	case res.LastCheckTime.IsZero() && !svc.LastCheckTime.IsZero():
		res.LastCheckTime = svc.LastCheckTime
	case !res.LastCheckTime.IsZero() && svc.LastCheckTime.IsZero():
		svc.LastCheckTime = res.LastCheckTime
	case res.LastCheckTime.IsZero() && svc.LastCheckTime.IsZero():
		res.LastCheckTime = ts
		svc.LastCheckTime = ts
	}
	switch {
	case res.NextCheckTime.IsZero() && !svc.NextCheckTime.IsZero():
		res.NextCheckTime = svc.NextCheckTime
	case !res.NextCheckTime.IsZero() && svc.NextCheckTime.IsZero():
		svc.NextCheckTime = res.NextCheckTime
	case res.NextCheckTime.IsZero() && svc.NextCheckTime.IsZero():
		res.NextCheckTime = ts
		svc.NextCheckTime = ts
	}
}
