package events

import (
	"encoding/json"

	"github.com/gwos/tcg/transit"
	"github.com/rs/zerolog/log"
)

// EventsBatchBuilder implements builder
type EventsBatchBuilder struct{}

// Build builds the batch payloads if not empty
func (bld *EventsBatchBuilder) Build(input [][]byte) [][]byte {
	events := make([]transit.GroundworkEvent, 0)
	for _, p := range input {
		r := transit.GroundworkEventsRequest{}
		if err := json.Unmarshal(p, &r); err != nil {
			log.Err(err).
				RawJSON("payload", p).
				Msg("could not unmarshal events payload for batch")
			continue
		}
		events = append(events, r.Events...)
	}

	if len(events) > 0 {
		r := transit.GroundworkEventsRequest{Events: events}
		p, err := json.Marshal(r)
		if err == nil {
			log.Debug().Msgf("batched %d events", len(r.Events))
			return [][]byte{p}
		}
		log.Err(err).
			Interface("events", r).
			Msg("could not marshal events")
	}
	return nil
}
