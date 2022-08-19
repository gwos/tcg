package config

import (
	"fmt"
	"net"
	"os"

	"github.com/rs/zerolog/log"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/jaeger"
	"go.opentelemetry.io/otel/sdk/resource"
	tracesdk "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.7.0"
)

// Jaegertracing defines the configuration of telemetry provider
type Jaegertracing struct {
	// Agent defines address for communicating with AgentJaegerThriftCompactUDP,
	// hostport, like jaeger-agent:6831
	Agent string `yaml:"agent"`
	// Collector defines traces endpoint,
	// in case the client should connect directly to the CollectorHTTP,
	// endpoint, like http://jaeger-collector:14268/api/traces
	Collector string `yaml:"collector"`
	// Tags defines tracer-level tags, which get added to all reported spans
	Tags map[string]string `yaml:"tags"`
}

// initJaegertracing inits tracing provider with Jaeger exporter
func initJaegertracing(jt Jaegertracing, serviceName string) (*tracesdk.TracerProvider, error) {
	var errNotConfigured = fmt.Errorf("telemetry is not configured")
	/* Jaegertracing supports a few options to receive spans
	[https://github.com/jaegertracing/jaeger/blob/master/ports/ports.go]
	// AgentJaegerThriftCompactUDP is the default port for receiving Jaeger Thrift over UDP in compact encoding
	AgentJaegerThriftCompactUDP = 6831
	// AgentJaegerThriftBinaryUDP is the default port for receiving Jaeger Thrift over UDP in binary encoding
	AgentJaegerThriftBinaryUDP = 6832
	// AgentZipkinThriftCompactUDP is the default port for receiving Zipkin Thrift over UDP in binary encoding
	AgentZipkinThriftCompactUDP = 5775
	// CollectorGRPC is the default port for gRPC server for sending spans
	CollectorGRPC = 14250
	// CollectorHTTP is the default port for HTTP server for sending spans (e.g. /api/traces endpoint)
	CollectorHTTP = 14268

	The otel jaeger exporter supports AgentJaegerThriftCompactUDP and CollectorHTTP protocols.
	otel-v0.20.0 Note the possible mistakes in defaults:
		* "6832" for jaeger.WithAgentPort()
		* "http://localhost:14250" for jaeger.WithCollectorEndpoint()

	Checking configuration to prevent exporter run with internal defaults in environment without receiver.
	The OTEL_EXPORTER_ env vars take precedence on the TCG config (with TCG_JAEGERTRACING_ env vars).
	And the Agent entrypoint setting takes precedence on the Collector entrypoint. */
	otelExporterJaegerAgentHost := os.Getenv("OTEL_EXPORTER_JAEGER_AGENT_HOST")
	otelExporterJaegerAgentPort := os.Getenv("OTEL_EXPORTER_JAEGER_AGENT_PORT")
	otelExporterJaegerEndpoint := os.Getenv("OTEL_EXPORTER_JAEGER_ENDPOINT")
	otelExporterJaegerPassword := os.Getenv("OTEL_EXPORTER_JAEGER_PASSWORD")
	otelExporterJaegerUser := os.Getenv("OTEL_EXPORTER_JAEGER_USER")
	tcgJaegerAgent := jt.Agent
	tcgJaegerCollector := jt.Collector

	var endpointOption jaeger.EndpointOption
	switch {
	case len(otelExporterJaegerAgentHost)+len(otelExporterJaegerAgentPort) != 0:
		endpointOption = jaeger.WithAgentEndpoint()
	case len(otelExporterJaegerEndpoint)+len(otelExporterJaegerPassword)+len(otelExporterJaegerUser) != 0:
		endpointOption = jaeger.WithCollectorEndpoint()
	case len(tcgJaegerAgent) != 0:
		if host, port, err := net.SplitHostPort(tcgJaegerAgent); err == nil {
			endpointOption = jaeger.WithAgentEndpoint(
				jaeger.WithAgentHost(host),
				jaeger.WithAgentPort(port),
			)
		} else {
			log.Err(err).Msg("could not parse the JaegerAgent")
			return nil, err
		}
	case len(tcgJaegerCollector) != 0:
		endpointOption = jaeger.WithCollectorEndpoint(jaeger.WithEndpoint(tcgJaegerCollector))
	default:
		log.Debug().Msg(errNotConfigured.Error())
		return nil, errNotConfigured
	}

	attrs := []attribute.KeyValue{
		semconv.ServiceNameKey.String(serviceName),
		attribute.String("runtime", "golang"),
	}
	for k, v := range jt.Tags {
		attrs = append(attrs, attribute.String(k, v))
	}

	/* It may be useful to look for modern API state and usage at:
	https://github.com/open-telemetry/opentelemetry-go/blob/main/example/jaeger/main.go */

	exporter, err := jaeger.New(endpointOption)
	if err != nil {
		log.Err(err).Msg("could not create exporter")
		return nil, err
	}
	tp := tracesdk.NewTracerProvider(
		/* Always be sure to batch in production */
		tracesdk.WithBatcher(exporter),
		/* Record information about this application in an Resource */
		tracesdk.WithResource(resource.NewWithAttributes(semconv.SchemaURL, attrs...)),
	)
	return tp, nil
}
