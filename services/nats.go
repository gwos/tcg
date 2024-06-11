package services

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"

	"github.com/gwos/tcg/nats"
	"github.com/gwos/tcg/sdk/clients"
	tcgerr "github.com/gwos/tcg/sdk/errors"
	"github.com/gwos/tcg/tracing"
	"github.com/rs/zerolog/log"
	"go.opentelemetry.io/otel/trace"
)

func Put2Nats(ctx context.Context, subj string, payload []byte, headers ...string) error {
	ctx, span := tracing.StartTraceSpan(ctx, "services", "Put2Nats")
	var err error
	defer func() {
		tracing.EndTraceSpan(span,
			tracing.TraceAttrError(err),
		)
		if err != nil {
			log.Err(err).Msg("Put2Nats failed")
		}
	}()

	headers = append(headers,
		clients.HdrSpanSpanID, span.SpanContext().SpanID().String(),
		clients.HdrSpanTraceID, span.SpanContext().TraceID().String(),
		clients.HdrSpanTraceFlags, span.SpanContext().TraceFlags().String(),
	)

	if len(payload) > int(agentService.NatsMaxPayload) {
		n0 := len(payload)
		_, payload, err = clients.GZIP(ctx, payload)
		if err != nil {
			return err
		}
		if len(payload) > int(agentService.NatsMaxPayload) {
			err = fmt.Errorf("%w: %v / %v / %v / %v / gzip compressed",
				nats.ErrPayloadLim, subj, agentService.NatsMaxPayload, n0, len(payload))
			return err
		}
		headers = append(headers, clients.HdrCompressed, "gzip",
			clients.HdrPayloadLen, fmt.Sprint(n0))
	}
	headers = append(headers, clients.HdrPayloadLen, fmt.Sprint(len(payload)))

	err = nats.Publish(subj, payload, headers...)
	return err
}

func getCtx(sc trace.SpanContext) context.Context {
	if sc.IsValid() {
		return trace.ContextWithRemoteSpanContext(context.Background(), sc)
	}
	return context.Background()
}

func makeDurable(durable, subj string, handleWithCtx func(context.Context, []byte) error) nats.DurableCfg {
	for _, s := range []string{"/", ".", "*", ">"} {
		durable = strings.ReplaceAll(durable, s, "")
	}
	return nats.DurableCfg{
		Durable: durable,
		Subject: subj,
		Handler: func(msg nats.NatsMsg) error {
			var (
				ctx     context.Context
				err     error
				data    = msg.Data
				headers = msg.Header
				sCtxCfg = trace.SpanContextConfig{}
				spanID  []byte
				traceID []byte
				trFlags []byte
			)
			/* try to process as legacy flow with wrapped payload */
			p := natsPayload{}
			if err = p.Unmarshal(data); err == nil {
				data = p.Payload
				ctx = getCtx(p.SpanContext)
				if pType, t := p.Type.String(), headers.Get(clients.HdrPayloadType); t == "" {
					headers.Set(clients.HdrPayloadType, pType)
				}
			} else {
				/* try to process as latest flow with headers */
				if s, t, tf := headers.Get(clients.HdrSpanSpanID), headers.Get(clients.HdrSpanTraceID),
					headers.Get(clients.HdrSpanTraceFlags); s != "" && t != "" {
					if spanID, err = hex.DecodeString(s); err == nil {
						copy(sCtxCfg.SpanID[:], spanID)
					}
					if traceID, err = hex.DecodeString(t); err == nil {
						copy(sCtxCfg.TraceID[:], traceID)
					}
					if trFlags, err = hex.DecodeString(tf); err == nil {
						sCtxCfg.TraceFlags = trace.TraceFlags(trFlags[0])
					}
					ctx = getCtx(trace.NewSpanContext(sCtxCfg))
				}
			}
			ctx = context.WithValue(ctx, clients.CtxHeaders, headers)

			ctx, span := tracing.StartTraceSpan(ctx, "services", "nats:dispatch")
			defer func() {
				tracing.EndTraceSpan(span,
					tracing.TraceAttrError(err),
					tracing.TraceAttrPayloadLen(data),
					tracing.TraceAttrStr("type", headers.Get(clients.HdrPayloadType)),
					tracing.TraceAttrStr("durable", durable),
					tracing.TraceAttrStr("subject", subj),
				)
			}()

			if err = handleWithCtx(ctx, data); err == nil {
				agentService.stats.x.Add("sentTo:"+durable, 1)
				agentService.stats.BytesSent.Add(int64(len(data)))
				agentService.stats.MessagesSent.Add(1)
				if pType, err := new(payloadType).FromStr(headers.Get(clients.HdrPayloadType)); err == nil &&
					*pType == typeMetrics {
					agentService.stats.MetricsSent.Add(1)
				}
			}

			if errors.Is(err, tcgerr.ErrUnauthorized) {
				/* it looks like an issue with credentialed user
				so, wait for configuration update */
				log.Err(err).Msg("dispatcher got an issue with credentialed user, wait for configuration update")
				_ = agentService.StopTransport()
			} else if errors.Is(err, tcgerr.ErrUndecided) {
				/* it looks like an issue with data */
				log.Err(err).Msg("dispatcher got an issue with data")
			} else if err != nil {
				log.Err(err).Msg("dispatcher got an issue")
			}

			return err
		},
	}
}

func makeSubscriptions(gwClients []*clients.GWClient) []nats.DurableCfg {
	var subs = make([]nats.DurableCfg, 0, len(gwClients))
	for _, gwClient := range gwClients {
		gwClient := gwClient /* hold loop var copy */
		subs = append(subs,
			makeDurable(
				fmt.Sprintf("#%s#%s#", subjDowntimes, gwClient.HostName),
				subjDowntimes,
				adaptClient(gwClient),
			),
			makeDurable(
				fmt.Sprintf("#%s#%s#", subjEvents, gwClient.HostName),
				subjEvents,
				adaptClient(gwClient),
			),
			makeDurable(
				fmt.Sprintf("#%s#%s#", subjInventoryMetrics, gwClient.HostName),
				subjInventoryMetrics,
				adaptClient(gwClient),
			),
		)
	}
	return subs
}

func adaptClient(gwClient *clients.GWClient) func(context.Context, []byte) error {
	return func(ctx context.Context, p []byte) error {
		pTypeHandlers := map[payloadType]func(context.Context, []byte) ([]byte, error){
			typeEvents:          gwClient.SendEvents,
			typeEventsAck:       gwClient.SendEventsAck,
			typeEventsUnack:     gwClient.SendEventsUnack,
			typeClearInDowntime: gwClient.ClearInDowntime,
			typeSetInDowntime:   gwClient.SetInDowntime,
			typeInventory:       gwClient.SynchronizeInventory,
			typeMetrics:         gwClient.SendResourcesWithMetrics,
		}
		var pType payloadType
		if h := ctx.Value(clients.CtxHeaders); h != nil {
			if h, ok := h.(interface{ Get(string) string }); ok {
				if _, err := pType.FromStr(h.Get(clients.HdrPayloadType)); err != nil {
					return err
				}
				if h.Get(clients.HdrTodoTracerCtx) != "" &&
					h.Get(clients.HdrCompressed) == "" {
					p = agentService.fixTracerContext(p)
				}
			}
		}
		if fn, ok := pTypeHandlers[pType]; ok {
			_, err := fn(ctx, p)
			return err
		}
		return fmt.Errorf("%w: unknown payload type: %v", nats.ErrDispatcher, pType)
	}
}
