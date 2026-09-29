package metrics

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/gwos/tcg/batcher"
	"github.com/gwos/tcg/sdk/transit"
	"github.com/rs/zerolog/log"
)

// MetricsBatchBuilder implements builder
type MetricsBatchBuilder struct{}

// Build builds the batch payloads for HostUnchanged and not empty
// splits incoming requests bigger than maxBytes
func (bld *MetricsBatchBuilder) Build(buf []batcher.Sized[*transit.ResourcesWithServicesRequest], maxBytes int) [][]byte {
	// counter, batched request, and accum
	c, bq, qq := 0, transit.ResourcesWithServicesRequest{}, make([]transit.ResourcesWithServicesRequest, 0)
	flush := func() {
		if len(bq.Resources) > 0 {
			qq = append(qq, bq)
			c, bq = 0, transit.ResourcesWithServicesRequest{}
		}
	}

	for _, it := range buf {
		q := it.Value
		if q == nil {
			continue
		}
		if it.Size > maxBytes {
			// keep the order: put collected requests into accum before the split parts
			flush()
			qq = append(qq, xxl2qq(q, it.Size, maxBytes)...)
			continue
		}

		// in case of not HostUnchanged stop combining, put bq and q into accum
		if hasStatus(q) {
			flush()
			qq = append(qq, *q)
			continue
		}

		bq.SetContext(q.Context)
		bq.Groups = append(bq.Groups, q.Groups...)
		bq.Resources = append(bq.Resources, q.Resources...)
		c += it.Size
		if c >= maxBytes {
			flush()
		}
	}
	flush()

	payloads := make([][]byte, 0, len(qq))
	for _, q := range qq {
		packGroups(&q.Groups)
		p, err := json.Marshal(q)
		if err != nil {
			// a batch combines many requests, drop only the resources and services
			// that cannot be marshaled, e.g. with NaN metric value
			log.Err(err).Msg("could not marshal batched resources, dropping invalid ones")
			dropInvalid(&q)
			if len(q.Resources) == 0 {
				continue
			}
			if p, err = json.Marshal(q); err != nil {
				log.Err(err).
					Str("resources", fmt.Sprintf("%+v", q)).
					Msg("could not marshal resources")
				continue
			}
		}
		log.Debug().
			Int("payloadLen", len(p)).
			Msgf("batched %d resources", len(q.Resources))
		payloads = append(payloads, p)
	}
	return payloads
}

// dropInvalid removes services and resources that cannot be marshaled
func dropInvalid(q *transit.ResourcesWithServicesRequest) {
	resources := make([]transit.MonitoredResource, 0, len(q.Resources))
	for _, res := range q.Resources {
		services := make([]transit.MonitoredService, 0, len(res.Services))
		for _, svc := range res.Services {
			if _, err := json.Marshal(svc); err != nil {
				log.Err(err).
					Str("host", res.Name).
					Str("service", svc.Name).
					Msg("could not marshal service, dropping it")
				continue
			}
			services = append(services, svc)
		}
		// services are checked above, check the resource without them
		res.Services = nil
		if _, err := json.Marshal(res); err != nil {
			log.Err(err).
				Str("host", res.Name).
				Msg("could not marshal resource, dropping it")
			continue
		}
		res.Services = services
		resources = append(resources, res)
	}
	q.Resources = resources
}

func hasStatus(q *transit.ResourcesWithServicesRequest) bool {
	for _, res := range q.Resources {
		if res.Status != transit.HostUnchanged {
			return true
		}
	}
	return false
}

func xxl2qq(q *transit.ResourcesWithServicesRequest, size, maxBytes int) []transit.ResourcesWithServicesRequest {
	/* split big request for parts contained ~lim services */
	cnt := 0
	for _, res := range q.Resources {
		cnt += len(res.Services)
	}
	lim := cnt/(size/maxBytes+1) + 1
	log.Debug().Msgf("#MetricsBatchBuilder maxBytes/size/cnt/lim %v/%v/%v/%v",
		maxBytes, size, cnt, lim)

	qq := make([]transit.ResourcesWithServicesRequest, 0, size/maxBytes+1)
	c, x := 0, transit.ResourcesWithServicesRequest{Groups: q.Groups}
	for _, res := range q.Resources {
		pr := res
		pr.Services = nil

		for _, svc := range res.Services {
			pr.Services = append(pr.Services, svc)
			if c += 1; c < lim {
				continue
			}

			x.Resources = append(x.Resources, pr)
			x.SetContext(q.Context)
			qq = append(qq, x)

			c, x = 0, transit.ResourcesWithServicesRequest{Groups: q.Groups}
			pr = res
			pr.Services = nil
		}

		/* keep the rest of services and resources without services */
		if len(pr.Services) > 0 {
			x.Resources = append(x.Resources, pr)
		} else if len(res.Services) == 0 {
			x.Resources = append(x.Resources, res)
		}
	}

	if len(x.Resources) > 0 {
		x.SetContext(q.Context)
		qq = append(qq, x)
	}

	for i := range qq {
		t := qq[i].Context.TraceToken
		if len(t) > 14 {
			qq[i].Context.TraceToken = fmt.Sprintf("%s-%04d-%s", t[:8], i, t[14:])
		}
	}
	return qq
}

func packGroups(groups *[]transit.ResourceGroup) {
	if len(*groups) == 0 {
		return
	}

	type RG struct {
		transit.ResourceGroup
		resources map[string]transit.ResourceRef
	}

	m := make(map[string]RG)
	for _, g := range *groups {
		gk := strings.Join([]string{string(g.Type), g.GroupName}, ":")
		if _, ok := m[gk]; !ok {
			m[gk] = RG{ResourceGroup: g, resources: make(map[string]transit.ResourceRef)}
		}
		rg := m[gk]
		for _, res := range g.Resources {
			rk := strings.Join([]string{string(res.Type), res.Name}, ":")
			rg.resources[rk] = res
		}
		m[gk] = rg
	}
	*groups = make([]transit.ResourceGroup, 0)

	for _, rg := range m {
		g := rg.ResourceGroup
		if len(rg.resources) > 0 {
			g.Resources = make([]transit.ResourceRef, 0)
			for _, r := range rg.resources {
				g.Resources = append(g.Resources, r)
			}
		}
		*groups = append(*groups, g)
	}
}
