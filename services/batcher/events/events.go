package events

import (
	"encoding/json"

	"github.com/gwos/tcg/transit"
	"github.com/rs/zerolog/log"
)

// EventsBatchBuilder implements builder
type EventsBatchBuilder struct {
	events []transit.GroundworkEvent
}

// NewEventsBatchBuilder returns new instance
func NewEventsBatchBuilder() *EventsBatchBuilder {
	return &EventsBatchBuilder{
		events: make([]transit.GroundworkEvent, 0),
	}
}

// Add adds single transit.GroundworkEventsRequest to batch
func (bld *EventsBatchBuilder) Add(p []byte) {
	r := transit.GroundworkEventsRequest{}
	if err := json.Unmarshal(p, &r); err != nil {
		log.Err(err).
			RawJSON("payload", p).
			Msg("could not unmarshal events payload for batch")
		return
	}
	bld.events = append(bld.events, r.Events...)
}

// Build builds the batch payloads if not empty
func (bld *EventsBatchBuilder) Build() [][]byte {
	r := transit.GroundworkEventsRequest{
		Events: bld.events,
	}
	if len(r.Events) > 0 {
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
