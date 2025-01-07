package nats

import (
	"context"
	"errors"
	"sync"
	"time"

	tcgerr "github.com/gwos/tcg/sdk/errors"
	"github.com/nats-io/nats.go"
	"github.com/patrickmn/go-cache"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

var (
	dispatcher     *natsDispatcher
	onceDispatcher sync.Once

	// RetryDelays is overridden from config package
	RetryDelays = []time.Duration{time.Second * 30, time.Minute * 1, time.Minute * 5, time.Minute * 20}
)

type dispatcherRetry struct {
	Timestamp time.Time
	LastError error
	Retry     int
}

// natsDispatcher provides deliverer for nats messages
// with retry logic based on subscribe/close durable subscriptions
type natsDispatcher struct {
	*state

	duraSeqs *cache.Cache
	retries  *cache.Cache
	cancel   context.CancelFunc
}

func getDispatcher() *natsDispatcher {
	onceDispatcher.Do(func() {
		dispatcher = &natsDispatcher{
			state:    s,
			duraSeqs: cache.New(-1, -1),
			retries:  cache.New(time.Minute*30, time.Minute*30),
		}
	})
	return dispatcher
}

func (d *natsDispatcher) Flush() {
	d.duraSeqs.Flush()
	d.retries.Flush()
}

func (d *natsDispatcher) OpenDurable(ctx context.Context, opt DurableCfg) {
	js, err := d.ncDispatcher.JetStream(
		nats.DirectGet(),
	)
	if err != nil {
		log.Warn().Err(err).
			Str("durable", opt.Durable).
			Msg("nats dispatcher failed JetStream")
		return
	}

	if _, err := js.ConsumerInfo(streamName, opt.Durable); errors.Is(err, nats.ErrConsumerNotFound) {
		if _, err = js.AddConsumer(streamName, &nats.ConsumerConfig{
			// AckWait:       d.config.AckWait,
			AckPolicy:     nats.AckExplicitPolicy,
			DeliverPolicy: nats.DeliverLastPolicy,
			Durable:       opt.Durable,
			Name:          opt.Durable,
		}); err != nil {
			log.Warn().Err(err).
				Str("durable", opt.Durable).
				Msg("nats dispatcher failed AddConsumer")
			return
		}
	} else if err != nil {
		log.Warn().Err(err).
			Str("durable", opt.Durable).
			Msg("nats dispatcher failed ConsumerInfo")
		return
	}

	sub, err := js.PullSubscribe(
		opt.Subject, opt.Durable,

		nats.Bind(streamName, opt.Durable),
		// nats.BindStream(streamName),
		// nats.Durable(opt.Durable),
		// nats.AckWait(d.config.AckWait),
		nats.ManualAck(),
	)
	if err != nil {
		log.Warn().Err(err).
			Str("durable", opt.Durable).
			Msg("nats dispatcher failed Subscribe")
		return
	}

	go d.fetch(ctx, opt, sub)
}

func (d *natsDispatcher) fetch(ctx context.Context, opt DurableCfg, sub *nats.Subscription) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		// Fetch will return as soon as any message is available rather than wait until the full batch size is available,
		// using a batch size of more than 1 allows for higher throughput when needed.
		msgs, err := sub.Fetch(4, nats.Context(ctx))
		if err != nil && !errors.Is(err, context.Canceled) && !errors.Is(err, context.DeadlineExceeded) {
			log.Warn().Err(err).
				Str("durable", opt.Durable).
				Msg("nats dispatcher failed Fetch")
			continue
		}

		// Process fetched messages and delay next fetching in case of transient error
		var delayRetry *dispatcherRetry
		for _, msg := range msgs {
			if msg.Subject != opt.Subject {
				// TODO: resolve this redundant case
				_ = msg.Ack()
				continue
			}
			if delayRetry != nil {
				_ = msg.Nak()
				continue
			}

			delayRetry = d.processMsg(ctx, opt, msg)
			if delayRetry != nil {
				_ = msg.Nak()
			} else {
				_ = msg.Ack()
			}
		}
		if delayRetry != nil {
			select {
			case <-ctx.Done(): // context cancelled
			case <-time.After(RetryDelays[delayRetry.Retry]): // delay ended
			}
		}
	}
}

func (d *natsDispatcher) processMsg(_ context.Context, opt DurableCfg, msg *nats.Msg) *dispatcherRetry {
	meta, err := msg.Metadata()
	if err != nil {
		log.Warn().Err(err).Any("msg", *msg).
			Str("durable", opt.Durable).
			Msg("nats dispatcher failed Metadata")
		return nil
	}
	logDetailsFn := func(a ...bool) func(e *zerolog.Event) {
		if zerolog.GlobalLevel() <= zerolog.DebugLevel ||
			(len(a) > 0 && a[0]) {
			return func(e *zerolog.Event) {
				e.Uint64("nats.meta.sequence.stream", meta.Sequence.Stream)
				e.Uint64("nats.meta.sequence.consumer", meta.Sequence.Consumer)
				e.Int64("nats.meta.timestamp", meta.Timestamp.Unix())
			}
		}
		return func(e *zerolog.Event) {}
	}

	if seq, ok := d.duraSeqs.Get(opt.Durable); ok {
		if seq := seq.(uint64); seq >= meta.Sequence.Stream {
			log.Warn().Func(logDetailsFn(true)).
				Uint64("done.sequence", seq).
				Str("durable", opt.Durable).
				Any("nats.msg.headers", msg.Header).
				Msg("dispatcher lost order")
		}
	}

	err = opt.Handler(NatsMsg{msg})
	if err == nil {
		d.duraSeqs.Set(opt.Durable, meta.Sequence.Stream, -1)
		log.Info().Func(logDetailsFn()).
			Str("durable", opt.Durable).
			Any("nats.msg.headers", msg.Header).
			Msg("dispatcher delivered")
		return nil
	}
	if !errors.Is(err, tcgerr.ErrTransient) {
		log.Warn().Err(err).Func(logDetailsFn(true)).
			Str("durable", opt.Durable).
			Any("nats.msg.headers", msg.Header).
			Msg("dispatcher could not deliver: will not retry")
		return nil
	}

	retry := &dispatcherRetry{
		Timestamp: time.Now().UTC(),
		LastError: err,
		Retry:     0,
	}
	if lastRetry, ok := d.retries.Get(opt.Durable); ok {
		lastRetry := lastRetry.(dispatcherRetry)
		if retry.Timestamp.Before(lastRetry.Timestamp.Add(time.Second * 10)) {
			retry.Retry = lastRetry.Retry + 1
		}
	}

	if retry.Retry >= len(RetryDelays) {
		d.retries.Delete(opt.Durable)
		log.Warn().Err(err).Func(logDetailsFn(true)).
			Str("durable", opt.Durable).
			Any("nats.msg.headers", msg.Header).
			Msg("dispatcher could not deliver: stop retrying")
		return nil
	}

	d.retries.Set(opt.Durable, *retry, 0)
	log.Info().Err(err).Func(logDetailsFn()).
		Int("retry", retry.Retry).
		Str("durable", opt.Durable).
		Any("nats.msg.headers", msg.Header).
		Msg("dispatcher could not deliver: will retry")

	return retry
}
