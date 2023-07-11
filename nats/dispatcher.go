package nats

import (
	"context"
	"errors"
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

	retryDelays = map[int]time.Duration{
		1: time.Second * 30,
		2: time.Minute * 1,
		3: time.Minute * 5,
		4: time.Minute * 20,
	}
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

func (d *natsDispatcher) OpenDurable(ctx context.Context, opt DispatcherOption) {
	js, err := jetstream.New(d.ncDispatcher)
	if err != nil {
		log.Err(err).
			Str("durable", opt.Durable).
			Msg("nats dispatcher failed JetStream")
		return
	}

	cons, err := js.CreateOrUpdateConsumer(ctx, streamName, jetstream.ConsumerConfig{
		AckPolicy:     jetstream.AckExplicitPolicy,
		DeliverPolicy: jetstream.DeliverLastPolicy,
		FilterSubject: opt.Subject,
		Durable:       opt.Durable,
		Name:          opt.Durable,
	})
	// cons, err := js.OrderedConsumer(ctx, streamName, jetstream.OrderedConsumerConfig{
	// 	DeliverPolicy:  jetstream.DeliverLastPolicy,
	// 	FilterSubjects: []string{opt.Subject},
	// })
	if err != nil {
		log.Err(err).
			Str("durable", opt.Durable).
			Msg("nats dispatcher failed jetstream Consumer")
		return
	}

	go d.fetch(ctx, opt, cons)
}

func (d *natsDispatcher) fetch2(ctx context.Context, opt DispatcherOption, cons jetstream.Consumer) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		iter, err := cons.Messages(jetstream.PullMaxMessages(4))
		// if err != nil && !errors.Is(err, context.Canceled) && !errors.Is(err, context.DeadlineExceeded) {
		if err != nil {
			log.Err(err).
				Str("durable", opt.Durable).
				Msg("nats dispatcher failed consume Messages")
			continue
		}

		// Process fetched messages and delay next fetching in case of transient error
		var delayRetry *dispatcherRetry
		// Next can return error, e.g. when iterator is closed or no heartbeats were received
		for msg, err := iter.Next(); err == nil; msg, err = iter.Next() {
			if msg.Subject() != opt.Subject {
				println("\n__subj__", msg.Subject(), opt.Subject, opt.Durable)

				// NOTE: this redundant case resolved in Consumer with FilterSubject
				// but OrderedConsumer with FilterSubjects
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
		iter.Stop()

		if delayRetry != nil {
			select {
			case <-ctx.Done(): // context cancelled
			case <-time.After(retryDelays[delayRetry.Retry]): // delay ended
			}
		}
	}
}

func (d *natsDispatcher) fetch(ctx context.Context, opt DispatcherOption, cons jetstream.Consumer) {
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

		// Process fetched messages and delay next fetching in case of transient error
		var delayRetry *dispatcherRetry
		for msg := range msgBatch.Messages() {
			if msg.Subject() != opt.Subject {
				println("\n__subj__", msg.Subject(), opt.Subject, opt.Durable)

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
			case <-time.After(retryDelays[delayRetry.Retry]): // delay ended
			}
		}
	}
}

func (d *natsDispatcher) processMsg(ctx context.Context, opt DispatcherOption, msg jetstream.Msg) *dispatcherRetry {
	meta, err := msg.Metadata()
	if err != nil {
		log.Err(err).Interface("msg", msg).
			Str("durable", opt.Durable).
			Msg("nats dispatcher failed Metadata")
		return nil
	}
	logDetailsFn := func(a ...bool) func(e *zerolog.Event) {
		if zerolog.GlobalLevel() <= zerolog.DebugLevel ||
			(len(a) > 0 && a[0]) {
			return func(e *zerolog.Event) {
				e.RawJSON("nats.msg.data", msg.Data())
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
				Msg("dispatcher lost order")
		}
	}

	err = opt.Handler(ctx, msg.Data())
	if err == nil {
		d.duraSeqs.Set(opt.Durable, meta.Sequence.Stream, -1)
		log.Info().Func(logDetailsFn()).
			Str("durable", opt.Durable).
			Msg("dispatcher delivered")
		return nil
	}
	if !errors.Is(err, tcgerr.ErrTransient) {
		log.Warn().Err(err).Func(logDetailsFn(true)).
			Str("durable", opt.Durable).
			Msg("dispatcher could not deliver: will not retry")
		return nil
	}

	retry := &dispatcherRetry{
		Timestamp: time.Now().UTC(),
		LastError: err,
		Retry:     1,
	}
	if lastRetry, ok := d.retries.Get(opt.Durable); ok {
		lastRetry := lastRetry.(dispatcherRetry)
		if retry.Timestamp.Before(lastRetry.Timestamp.Add(time.Second * 10)) {
			retry.Retry = lastRetry.Retry + 1
		}
	}

	if retry.Retry >= len(retryDelays) {
		d.retries.Delete(opt.Durable)
		log.Warn().Err(err).Func(logDetailsFn(true)).
			Str("durable", opt.Durable).
			Msg("dispatcher could not deliver: stop retrying")
		return nil
	}

	d.retries.Set(opt.Durable, *retry, 0)
	log.Info().Err(err).Func(logDetailsFn()).
		Int("retry", retry.Retry).
		Str("durable", opt.Durable).
		Msg("dispatcher could not deliver: will retry")

	return retry
}
