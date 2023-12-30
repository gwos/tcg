package metrics

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/gwos/tcg/sdk/transit"
	"github.com/rs/zerolog/log"
)

// MetricsBatchBuilder implements builder
type MetricsBatchBuilder struct{}

// Build builds the batch payloads for HostUnchanged and not empty
// splits incoming payloads bigger than maxBytes
func (bld *MetricsBatchBuilder) Build(input [][]byte, maxBytes int) [][]byte {
	// counter, batched request, and accum
	c, bq := 0, transit.ResourcesWithServicesRequest{}
	qq := make([]transit.ResourcesWithServicesRequest, 0)

	for _, p := range input {
		if len(p) > maxBytes {
			qq = append(qq, xxl2qq(p, maxBytes)...)
			continue
		}

		q := transit.ResourcesWithServicesRequest{}
		if err := json.Unmarshal(p, &q); err != nil {
			log.Err(err).
				RawJSON("payload", p).
				Msg("could not unmarshal metrics payload for batch")
			continue
		}

		// in case of not HostUnchanged stop combining, put bq and q into accum
		if hasStatus(q) {
			if len(bq.Resources) > 0 {
				qq = append(qq, bq)
				c, bq = 0, transit.ResourcesWithServicesRequest{}
			}
			qq = append(qq, q)
			continue
		}

		bq.SetContext(*q.Context)
		bq.Groups = append(bq.Groups, q.Groups...)
		bq.Resources = append(bq.Resources, q.Resources...)
		c += len(p)
		if c >= maxBytes {
			qq = append(qq, bq)
			c, bq = 0, transit.ResourcesWithServicesRequest{}
		}
	}
	if len(bq.Resources) > 0 {
		qq = append(qq, bq)
	}

	output := make([][]byte, 0, len(qq))
	for _, q := range qq {
		q.Groups = packGroups(q.Groups)
		p, err := json.Marshal(q)
		if err == nil {
			log.Debug().Msgf("batched %d resources", len(q.Resources))
			output = append(output, p)
			continue
		}
		log.Err(err).
			Interface("resources", q).
			Msg("could not marshal resources")
	}
	return output
}

func hasStatus(q transit.ResourcesWithServicesRequest) bool {
	for _, res := range q.Resources {
		if res.Status != transit.HostUnchanged {
			return true
		}
	}
	return false
}

func xxl2qq(p []byte, maxBytes int) []transit.ResourcesWithServicesRequest {
	qq := make([]transit.ResourcesWithServicesRequest, 0)
	q := transit.ResourcesWithServicesRequest{}
	if err := json.Unmarshal(p, &q); err != nil {
		log.Err(err).
			RawJSON("payload", p).
			Msg("could not unmarshal metrics payload for batch")
		return qq
	}

	/* split big payload for parts contained ~lim metrics */
	cnt := 0
	for _, res := range q.Resources {
		for _, svc := range res.Services {
			cnt += len(svc.Metrics)
		}
	}
	lim := cnt/(len(p)/maxBytes+1) + 1
	log.Debug().Msgf("#MetricsBatchBuilder maxBytes:len(p):cnt:lim %v:%v:%v:%v",
		maxBytes, len(p), cnt, lim)

	// counter and accum
	c, rr := 0, make([]transit.MonitoredResource, 0)
	for _, res := range q.Resources {
		pr := res
		pr.Services = make([]transit.MonitoredService, 0)

		for i, svc := range res.Services {
			pr.Services = append(pr.Services, svc)
			c += len(svc.Metrics)

			if c >= lim && i < len(res.Services)-1 {
				rr = append(rr, pr)
				x := transit.ResourcesWithServicesRequest{
					Groups:    q.Groups,
					Resources: rr,
				}
				x.SetContext(*q.Context)
				qq = append(qq, x)

				c, rr = 0, make([]transit.MonitoredResource, 0)
				pr = res
				pr.Services = make([]transit.MonitoredService, 0)
			}
		}
		rr = append(rr, pr)
	}

	if len(rr) > 0 {
		x := transit.ResourcesWithServicesRequest{
			Groups:    q.Groups,
			Resources: rr,
		}
		x.SetContext(*q.Context)
		qq = append(qq, x)
	}

	for i := range qq {
		t := qq[i].Context.TraceToken
		if len(t) > 14 {
			t = fmt.Sprintf("%s-%04d-%s", t[:8], i, t[14:])
			qq[i].Context.TraceToken = t
		}
	}
	return qq
}

func packGroups(input []transit.ResourceGroup) []transit.ResourceGroup {
	if len(input) == 0 {
		return nil
	}

	type RG struct {
		transit.ResourceGroup
		resources map[string]transit.ResourceRef
	}

	m := make(map[string]RG)
	for _, g := range input {
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

	output := make([]transit.ResourceGroup, 0)
	for _, rg := range m {
		g := rg.ResourceGroup
		if len(rg.resources) > 0 {
			g.Resources = make([]transit.ResourceRef, 0)
			for _, r := range rg.resources {
				g.Resources = append(g.Resources, r)
			}
		}
		output = append(output, g)
	}

	return output
}
