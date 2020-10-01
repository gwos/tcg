package services

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_natsPayloadUnmarshalText(t *testing.T) {
	p1 := natsPayload{
		Payload: []byte(`{"key1":"val1"}`),
		Type:    typeMetrics,
	}
	m1, err1 := p1.MarshalText()
	assert.NoError(t, err1)
	p2 := natsPayload{}
	err2 := p2.UnmarshalText(m1)
	assert.NoError(t, err2)
	assert.Equal(t, p1, p2)
}
