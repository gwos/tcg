package config

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/rs/zerolog/log"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/sdk/resource"
	tracesdk "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.7.0"
)

func initOTLP(serviceName string) (*tracesdk.TracerProvider, error) {
	var (
		ctx = context.TODO()
		exp *otlptrace.Exporter
		err error
	)
	errNotConfigured := fmt.Errorf("telemetry is not configured")
	otlpEndpoint := os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT") +
		os.Getenv("OTEL_EXPORTER_OTLP_TRACES_ENDPOINT")

	switch {
	case strings.Contains(otlpEndpoint, "4317") ||
		strings.Contains(otlpEndpoint, "grpc"):
		exp, err = otlptracegrpc.New(ctx)
	case strings.Contains(otlpEndpoint, "4318") ||
		len(otlpEndpoint) != 0:
		exp, err = otlptracehttp.New(ctx)
	default:
		log.Debug().Msg(errNotConfigured.Error())
		return nil, errNotConfigured
	}

	if err != nil {
		log.Err(err).Msg("could not create exporter")
		return nil, err
	}
	log.Debug().Msg("telemetry configured OTEL_EXPORTER_OTLP")

	attrs := []attribute.KeyValue{
		semconv.ServiceNameKey.String(serviceName),
		attribute.String("buildTag", buildTag),
		attribute.String("buildTime", buildTime),
		attribute.String("runtime", "golang"),
	}

	tp := tracesdk.NewTracerProvider(
		/* Always be sure to batch in production */
		tracesdk.WithBatcher(exp),
		/* Record information about this application in an Resource */
		tracesdk.WithResource(resource.NewWithAttributes(semconv.SchemaURL, attrs...)),
	)
	return tp, nil
}
