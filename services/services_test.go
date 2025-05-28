package services

import (
	"bytes"
	"encoding/gob"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/otel/trace"
)

func Test_statsMarshal(t *testing.T) {
	p, err := json.Marshal(GetAgentService().Stats())
	assert.NoError(t, err)
	assert.Contains(t, string(p), "agentId")
	assert.Contains(t, string(p), `"lastErrors":[`)
	assert.Contains(t, string(p), `"upSince":"`)
}

func Test_natsPayloadMarshal(t *testing.T) {
	p := natsPayload{
		Type:    typeMetrics,
		Payload: []byte(`{"key1":"val1"}`),
		SpanContext: trace.NewSpanContext(trace.SpanContextConfig{
			SpanID:     trace.SpanID{11},
			TraceID:    trace.TraceID{42},
			TraceFlags: trace.TraceFlags(3),
		}),
	}

	t.Run("v1", func(t *testing.T) {
		encoded, err := p.marshalV1()
		assert.NoError(t, err)
		q := natsPayload{}
		assert.NoError(t, q.unmarshalV1(encoded))
		assert.Equal(t, p, q)
	})

	t.Run("v2", func(t *testing.T) {
		encoded, err := p.marshalV2()
		assert.NoError(t, err)
		q := natsPayload{}
		assert.NoError(t, q.unmarshalV2(encoded))
		assert.Equal(t, p, q)
	})

	t.Run("Marshal", func(t *testing.T) {
		encoded, err := p.Marshal()
		assert.NoError(t, err)
		q := natsPayload{}
		assert.NoError(t, q.Unmarshal(encoded))
		assert.Equal(t, p, q)
	})

	t.Run("Unmarshal", func(t *testing.T) {
		var encoded []byte
		var err error
		q := natsPayload{}

		encoded, err = p.marshalV1()
		assert.NoError(t, err)
		assert.NoError(t, q.Unmarshal(encoded))
		assert.Equal(t, p, q)

		encoded, err = p.marshalV2()
		assert.NoError(t, err)
		assert.NoError(t, q.Unmarshal(encoded))
		assert.Equal(t, p, q)
	})
}

func Benchmark_natsPayloadMarshal(b *testing.B) {
	/* natsPayloadGob used for gob encoding
	takes only simple fields from SpanContext because of
	trace.SpanContextConfig doesn't support encoding (otel-v.0.20.0)
	the struct fields ordered to simplify debug view in nats store */
	type natsPayloadGob struct {
		TraceFlags trace.TraceFlags
		TraceID    trace.TraceID
		SpanID     trace.SpanID

		Payload []byte
		Type    payloadType
	}

	marshalGob := func(p natsPayload) ([]byte, error) {
		var buf bytes.Buffer
		pGob := natsPayloadGob{
			Type:       p.Type,
			Payload:    p.Payload,
			SpanID:     p.SpanContext.SpanID(),
			TraceID:    p.SpanContext.TraceID(),
			TraceFlags: p.SpanContext.TraceFlags(),
		}
		enc := gob.NewEncoder(&buf)
		err := enc.Encode(pGob)
		return buf.Bytes(), err
	}

	unmarshalGob := func(p *natsPayload, input []byte) error {
		var pGob natsPayloadGob
		dec := gob.NewDecoder(bytes.NewBuffer(input))
		if err := dec.Decode(&pGob); err != nil {
			return err
		}
		*p = natsPayload{
			Type:    pGob.Type,
			Payload: pGob.Payload,
			SpanContext: trace.NewSpanContext(trace.SpanContextConfig{
				SpanID:     pGob.SpanID,
				TraceID:    pGob.TraceID,
				TraceFlags: pGob.TraceFlags,
			}),
		}
		return nil
	}

	marshalV2fprintf := func(p natsPayload) ([]byte, error) {
		spanID := p.SpanContext.SpanID()
		traceID := p.SpanContext.TraceID()
		traceFlags := p.SpanContext.TraceFlags()
		buf := bytes.NewBuffer(make([]byte, 0, len(p.Payload)+120))
		_, _ = fmt.Fprintf(buf, `{"v2":{"type":"%s","payload":%s,"spanID":"%s","traceID":"%s","traceFlags":%d}}`,
			p.Type.String(), p.Payload,
			hex.EncodeToString(spanID[:]), hex.EncodeToString(traceID[:]), traceFlags)
		return buf.Bytes(), nil
	}
	marshalV2write := func(p natsPayload) ([]byte, error) {
		spanID := p.SpanContext.SpanID()
		traceID := p.SpanContext.TraceID()
		traceFlags := p.SpanContext.TraceFlags()
		buf := bytes.NewBuffer(make([]byte, 0, len(p.Payload)+132))
		_, _ = buf.WriteString(`{"v2":{"type":"`)
		_, _ = buf.WriteString(p.Type.String())
		_, _ = buf.WriteString(`","payload":`)
		_, _ = buf.Write(p.Payload)
		_, _ = buf.WriteString(`,"spanID":"`)
		_, _ = buf.WriteString(hex.EncodeToString(spanID[:]))
		_, _ = buf.WriteString(`","traceID":"`)
		_, _ = buf.WriteString(hex.EncodeToString(traceID[:]))
		_, _ = buf.WriteString(`","traceFlags":`)
		_, _ = buf.WriteString(strconv.FormatUint(uint64(traceFlags), 10))
		_, _ = buf.WriteString(`}}`)
		return buf.Bytes(), nil
	}
	marshalV2append := func(p natsPayload) ([]byte, error) {
		spanID := p.SpanContext.SpanID()
		traceID := p.SpanContext.TraceID()
		traceFlags := p.SpanContext.TraceFlags()
		buf := make([]byte, 0, len(p.Payload)+132)
		buf = append(buf, `{"v2":{"type":"`...)
		buf = append(buf, p.Type.String()...)
		buf = append(buf, `","payload":`...)
		buf = append(buf, p.Payload...)
		buf = append(buf, `,"spanID":"`...)
		buf = append(buf, hex.EncodeToString(spanID[:])...)
		buf = append(buf, `","traceID":"`...)
		buf = append(buf, hex.EncodeToString(traceID[:])...)
		buf = append(buf, `","traceFlags":`...)
		buf = strconv.AppendUint(buf, uint64(traceFlags), 10)
		buf = append(buf, `}}`...)
		return buf, nil
	}
	marshalV2join := func(p natsPayload) ([]byte, error) {
		spanID := p.SpanContext.SpanID()
		traceID := p.SpanContext.TraceID()
		traceFlags := p.SpanContext.TraceFlags()
		return bytes.Join([][]byte{
			[]byte(`{"v2":{"type":"`), []byte(p.Type.String()),
			[]byte(`","payload":`), p.Payload,
			[]byte(`,"spanID":"`), []byte(hex.EncodeToString(spanID[:])),
			[]byte(`","traceID":"`), []byte(hex.EncodeToString(traceID[:])),
			[]byte(`","traceFlags":`), []byte(strconv.FormatUint(uint64(traceFlags), 10)),
			[]byte(`}}`),
		}, []byte(``)), nil
	}
	marshalV2json := func(p natsPayload) ([]byte, error) {
		spanID := p.SpanContext.SpanID()
		traceID := p.SpanContext.TraceID()
		p2 := struct{ V2 natsPayload2 }{natsPayload2{
			Type:       p.Type.String(),
			Payload:    p.Payload,
			SpanID:     hex.EncodeToString(spanID[:]),
			TraceID:    hex.EncodeToString(traceID[:]),
			TraceFlags: uint8(p.SpanContext.TraceFlags()),
		}}
		return json.Marshal(p2)
	}

	p := natsPayload{
		Type:    typeClearInDowntime,
		Payload: []byte(`{"key1":"val1"}`),
		SpanContext: trace.NewSpanContext(trace.SpanContextConfig{
			SpanID:     trace.SpanID{11},
			TraceID:    trace.TraceID{42},
			TraceFlags: trace.TraceFlags(3),
		}),
	}

	b.ResetTimer()

	b.Run("marshalGob", func(b *testing.B) {
		for b.Loop() {
			encoded, err := marshalGob(p)
			assert.NoError(b, err)
			q := natsPayload{}
			assert.NoError(b, unmarshalGob(&q, encoded))
			assert.Equal(b, p, q)
		}
	})

	b.Run("marshalJSON", func(b *testing.B) {
		for b.Loop() {
			encoded, err := json.Marshal(p)
			assert.NoError(b, err)
			q := natsPayload{}
			assert.NoError(b, json.Unmarshal(encoded, &q))
			/* Note: NotEqual */
			assert.NotEqual(b, p, q)
		}
	})

	b.Run("marshalV1", func(b *testing.B) {
		for b.Loop() {
			encoded, err := p.marshalV1()
			assert.NoError(b, err)
			q := natsPayload{}
			assert.NoError(b, q.unmarshalV1(encoded))
			assert.Equal(b, p, q)
		}
	})

	b.Run("marshalV2", func(b *testing.B) {
		for b.Loop() {
			encoded, err := p.marshalV2()
			assert.NoError(b, err)
			q := natsPayload{}
			assert.NoError(b, q.unmarshalV2(encoded))
			assert.Equal(b, p, q)
		}
	})

	b.Run("marshalV2fprintf", func(b *testing.B) {
		for b.Loop() {
			encoded, err := marshalV2fprintf(p)
			assert.NoError(b, err)
			q := natsPayload{}
			assert.NoError(b, q.unmarshalV2(encoded))
			assert.Equal(b, p, q)
		}
	})

	b.Run("marshalV2write", func(b *testing.B) {
		for b.Loop() {
			encoded, err := marshalV2write(p)
			assert.NoError(b, err)
			q := natsPayload{}
			assert.NoError(b, q.unmarshalV2(encoded))
			assert.Equal(b, p, q)
		}
	})

	b.Run("marshalV2append", func(b *testing.B) {
		for b.Loop() {
			encoded, err := marshalV2append(p)
			assert.NoError(b, err)
			q := natsPayload{}
			assert.NoError(b, q.unmarshalV2(encoded))
			assert.Equal(b, p, q)
		}
	})

	b.Run("marshalV2join", func(b *testing.B) {
		for b.Loop() {
			encoded, err := marshalV2join(p)
			assert.NoError(b, err)
			q := natsPayload{}
			assert.NoError(b, q.unmarshalV2(encoded))
			assert.Equal(b, p, q)
		}
	})

	b.Run("marshalV2json", func(b *testing.B) {
		for b.Loop() {
			encoded, err := marshalV2json(p)
			assert.NoError(b, err)
			q := natsPayload{}
			assert.NoError(b, q.unmarshalV2(encoded))
			assert.Equal(b, p, q)
		}
	})
}
