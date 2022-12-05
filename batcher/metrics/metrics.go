package metrics

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/gwos/tcg/sdk/transit"
	"github.com/rs/zerolog/log"
)

const lastPluginOutputLimit = 254
const lastPluginOutputReplacement = "...<shortened>"

type mapItem struct {
	contexts []transit.TracerContext
	groups   []transit.ResourceGroup
	res      []transit.MonitoredResource
}

// Add adds single transit.ResourcesWithServicesRequest to batch
func add(byGroups map[string]mapItem, p []byte) {
	r := transit.ResourcesWithServicesRequest{}
	if err := json.Unmarshal(p, &r); err != nil {
		log.Err(err).
			RawJSON("payload", p).
			Msg("could not unmarshal metrics payload for batch")
		return
	}

	for i := range r.Resources {
		for j := range r.Resources[i].Services {
			applyTime(&r.Resources[i], &r.Resources[i].Services[j], r.Context.TimeStamp)
			applyLastPluginOutputLimit(&r.Resources[i], &r.Resources[i].Services[j])
		}
		applyTime(&r.Resources[i], &transit.MonitoredService{}, r.Context.TimeStamp)
		applyLastPluginOutputLimit(&r.Resources[i], &transit.MonitoredService{})
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
		r := transit.ResourcesWithServicesRequest{}
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
	res *transit.MonitoredResource,
	svc *transit.MonitoredService,
	ts *transit.Timestamp,
) {
	isT := func(ts *transit.Timestamp) bool { return ts != nil && !ts.IsZero() }
	if !isT(ts) {
		ts = transit.NewTimestamp()
	}
	switch {
	case !isT(res.LastCheckTime) && isT(svc.LastCheckTime):
		res.LastCheckTime = svc.LastCheckTime
	case isT(res.LastCheckTime) && !isT(svc.LastCheckTime):
		svc.LastCheckTime = res.LastCheckTime
	case !isT(res.LastCheckTime) && !isT(svc.LastCheckTime):
		res.LastCheckTime = ts
		svc.LastCheckTime = ts
	}
	switch {
	case !isT(res.NextCheckTime) && isT(svc.NextCheckTime):
		res.NextCheckTime = svc.NextCheckTime
	case isT(res.NextCheckTime) && !isT(svc.NextCheckTime):
		svc.NextCheckTime = res.NextCheckTime
	case !isT(res.NextCheckTime) && !isT(svc.NextCheckTime):
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

func applyLastPluginOutputLimit(res *transit.MonitoredResource, svc *transit.MonitoredService) {
	if len(res.LastPluginOutput) > lastPluginOutputLimit {
		res.LastPluginOutput = fmt.Sprintf(
			"%s%s",
			res.LastPluginOutput[:lastPluginOutputLimit-len(lastPluginOutputReplacement)],
			lastPluginOutputReplacement,
		)
	}
	if len(svc.LastPluginOutput) > lastPluginOutputLimit {
		svc.LastPluginOutput = fmt.Sprintf(
			"%s%s",
			svc.LastPluginOutput[:lastPluginOutputLimit-len(lastPluginOutputReplacement)],
			lastPluginOutputReplacement,
		)
	}
}
