package events

import (
	"encoding/json"
	"fmt"

	"github.com/gwos/tcg/batcher"
	"github.com/gwos/tcg/sdk/transit"
	"github.com/rs/zerolog/log"
)

// EventsBatchBuilder implements builder
type EventsBatchBuilder struct{}

// Build builds the batch payloads if not empty
// splits incoming requests bigger than maxBytes
func (bld *EventsBatchBuilder) Build(buf []batcher.Sized[*transit.GroundworkEventsRequest], maxBytes int) [][]byte {
	// counter, batched request, and accum
	c, bq := 0, transit.GroundworkEventsRequest{}
	qq := make([]transit.GroundworkEventsRequest, 0)

	for _, it := range buf {
		q := it.Value
		if q == nil {
			continue
		}
		if it.Size > maxBytes {
			xxl2qq(&qq, q, it.Size, maxBytes)
			continue
		}

		bq.Events = append(bq.Events, q.Events...)
		c += it.Size
		if c >= maxBytes {
			qq = append(qq, bq)
			c, bq = 0, transit.GroundworkEventsRequest{}
		}
	}

	if len(bq.Events) > 0 {
		qq = append(qq, bq)
	}

	payloads := make([][]byte, 0, len(qq))
	for _, q := range qq {
		p, err := json.Marshal(q)
		if err == nil {
			log.Debug().
				Int("payloadLen", len(p)).
				Msgf("batched %d events", len(q.Events))
			payloads = append(payloads, p)
			continue
		}
		log.Err(err).
			Str("events", fmt.Sprintf("%+v", q)).
			Msg("could not marshal events")
	}
	return payloads
}

func xxl2qq(qq *[]transit.GroundworkEventsRequest, q *transit.GroundworkEventsRequest, size, maxBytes int) {
	/* split big request for parts contained ~lim events */
	cnt := len(q.Events)
	lim := cnt/(size/maxBytes+1) + 1
	log.Debug().Msgf("#EventsBatchBuilder maxBytes/size/cnt/lim %v/%v/%v/%v",
		maxBytes, size, cnt, lim)

	for i1, i2 := 0, lim; i1 < cnt; i1, i2 = i1+lim, i2+lim {
		if i2 > len(q.Events) {
			i2 = len(q.Events)
		}
		*qq = append(*qq, transit.GroundworkEventsRequest{
			Events: q.Events[i1:i2],
		})
	}
}
