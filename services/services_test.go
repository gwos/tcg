package services

import (
	"bytes"
	"encoding/gob"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/otel/trace"
)

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
		assert.NoError(t, q.Unmarshal([]byte(fmt.Sprintf(`{"v1":%s}`, encoded))))
		assert.Equal(t, p, q)

		encoded, err = p.marshalV2()
		assert.NoError(t, err)
		assert.NoError(t, q.Unmarshal([]byte(fmt.Sprintf(`{"v2":%s}`, encoded))))
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

	marshalV2join := func(p natsPayload) ([]byte, error) {
		spanID := p.SpanContext.SpanID()
		traceID := p.SpanContext.TraceID()
		traceFlags := p.SpanContext.TraceFlags()
		return bytes.Join([][]byte{
			[]byte(`{"type":"`), []byte(p.Type.String()),
			[]byte(`","payload":`), p.Payload,
			[]byte(`,"spanID":"`), []byte(hex.EncodeToString(spanID[:])),
			[]byte(`","traceID":"`), []byte(hex.EncodeToString(traceID[:])),
			[]byte(`","traceFlags":`), []byte(fmt.Sprintf(`%d`, traceFlags)),
			[]byte(`}`),
		}, []byte(``)), nil
	}
	marshalV2json := func(p natsPayload) ([]byte, error) {
		spanID := p.SpanContext.SpanID()
		traceID := p.SpanContext.TraceID()
		p2 := natsPayload2{
			Type:       p.Type.String(),
			Payload:    p.Payload,
			SpanID:     hex.EncodeToString(spanID[:]),
			TraceID:    hex.EncodeToString(traceID[:]),
			TraceFlags: uint8(p.SpanContext.TraceFlags()),
		}
		return json.Marshal(p2)
	}

	p := natsPayload{
		Type:    typeMetrics,
		Payload: []byte(`{"key1":"val1"}`),
		SpanContext: trace.NewSpanContext(trace.SpanContextConfig{
			SpanID:     trace.SpanID{11},
			TraceID:    trace.TraceID{42},
			TraceFlags: trace.TraceFlags(3),
		}),
	}

	b.ResetTimer()

	b.Run("marshalGob", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			encoded, err := marshalGob(p)
			assert.NoError(b, err)
			q := natsPayload{}
			assert.NoError(b, unmarshalGob(&q, encoded))
			assert.Equal(b, p, q)
		}
	})

	b.Run("marshalJSON", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			encoded, err := json.Marshal(p)
			assert.NoError(b, err)
			q := natsPayload{}
			assert.NoError(b, json.Unmarshal(encoded, &q))
			/* Note: NotEqual */
			assert.NotEqual(b, p, q)
		}
	})

	b.Run("marshalV1", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			encoded, err := p.marshalV1()
			assert.NoError(b, err)
			q := natsPayload{}
			assert.NoError(b, q.unmarshalV1(encoded))
			assert.Equal(b, p, q)
		}
	})

	b.Run("marshalV2join", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			encoded, err := marshalV2join(p)
			assert.NoError(b, err)
			q := natsPayload{}
			assert.NoError(b, q.unmarshalV2(encoded))
			assert.Equal(b, p, q)
		}
	})

	b.Run("marshalV2json", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			encoded, err := marshalV2json(p)
			assert.NoError(b, err)
			q := natsPayload{}
			assert.NoError(b, q.unmarshalV2(encoded))
			assert.Equal(b, p, q)
		}
	})

	b.Run("marshalV2", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			encoded, err := p.marshalV2()
			assert.NoError(b, err)
			q := natsPayload{}
			assert.NoError(b, q.unmarshalV2(encoded))
			assert.Equal(b, p, q)
		}
	})
}

func Benchmark_natsPayloadFprintf(b *testing.B) {
	p := natsPayload{
		Type:    typeMetrics,
		Payload: []byte(`{"key1":"val1"}`),
		SpanContext: trace.NewSpanContext(trace.SpanContextConfig{
			SpanID:     trace.SpanID{11},
			TraceID:    trace.TraceID{42},
			TraceFlags: trace.TraceFlags(3),
		}),
	}

	b.Run("V2_1", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			pp, err := p.marshalV2()
			assert.NoError(b, err)
			mm := []byte(fmt.Sprintf(`{"v2":%s}`, pp))

			q := natsPayload{}
			assert.NoError(b, q.Unmarshal(mm))
			assert.Equal(b, p, q)
		}
	})

	b.Run("V2_2", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			pp, err := p.marshalV2()
			assert.NoError(b, err)
			buf := bytes.NewBuffer(make([]byte, 0, len(pp)+8))
			_, err = fmt.Fprintf(buf, `{"v2":%s}`, pp)
			mm := buf.Bytes()

			q := natsPayload{}
			assert.NoError(b, q.Unmarshal(mm))
			assert.Equal(b, p, q)
		}
	})

	b.Run("V2_3", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			pp, err := p.marshalV2()
			assert.NoError(b, err)
			buf := bytes.NewBuffer(make([]byte, 0, len(pp)+8))
			buf.Write([]byte(`{"v2":`))
			buf.Write(pp)
			buf.Write([]byte(`}`))
			mm := buf.Bytes()

			q := natsPayload{}
			assert.NoError(b, q.Unmarshal(mm))
			assert.Equal(b, p, q)
		}
	})

	b.Run("V2_4", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			pp, err := p.marshalV2()
			assert.NoError(b, err)
			mm := bytes.Join([][]byte{
				[]byte(`{"v2":`), pp, []byte(`}`),
			}, []byte(``))

			q := natsPayload{}
			assert.NoError(b, q.Unmarshal(mm))
			assert.Equal(b, p, q)
		}
	})

	type v2wrap struct {
		V2 json.RawMessage `json:"v2"`
	}

	b.Run("V2_5", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			pp, err := p.marshalV2()
			assert.NoError(b, err)
			mm, err := json.Marshal(v2wrap{pp})

			q := natsPayload{}
			assert.NoError(b, q.Unmarshal(mm))
			assert.Equal(b, p, q)
		}
	})
}
