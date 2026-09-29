package nats

import (
	"strconv"
	"testing"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/stretchr/testify/assert"
)

func TestPubKeepsOrder(t *testing.T) {
	pub, maxPayload := s.pub, s.config.MaxPayload
	t.Cleanup(func() { s.pub, s.config.MaxPayload = pub, maxPayload })
	// a small channel read slowly, so most messages wait in the overflow buffer
	s.pub, s.config.MaxPayload = make(chan *nats.Msg, 2), 1024

	const n = 200
	done := make(chan []string)
	go func() {
		got := make([]string, 0, n)
		for len(got) < n {
			msg := <-s.pub
			got = append(got, string(msg.Data))
			if len(got)%20 == 0 {
				time.Sleep(time.Millisecond)
			}
		}
		done <- got
	}()

	expected := make([]string, 0, n)
	for i := range n {
		expected = append(expected, strconv.Itoa(i))
		assert.NoError(t, Pub("tcg.test", []byte(expected[i]), nil))
	}

	select {
	case got := <-done:
		assert.Equal(t, expected, got)
	case <-time.After(5 * time.Second):
		t.Fatal("messages were not delivered")
	}
	s.pubMu.Lock()
	defer s.pubMu.Unlock()
	assert.Empty(t, s.pubOverflow)
}
