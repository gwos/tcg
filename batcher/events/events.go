package events

import (
	"encoding/json"
	"fmt"

	"github.com/gwos/tcg/log"
	"github.com/gwos/tcg/transit"
)

// EventsBatchBuilder implements builder
type EventsBatchBuilder struct{}

// Build builds the batch payloads if not empty
func (bld *EventsBatchBuilder) Build(input [][]byte) [][]byte {
	events := make([]transit.GroundworkEvent, 0)
	for _, p := range input {
		r := transit.GroundworkEventsRequest{}
		if err := json.Unmarshal(p, &r); err != nil {
			log.With(log.Fields{
				"error":   err,
				"payload": p,
			}).Log(log.ErrorLevel, "could not unmarshal events payload for batch")
			continue
		}
		events = append(events, r.Events...)
	}

	if len(events) > 0 {
		r := transit.GroundworkEventsRequest{Events: events}
		p, err := json.Marshal(r)
		if err == nil {
			log.Debug(fmt.Sprintf("batched %d events", len(r.Events)))
			return [][]byte{p}
		}
		log.With(log.Fields{
			"error":  err,
			"events": r,
		}).Log(log.ErrorLevel, "could not marshal events")
	}
	return nil
}
