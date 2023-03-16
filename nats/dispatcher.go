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
	retryes  *cache.Cache

	taskQueue *taskqueue.TaskQueue
}

func getDispatcher() *natsDispatcher {
	onceDispatcher.Do(func() {
		dispatcher = &natsDispatcher{
			state: s,

			durables: cache.New(-1, -1),
			msgsDone: cache.New(time.Minute*10, time.Minute*10),
			retryes:  cache.New(time.Minute*30, time.Minute*30),
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
// it closes current subscription (doesn't unsubscribe) and plans retry
func (d *natsDispatcher) handleError(subscription *nats.Subscription, msg *nats.Msg, err error, opt DispatcherOption) {
	logEvent := log.Info().Err(err).Str("durable", opt.Durable).
		Func(func(e *zerolog.Event) {
			if zerolog.GlobalLevel() <= zerolog.DebugLevel {
				e.RawJSON("nats.data", msg.Data)
				//Uint64("stan.sequence", msg.Sequence).
				//Int64("stan.timestamp", msg.Timestamp)
			}
		})

	if errors.Is(err, tcgerr.ErrTransient) {
		retry := dispatcherRetry{
			LastError: nil,
			Retry:     0,
		}
		if r, isRetry := d.retryes.Get(opt.Durable); isRetry {
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
			d.retryes.Set(opt.Durable, retry, 0)
			d.Unlock()

			delay := retryDelays[retry.Retry]
			_ = time.AfterFunc(delay, func() { _ = d.retryDurable(opt) })
		} else {
			logEvent.Msg("dispatcher could not deliver: stop retrying")
			d.retryes.Delete(opt.Durable)
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

	subscription, errSubs = d.jsDispatcher.Subscribe(
		opt.Subject,
		func(msg *nats.Msg) {
			md, err := msg.Metadata()
			if err != nil {
				return
			}
			ckDone := fmt.Sprintf("%s#%d", opt.Durable, md.Sequence.Stream)
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
						e.RawJSON("nats.data", msg.Data).
							Uint64("stan.sequence", md.Sequence.Stream).
							Int64("stan.timestamp", md.Timestamp.Unix())
					}
				}).
				Msg("dispatcher delivered")
		},
		nats.Durable(opt.Durable),
		nats.ManualAck(),
		nats.AckWait(d.config.AckWait),
	)
	if errSubs != nil {
		return nil, errSubs
	}

	// Workaround v8.1.3 to fix processing large natsstore from prior versions
	// Modern envs should use the correct value of MaxInflight setting
	return subscription, subscription.SetPendingLimits(d.config.MaxPendingMsgs, d.config.MaxPendingBytes)
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
	if d.jsDispatcher != nil {
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
