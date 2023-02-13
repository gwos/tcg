package events

import (
	"encoding/json"

	"github.com/gwos/tcg/sdk/transit"
	"github.com/rs/zerolog/log"
)

// EventsBatchBuilder implements builder
type EventsBatchBuilder struct{}

// Build builds the batch payloads if not empty
// splits incoming payloads bigger than maxBytes
func (bld *EventsBatchBuilder) Build(input [][]byte, maxBytes int) [][]byte {
	// counter, batched request, and accum
	c, bq := 0, transit.GroundworkEventsRequest{}
	qq := make([]transit.GroundworkEventsRequest, 0)
	// oversized payloads
	xxl := make([][]byte, 0)

	for _, p := range input {
		if len(p) > maxBytes {
			xxl = append(xxl, p)
			continue
		}
		q := transit.GroundworkEventsRequest{}
		if err := json.Unmarshal(p, &q); err != nil {
			log.Err(err).
				RawJSON("payload", p).
				Msg("could not unmarshal events payload for batch")
			continue
		}
		bq.Events = append(bq.Events, q.Events...)
		c += len(p)
		if c > maxBytes {
			qq = append(qq, bq)
			c, bq = 0, transit.GroundworkEventsRequest{}
		}
	}
	if len(bq.Events) > 0 {
		qq = append(qq, bq)
	}
	qq = append(qq, xxl2qq(xxl, maxBytes)...)

	output := make([][]byte, 0, len(qq))
	for _, q := range qq {
		p, err := json.Marshal(q)
		if err == nil {
			log.Debug().Msgf("batched %d events", len(q.Events))
			output = append(output, p)
			continue
		}
		log.Err(err).
			Interface("events", q).
			Msg("could not marshal events")
	}
	return output
}

func xxl2qq(input [][]byte, maxBytes int) []transit.GroundworkEventsRequest {
	qq := make([]transit.GroundworkEventsRequest, 0)

	for _, p := range input {
		q := transit.GroundworkEventsRequest{}
		if err := json.Unmarshal(p, &q); err != nil {
			log.Err(err).
				RawJSON("payload", p).
				Msg("could not unmarshal events payload for batch")
			continue
		}

		/* split big payload for parts contained ~lim events */
		cnt := len(q.Events)
		lim := cnt/(len(p)/maxBytes+1) + 1
		log.Debug().Msgf("#EventsBatchBuilder maxBytes:len(p):cnt:lim %v:%v:%v:%v",
			maxBytes, len(p), cnt, lim)

		for i1, i2 := 0, lim; i1 < cnt; i1, i2 = i1+lim, i2+lim {
			if i2 > len(q.Events) {
				i2 = len(q.Events)
			}
			qq = append(qq, transit.GroundworkEventsRequest{
				Events: q.Events[i1:i2],
			})
		}
	}

	return qq
}
