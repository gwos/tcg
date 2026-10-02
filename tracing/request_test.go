package tracing

import (
	"bytes"
	"compress/gzip"
	"context"
	"io"
	"testing"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
)

func TestGZip(t *testing.T) {
	for _, payload := range [][]byte{nil, []byte(`{"key":"value"}`), bytes.Repeat([]byte("abc"), 100000)} {
		var buf bytes.Buffer
		if _, err := GZip(context.Background(), &buf, payload); err != nil {
			t.Fatal(err)
		}
		r, err := gzip.NewReader(&buf)
		if err != nil {
			t.Fatal(err)
		}
		got, err := io.ReadAll(r)
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(got, payload) {
			t.Errorf("round trip length = %d; want %d", len(got), len(payload))
		}
	}
}

func TestGZipSpanAttributes(t *testing.T) {
	recorder := tracetest.NewSpanRecorder()
	orig := otel.GetTracerProvider()
	otel.SetTracerProvider(sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(recorder)))
	t.Cleanup(func() { otel.SetTracerProvider(orig) })

	payload := bytes.Repeat([]byte("abc"), 10000)
	var buf bytes.Buffer
	if _, err := GZip(context.Background(), &buf, payload); err != nil {
		t.Fatal(err)
	}

	spans := recorder.Ended()
	if len(spans) != 1 || spans[0].Name() != "gzip" {
		t.Fatalf("spans = %v", spans)
	}
	attrs := make(map[attribute.Key]attribute.Value)
	for _, kv := range spans[0].Attributes() {
		attrs[kv.Key] = kv.Value
	}
	if got := attrs["inputLen"].AsInt64(); got != int64(len(payload)) {
		t.Errorf("inputLen = %d; want %d", got, len(payload))
	}
	if got := attrs["outputLen"].AsInt64(); got != int64(buf.Len()) || got >= int64(len(payload)) {
		t.Errorf("outputLen = %d; want the compressed size %d", got, buf.Len())
	}
	if attrs["err"].AsBool() {
		t.Errorf("err = true")
	}
}
