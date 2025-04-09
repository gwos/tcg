package services

import (
	"bytes"
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"strings"

	tcgnats "github.com/gwos/tcg/nats"
	"github.com/gwos/tcg/sdk/clients"
	tcgerr "github.com/gwos/tcg/sdk/errors"
	"github.com/gwos/tcg/tracing"
	"github.com/nats-io/nats.go"
	"github.com/rs/zerolog/log"
	"go.opentelemetry.io/otel/trace"
)

func Put2Nats(ctx context.Context, subj string, payload []byte, header http.Header) error {
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

	header.Set(clients.HdrSpanSpanID, span.SpanContext().SpanID().String())
	header.Set(clients.HdrSpanTraceID, span.SpanContext().TraceID().String())
	header.Set(clients.HdrSpanTraceFlags, span.SpanContext().TraceFlags().String())

	if len(payload) > int(agentService.NatsMaxPayload) {
		n0 := len(payload)
		buf := new(bytes.Buffer)
		_, err = clients.GZip(ctx, buf, payload)
		if err != nil {
			return err
		}
		if buf.Len() > int(agentService.NatsMaxPayload) {
			err = fmt.Errorf("%w: %v / %v / %v / %v / gzip compressed",
				tcgnats.ErrPayloadLim, subj, agentService.NatsMaxPayload, n0, buf.Len())
			return err
		}
		payload = buf.Bytes()
		header.Set(clients.HdrCompressed, "gzip")
		header.Set(clients.HdrPayloadLen, fmt.Sprint(n0))
	}
	header.Add(clients.HdrPayloadLen, fmt.Sprint(len(payload)))

	err = tcgnats.Pub(subj, payload, header)
	return err
}

func getCtx(ctx context.Context, sc trace.SpanContext) context.Context {
	if sc.IsValid() {
		return trace.ContextWithRemoteSpanContext(ctx, sc)
	}
	return ctx
}

func makeDurable(durable string, handleWithCtx func(context.Context, *nats.Msg) error) tcgnats.DurableCfg {
	for _, s := range []string{"/", ".", "*", ">"} {
		durable = strings.ReplaceAll(durable, s, "")
	}
	return tcgnats.DurableCfg{
		Durable: durable,
		Handler: func(ctx context.Context, msg *nats.Msg) error {
			var (
				err     error
				data    = msg.Data
				header  = msg.Header
				subject = msg.Subject
				sCtxCfg = trace.SpanContextConfig{}
				spanID  []byte
				traceID []byte
				trFlags []byte
			)
			/* try to process as legacy flow with wrapped payload */
			p := natsPayload{}
			if err = p.Unmarshal(data); err == nil {
				data = p.Payload
				ctx = getCtx(ctx, p.SpanContext)
				if pType, t := p.Type.String(), header.Get(clients.HdrPayloadType); t == "" {
					header.Set(clients.HdrPayloadType, pType)
				}
			} else {
				/* try to process as latest flow with header */
				if s, t, tf := header.Get(clients.HdrSpanSpanID), header.Get(clients.HdrSpanTraceID),
					header.Get(clients.HdrSpanTraceFlags); s != "" && t != "" {
					if spanID, err = hex.DecodeString(s); err == nil {
						copy(sCtxCfg.SpanID[:], spanID)
					}
					if traceID, err = hex.DecodeString(t); err == nil {
						copy(sCtxCfg.TraceID[:], traceID)
					}
					if trFlags, err = hex.DecodeString(tf); err == nil {
						sCtxCfg.TraceFlags = trace.TraceFlags(trFlags[0])
					}
					ctx = getCtx(ctx, trace.NewSpanContext(sCtxCfg))
				}
			}

			ctx, span := tracing.StartTraceSpan(ctx, "services", "nats:dispatch")
			defer func() {
				tracing.EndTraceSpan(span,
					tracing.TraceAttrError(err),
					tracing.TraceAttrPayloadLen(data),
					tracing.TraceAttrStr("type", header.Get(clients.HdrPayloadType)),
					tracing.TraceAttrStr("durable", durable),
					tracing.TraceAttrStr("subject", subject),
				)
			}()

			if err = handleWithCtx(ctx, msg); err == nil {
				agentService.stats.x.Add("sentTo:"+durable, 1)
				agentService.stats.BytesSent.Add(int64(len(data)))
				agentService.stats.MessagesSent.Add(1)
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

func makeSubscriptions(gwClients []clients.GWClient) []tcgnats.DurableCfg {
	var subs = make([]tcgnats.DurableCfg, 0, len(gwClients))
	for i := range gwClients {
		// gwClient := gwClient /* hold loop var copy */
		gwClient := &gwClients[i]
		subs = append(subs, makeDurable(
			fmt.Sprintf("#%s#", gwClient.HostName),
			adaptClient(gwClient),
		))
	}
	return subs
}

func adaptClient(gwClient *clients.GWClient) func(context.Context, *nats.Msg) error {
	return func(ctx context.Context, msg *nats.Msg) error {
		data, header, pType := msg.Data, msg.Header, new(payloadType)
		if _, err := pType.FromStr(header.Get(clients.HdrPayloadType)); err != nil {
			return err
		}
		if header.Get(clients.HdrTodoTracerCtx) != "" &&
			header.Get(clients.HdrCompressed) == "" {
			// TODO: process redundant case (HdrTodoTracerCtx && HdrCompressed)
			data = agentService.fixTracerContext(data)
			header.Del(clients.HdrTodoTracerCtx)
		}
		ctx = clients.CtxWithHeader(ctx, http.Header(header))

		var fn func(context.Context, []byte) ([]byte, error)
		switch *pType {
		case typeEvents:
			fn = gwClient.SendEvents
		case typeEventsAck:
			fn = gwClient.SendEventsAck
		case typeEventsUnack:
			fn = gwClient.SendEventsUnack
		case typeClearInDowntime:
			fn = gwClient.ClearInDowntime
		case typeSetInDowntime:
			fn = gwClient.SetInDowntime
		case typeInventory:
			fn = gwClient.SynchronizeInventory
		case typeMetrics:
			fn = gwClient.SendResourcesWithMetrics
		default:
			return fmt.Errorf("%w: unknown payload type: %v", tcgnats.ErrDispatcher, *pType)
		}
		_, err := fn(ctx, data)
		if err == nil && *pType == typeMetrics {
			agentService.stats.MetricsSent.Add(1)
		}

		return err
	}
}
