package nats

import (
	"context"
	"errors"
	"expvar"
	"fmt"
	"sync"
	"time"

	tcgerr "github.com/gwos/tcg/sdk/errors"
	"github.com/nats-io/nats.go/jetstream"
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
	js, err := jetstream.New(d.ncDispatcher)
	if err != nil {
		log.Err(err).
			Str("durable", opt.Durable).
			Msg("nats dispatcher failed JetStream")
		return
	}

	//// nats: cannot run concurrent processing using ordered consumer
	// cons, err := js.OrderedConsumer(ctx, streamName, jetstream.OrderedConsumerConfig{
	// 	DeliverPolicy:  jetstream.DeliverLastPolicy,
	// 	FilterSubjects: []string{opt.Subject},
	// })
	cons, err := js.CreateOrUpdateConsumer(ctx, streamName, jetstream.ConsumerConfig{
		AckWait:        d.config.AckWait,
		AckPolicy:      jetstream.AckExplicitPolicy,
		DeliverPolicy:  jetstream.DeliverLastPolicy,
		FilterSubjects: subjects,
		Durable:        opt.Durable,
		Name:           opt.Durable,
	})
	if err != nil {
		log.Err(err).
			Str("durable", opt.Durable).
			Msg("nats dispatcher failed jetstream Consumer")
		return
	}

	go d.fetch(ctx, opt, cons)
}

func (d *natsDispatcher) fetch(ctx context.Context, opt DurableCfg, cons jetstream.Consumer) {
	xFetchedAt, xProcessedAt, xRetryDelay := new(expvar.Int), new(expvar.Int), new(expvar.String)
	xFetchedAt.Set(-1)
	xProcessedAt.Set(-1)
	xRetryDelay.Set("")
	xStats.Set(opt.Durable+":fetchedAt", xFetchedAt)
	xStats.Set(opt.Durable+":processedAt", xProcessedAt)
	xStats.Set(opt.Durable+":retryDelay", xRetryDelay)
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		// Fetch will return as soon as any message is available rather than wait until the full batch size is available,
		// using a batch size of more than 1 allows for higher throughput when needed.
		msgBatch, err := cons.FetchNoWait(4)
		if err != nil && !errors.Is(err, context.Canceled) && !errors.Is(err, context.DeadlineExceeded) {
			log.Err(err).
				Str("durable", opt.Durable).
				Msg("nats dispatcher failed Fetch")
			continue
		}

		// if len(msgs) == 0 { // js replaced msgs =>> msgBatch
		// 	continue
		// }
		// xFetchedAt.Set(time.Now().UnixMilli())

		// Process fetched messages and delay next fetching in case of transient error
		var delayRetry *dispatcherRetry
		for msg := range msgBatch.Messages() {
			if delayRetry != nil {
				_ = msg.Nak()

				log.Trace().
					Str("durable", opt.Durable).
					Msg("dispatcher skipping: delayRetry != nil")
				continue
			}

			xProcessedAt.Set(time.Now().UnixMilli())
			delayRetry = d.processMsg(ctx, opt, msg)
			if delayRetry != nil {
				_ = msg.Nak()
			} else {
				_ = msg.Ack()
			}
		}
		if delayRetry != nil {
			xRetryDelay.Set(fmt.Sprintf("%v / %v / %v", delayRetry.Retry, RetryDelays[delayRetry.Retry], time.Now().UTC().Format(time.RFC3339)))
			log.Debug().
				Stringer("delay", RetryDelays[delayRetry.Retry]).
				Int("retry", delayRetry.Retry).
				Msg("dispatcher delaying retry")

			select {
			case <-ctx.Done(): // context cancelled
			case <-time.After(RetryDelays[delayRetry.Retry]): // delay ended
			}
			xRetryDelay.Set("")
		}
	}
}

func (d *natsDispatcher) processMsg(ctx context.Context, opt DurableCfg, msg jetstream.Msg) *dispatcherRetry {
	meta, err := msg.Metadata()
	if err != nil {
		log.Err(err).Str("msg", fmt.Sprintf("%+v", msg)).
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
				e.Int64("nats.meta.timestamp", meta.Timestamp.UnixMilli())
			}
		}
		return func(e *zerolog.Event) {}
	}

	if seq, ok := d.duraSeqs.Get(opt.Durable); ok {
		if seq := seq.(uint64); seq >= meta.Sequence.Stream {
			log.Warn().Func(logDetailsFn(true)).
				Uint64("done.sequence", seq).
				Str("durable", opt.Durable).
				Str("nats.msg.Headers", fmt.Sprintf("%+v", msg.Headers())).
				Msg("dispatcher lost order")
		}
	}

	err = opt.Handler(ctx, msg)
	if err == nil {
		d.duraSeqs.Set(opt.Durable, meta.Sequence.Stream, -1)
		log.Info().Func(logDetailsFn()).
			Str("durable", opt.Durable).
			Str("nats.msg.Headers", fmt.Sprintf("%+v", msg.Headers())).
			Msg("dispatcher delivered")
		return nil
	}
	if !errors.Is(err, tcgerr.ErrTransient) {
		log.Warn().Err(err).Func(logDetailsFn(true)).
			Str("durable", opt.Durable).
			Str("nats.msg.Headers", fmt.Sprintf("%+v", msg.Headers())).
			Msg("dispatcher could not deliver: will not retry")
		return nil
	}

	/* processing transient error */
	retry := &dispatcherRetry{
		Timestamp: time.Now().UTC(),
		LastError: err,
		Retry:     0,
	}
	if lastRetry, ok := d.retries.Get(opt.Durable); ok {
		lastRetry := lastRetry.(dispatcherRetry)
		retry.Retry = lastRetry.Retry + 1
	}

	if retry.Retry >= len(RetryDelays) {
		d.retries.Delete(opt.Durable)
		log.Warn().Err(err).Func(logDetailsFn(true)).
			Str("durable", opt.Durable).
			Str("nats.msg.Headers", fmt.Sprintf("%+v", msg.Headers())).
			Msg("dispatcher could not deliver: stop retrying")
		return nil
	}

	d.retries.Set(opt.Durable, *retry, 0)
	log.Info().Err(err).Func(logDetailsFn()).
		Int("retry", retry.Retry).
		Str("durable", opt.Durable).
		Str("nats.msg.Headers", fmt.Sprintf("%+v", msg.Headers())).
		Msg("dispatcher could not deliver: will retry")

	return retry
}
