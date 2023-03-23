package nats

import (
	"errors"
	"fmt"
	"sync"
	"time"

	tcgerr "github.com/gwos/tcg/sdk/errors"
	"github.com/gwos/tcg/taskqueue"
	"github.com/nats-io/nats.go"
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

const taskRetry = "taskRetry"

type dispatcherRetry struct {
	LastError error
	Retry     int
}

// natsDispatcher provides deliverer for nats messages
// with retry logic based on subscribe/close durable subscriptions
type natsDispatcher struct {
	*state

	durables *cache.Cache
	msgsDone *cache.Cache
	retries  *cache.Cache

	taskQueue *taskqueue.TaskQueue
}

func getDispatcher() *natsDispatcher {
	onceDispatcher.Do(func() {
		dispatcher = &natsDispatcher{
			state: s,

			durables: cache.New(-1, -1),
			msgsDone: cache.New(time.Minute*10, time.Minute*10),
			retries:  cache.New(time.Minute*30, time.Minute*30),
		}
		// provide buffer to handle corner case: a few targets anavailable on startup
		dispatcher.taskQueue = taskqueue.NewTaskQueue(
			taskqueue.WithCapacity(64),
			taskqueue.WithHandlers(map[taskqueue.Subject]taskqueue.Handler{
				taskRetry: dispatcher.taskRetryHandler,
			}),
		)
	})
	return dispatcher
}

// handleError handles the error in the processor of durable subscription
// in case of some transient error (like networking issue)
// it unsubscribes current subscription (durable consumer should not be deleted)
// and schedules retry
func (d *natsDispatcher) handleError(subscription *nats.Subscription, msg *nats.Msg, err error, opt DispatcherOption) {
	logEvent := log.Info().Err(err).Str("durable", opt.Durable).
		Func(func(e *zerolog.Event) {
			if zerolog.GlobalLevel() <= zerolog.DebugLevel {
				e.RawJSON("nats.msg.data", msg.Data)
				if meta, err := msg.Metadata(); err == nil {
					e.Uint64("nats.meta.sequence.stream", meta.Sequence.Stream)
					e.Uint64("nats.meta.sequence.consumer", meta.Sequence.Consumer)
					e.Int64("nats.meta.timestamp", meta.Timestamp.Unix())
				}
			}
		})

	if errors.Is(err, tcgerr.ErrTransient) {
		retry := dispatcherRetry{
			LastError: nil,
			Retry:     0,
		}
		if r, isRetry := d.retries.Get(opt.Durable); isRetry {
			retry = r.(dispatcherRetry)
		}
		retry.LastError = err
		retry.Retry++

		if retry.Retry < len(retryDelays) {
			logEvent.Int("retry", retry.Retry).
				Msg("dispatcher could not deliver: will retry")

			d.Lock()
			_ = subscription.Unsubscribe()
			d.durables.Delete(opt.Durable)
			d.retries.Set(opt.Durable, retry, 0)
			d.Unlock()

			delay := retryDelays[retry.Retry]
			_ = time.AfterFunc(delay, func() { _ = d.retryDurable(opt) })
		} else {
			logEvent.Msg("dispatcher could not deliver: stop retrying")
			d.retries.Delete(opt.Durable)
		}
	} else {
		logEvent.Msg("dispatcher could not deliver: will not retry")
	}
}

func (d *natsDispatcher) openDurable(opt DispatcherOption) (*nats.Subscription, error) {
	var (
		errSubs      error
		subscription *nats.Subscription
	)

	js, err := d.ncDispatcher.JetStream(
		nats.DirectGet(),
	)
	if err != nil {
		log.Warn().Err(err).Msg("nats dispatcher failed JetStream")
		return nil, err
	}

	// if ci, err := js.ConsumerInfo(streamName, opt.Durable); err == nats.ErrConsumerNotFound {
	// 	if _, err = js.AddConsumer(streamName, &nats.ConsumerConfig{
	// 		AckPolicy:     nats.AckExplicitPolicy,
	// 		DeliverPolicy: nats.DeliverLastPolicy,
	// 		Durable:       opt.Durable,
	// 		Name:          opt.Durable,
	// 	}); err != nil {
	// 		log.Warn().Err(err).Msg("nats dispatcher failed AddConsumer")
	// 		return nil, err
	// 	}
	// } else if err != nil {
	// 	log.Warn().Err(err).Msg("nats dispatcher failed ConsumerInfo")
	// 	return nil, err
	// } else {
	// 	log.Debug().Interface("consumer", *ci).
	// 		Msg("found durable consumer")
	// }

	subscription, errSubs = js.Subscribe(
		opt.Subject,
		func(msg *nats.Msg) {
			meta, err := msg.Metadata()
			if err != nil {
				return
			}
			ckDone := fmt.Sprintf("%s#%d", opt.Durable, meta.Sequence.Stream)
			if _, isDone := d.msgsDone.Get(ckDone); isDone {
				_ = msg.Ack()
				return
			}
			if err := opt.Handler(msg.Data); err != nil {
				d.handleError(subscription, msg, err, opt)
				return
			}
			_ = msg.Ack()
			_ = d.msgsDone.Add(ckDone, 0, 10*time.Minute)
			log.Info().Str("durable", opt.Durable).
				Func(func(e *zerolog.Event) {
					if zerolog.GlobalLevel() <= zerolog.DebugLevel {
						e.RawJSON("nats.msg.data", msg.Data)
						e.Uint64("nats.meta.sequence.stream", meta.Sequence.Stream)
						e.Uint64("nats.meta.sequence.consumer", meta.Sequence.Consumer)
						e.Int64("nats.meta.timestamp", meta.Timestamp.Unix())
					}
				}).
				Msg("dispatcher delivered")
		},
		// nats.Bind(streamName, opt.Durable),
		nats.BindStream(streamName),
		nats.Durable(opt.Durable),
		nats.AckWait(d.config.AckWait),
		nats.ManualAck(),
	)

	return subscription, errSubs
}

func (d *natsDispatcher) retryDurable(opt DispatcherOption) error {
	_, err := d.taskQueue.PushAsync(taskRetry, opt)
	return err
}

func (d *natsDispatcher) taskRetryHandler(task *taskqueue.Task) error {
	opt := task.Args[0].(DispatcherOption)

	d.Lock()
	defer d.Unlock()

	var err error
	if d.ncDispatcher != nil {
		if _, isOpen := d.durables.Get(opt.Durable); !isOpen {
			if sub, err := d.openDurable(opt); err == nil {
				d.durables.Set(opt.Durable, sub, -1)
			}
		}
	} else {
		err = fmt.Errorf("%w: is not connected", ErrDispatcher)
	}
	if err != nil {
		log.Info().Err(err).
			Str("durable", opt.Durable).
			Msg("dispatcher failed")
	}
	return err
}
