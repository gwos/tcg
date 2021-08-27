package metrics

import (
	"encoding/json"
	"sort"
	"strings"
	"time"

	"github.com/gwos/tcg/milliseconds"
	"github.com/gwos/tcg/transit"
	"github.com/rs/zerolog/log"
)

type mapItem struct {
	contexts []transit.TracerContext
	groups   []transit.ResourceGroup
	res      []transit.DynamicMonitoredResource
}

// Add adds single transit.DynamicResourcesWithServicesRequest to batch
func add(byGroups map[string]mapItem, p []byte) {
	r := transit.DynamicResourcesWithServicesRequest{}
	if err := json.Unmarshal(p, &r); err != nil {
		log.Err(err).
			RawJSON("payload", p).
			Msg("could not unmarshal metrics payload for batch")
		return
	}

	for i := range r.Resources {
		for j := range r.Resources[i].Services {
			applyTime(&r.Resources[i], &r.Resources[i].Services[j], r.Context.TimeStamp)
		}
		applyTime(&r.Resources[i], &transit.DynamicMonitoredService{}, r.Context.TimeStamp)
	}

	k := makeGKey(r.Groups)
	if item, ok := byGroups[k]; ok {
		item.contexts = append(item.contexts, *r.Context)
		item.res = append(item.res, r.Resources...)
		byGroups[k] = item
	} else {
		byGroups[k] = mapItem{
			contexts: []transit.TracerContext{*r.Context},
			groups:   r.Groups,
			res:      r.Resources,
		}
	}
}

// MetricsBatchBuilder implements builder
type MetricsBatchBuilder struct{}

// Build builds the batch payloads if not empty
func (bld *MetricsBatchBuilder) Build(input [][]byte) [][]byte {
	byGroups := make(map[string]mapItem)
	for _, p := range input {
		add(byGroups, p)
	}

	pp := make([][]byte, len(byGroups))
	for _, item := range byGroups {
		r := transit.DynamicResourcesWithServicesRequest{}
		if len(item.contexts) > 0 {
			r.Context = &item.contexts[0]
		}
		r.Groups = item.groups
		r.Resources = item.res
		if len(r.Resources) > 0 {
			p, err := json.Marshal(r)
			if err == nil {
				log.Debug().Msgf("batched %d resources in %d groups",
					len(r.Resources), len(r.Groups))
				pp = append(pp, p)
				continue
			}
			log.Err(err).
				Interface("resources", r).
				Msg("could not marshal resources")
		}
	}
	return pp
}

func applyTime(
	res *transit.DynamicMonitoredResource,
	svc *transit.DynamicMonitoredService,
	ts milliseconds.MillisecondTimestamp,
) {
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

func makeGKey(gg []transit.ResourceGroup) string {
	keys := make([]string, len(gg))
	for _, g := range gg {
		keys = append(keys, string(g.Type)+":"+g.GroupName)
	}
	sort.Strings(keys)
	return strings.Join(keys, "#")
}
