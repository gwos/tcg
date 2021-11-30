package nats

import (
	"errors"
	"fmt"
	"sync"
	"time"

	tcgerr "github.com/gwos/tcg/sdk/errors"
	"github.com/gwos/tcg/taskQueue"
	"github.com/nats-io/stan.go"
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

	taskQueue *taskQueue.TaskQueue
}

func getDispatcher() *natsDispatcher {
	onceDispatcher.Do(func() {
		dispatcher = &natsDispatcher{
			state: s,

			durables: cache.New(-1, -1),
			msgsDone: cache.New(time.Minute*10, time.Minute*10),
			retryes:  cache.New(time.Minute*30, time.Minute*30),
		}
		dispatcher.taskQueue = taskQueue.NewTaskQueue(
			taskQueue.WithHandlers(map[taskQueue.Subject]taskQueue.Handler{
				taskRetry: dispatcher.taskRetryHandler,
			}),
		)
	})
	return dispatcher
}

// handleError handles the error in the processor of durable subscription
// in case of some transient error (like networking issue)
// it closes current subscription (doesn't unsubscribe) and plans retry
func (d *natsDispatcher) handleError(subscription stan.Subscription, msg *stan.Msg, err error, opt DispatcherOption) {
	logEvent := log.Info().Err(err).Str("durableName", opt.DurableName).
		Func(func(e *zerolog.Event) {
			if zerolog.GlobalLevel() <= zerolog.DebugLevel {
				e.RawJSON("stan.data", msg.Data).
					Uint64("stan.sequence", msg.Sequence).
					Int64("stan.timestamp", msg.Timestamp)
			}
		})

	if errors.Is(err, tcgerr.ErrTransient) {
		retry := dispatcherRetry{
			LastError: nil,
			Retry:     0,
		}
		if r, isRetry := d.retryes.Get(opt.DurableName); isRetry {
			retry = r.(dispatcherRetry)
		}
		retry.LastError = err
		retry.Retry++

		if retry.Retry < len(retryDelays) {
			logEvent.Int("retry", retry.Retry).
				Msg("dispatcher could not deliver: will retry")
			d.retryes.Set(opt.DurableName, retry, 0)
			delay := retryDelays[retry.Retry]

			go func() {
				d.Lock()
				_ = subscription.Close()
				d.durables.Delete(opt.DurableName)
				d.Unlock()

				time.AfterFunc(delay, func() { _ = d.retryDurable(opt) })
			}()
		} else {
			d.retryes.Delete(opt.DurableName)
			logEvent.Msg("dispatcher could not deliver: stop retrying")
		}

	} else {
		logEvent.Msg("dispatcher could not deliver: will not retry")
	}
}

func (d *natsDispatcher) openDurable(opt DispatcherOption) error {
	var (
		errSubs      error
		subscription stan.Subscription
	)
	if subscription, errSubs = d.connDispatcher.Subscribe(
		opt.Subject,
		func(msg *stan.Msg) {
			// Note: https://github.com/nats-io/nats-streaming-server/issues/1126#issuecomment-726903074
			// ..when the subscription starts and has a lot of backlog messages,
			// is that the server is going to send all pending messages for this consumer "at once",
			// that is, without releasing the consumer lock.
			// The application may get them and ack, but the ack won't be processed
			// because the server is still sending messages to this consumer.
			// ..if it takes longer to send all pending messages [then AckWait], the message will also get redelivered.
			// ..If the server redelivers the message is that it thinks that the message has not been acknowledged,
			// and it may in that case resend again, so you should Ack the message there.
			ckDone := fmt.Sprintf("%s#%d", opt.DurableName, msg.Sequence)
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
			log.Info().Str("durableName", opt.DurableName).
				Func(func(e *zerolog.Event) {
					if zerolog.GlobalLevel() <= zerolog.DebugLevel {
						e.RawJSON("stan.data", msg.Data).
							Uint64("stan.sequence", msg.Sequence).
							Int64("stan.timestamp", msg.Timestamp)
					}
				}).
				Msg("dispatcher delivered")
		},
		stan.SetManualAckMode(),
		stan.AckWait(d.config.AckWait),
		stan.MaxInflight(d.config.MaxInflight),
		stan.DurableName(opt.DurableName),
		stan.StartWithLastReceived(),
	); errSubs != nil {
		return errSubs
	}

	// Workaround v8.1.3 to fix processing large natsstore from prior versions
	// Modern envs should use the correct value of MaxInflight setting
	return subscription.SetPendingLimits(d.config.MaxPendingMsgs, d.config.MaxPendingBytes)
}

func (d *natsDispatcher) retryDurable(opt DispatcherOption) error {
	_, err := d.taskQueue.PushAsync(taskRetry, opt)
	return err
}

func (d *natsDispatcher) taskRetryHandler(task *taskQueue.Task) error {
	opt := task.Args[0].(DispatcherOption)

	d.Lock()
	defer d.Unlock()

	var err error
	if d.connDispatcher != nil {
		if _, isOpen := d.durables.Get(opt.DurableName); !isOpen {
			if err = d.openDurable(opt); err == nil {
				d.durables.Set(opt.DurableName, 0, -1)
			}
		}
	} else {
		err = fmt.Errorf("%w: is not connected", ErrDispatcher)
	}
	if err != nil {
		log.Info().Err(err).
			Str("durableName", opt.DurableName).
			Msg("dispatcher failed")
	}
	return err
}
