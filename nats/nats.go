package nats

import (
	"errors"
	"fmt"
	"github.com/gwos/tcg/cache"
	"github.com/gwos/tcg/clients"
	"github.com/gwos/tcg/log"
	stan "github.com/nats-io/go-nats-streaming"
	stand "github.com/nats-io/nats-streaming-server/server"
	"github.com/nats-io/nats-streaming-server/stores"
	"math"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Define NATS IDs
const (
	ClusterID    = "tcg-cluster"
	DispatcherID = "tcg-dispatcher"
	PublisherID  = "tcg-publisher"
)

var (
	cfg            Config
	connDispatcher stan.Conn
	connPublisher  stan.Conn
	ctrlChan       chan DispatcherOption
	onceDispatcher sync.Once
	stanServer     *stand.StanServer
	natsURL        string
)

// Config defines NATS configurable options
type Config struct {
	DispatcherAckWait     time.Duration
	DispatcherMaxInflight int
	MaxPubAcksInflight    int
	FilestoreDir          string
	StoreType             string
	NatsHost              string
}

// DispatcherFn defines message processor
type DispatcherFn func([]byte) error

// DispatcherOption defines subscription
type DispatcherOption struct {
	DurableName string
	Subject     string
	Handler     DispatcherFn
}

// DispatcherRetry defines retry
type DispatcherRetry struct {
	LastError error
	Retry     int
}

// Connect returns connection
func Connect(clientID string) (stan.Conn, error) {
	return stan.Connect(
		ClusterID,
		clientID,
		stan.NatsURL(natsURL),
	)
}

// StartServer runs NATS
func StartServer(config Config) error {
	var err error
	cfg = config
	natsOpts := stand.DefaultNatsServerOptions
	natsOpts.MaxPayload = math.MaxInt32
	addrParts := strings.Split(cfg.NatsHost, ":")
	if len(addrParts) == 2 {
		if addrParts[0] != "" {
			natsOpts.Host = addrParts[0]
		}
		if port, err := strconv.Atoi(addrParts[1]); err == nil {
			natsOpts.Port = port
		}
	}
	natsURL = fmt.Sprintf("nats://%s:%d", natsOpts.Host, natsOpts.Port)

	stanOpts := stand.GetDefaultOptions()
	stanOpts.ID = ClusterID
	stanOpts.FilestoreDir = cfg.FilestoreDir
	switch cfg.StoreType {
	case "MEMORY":
		stanOpts.StoreType = stores.TypeMemory
	case "FILE":
		stanOpts.StoreType = stores.TypeFile
	default:
		stanOpts.StoreType = stores.TypeFile
	}

	if stanServer == nil || stanServer.State() == stand.Shutdown {
		stanServer, err = stand.RunServerWithOpts(stanOpts, &natsOpts)
	}
	return err
}

// StopServer shutdowns NATS
func StopServer() {
	if connPublisher != nil {
		connPublisher.Close()
		connPublisher = nil
	}
	stanServer.Shutdown()
}

// StartDispatcher subscribes processors by subject
func StartDispatcher(options []DispatcherOption) error {
	onceDispatcher.Do(func() {
		ctrlChan = make(chan DispatcherOption)
		go listenCtrlChan()
	})

	if err := StopDispatcher(); err != nil {
		return err
	}
	var err error
	if connDispatcher == nil {
		if connDispatcher, err = stan.Connect(
			ClusterID,
			DispatcherID,
			stan.NatsURL(natsURL),
		); err != nil {
			return err
		}
	}

	for _, opt := range options {
		ctrlChan <- opt
	}
	return nil
}

// StopDispatcher ends dispatching
func StopDispatcher() error {
	if connDispatcher != nil {
		cache.DispatcherWorkersCache.Flush()
		err := connDispatcher.Close()
		connDispatcher = nil
		return err
	}
	return nil
}

// Publish adds message in queue
func Publish(subject string, msg []byte) error {
	var err error
	if connPublisher == nil {
		connPublisher, err = stan.Connect(
			ClusterID,
			PublisherID,
			stan.NatsURL(natsURL),
			stan.MaxPubAcksInflight(cfg.MaxPubAcksInflight),
		)
	}
	if err != nil {
		return err
	}

	err = connPublisher.Publish(subject, msg)
	return err
}

func listenCtrlChan() {
	for {
		opt := <-ctrlChan
		if connDispatcher != nil {
			ckWorker := opt.DurableName
			if _, isWorker := cache.DispatcherWorkersCache.Get(ckWorker); !isWorker {
				if err := startWorker(opt); err != nil {
					log.With(log.Fields{
						"error": err,
						"opt":   opt,
					}).Log(log.WarnLevel, "#nats.startWorker failed")
				}
			}
		}
	}
}

func startWorker(opt DispatcherOption) error {
	var (
		errSb        error
		subscription stan.Subscription
	)
	subscription, errSb = connDispatcher.Subscribe(
		opt.Subject,
		func(msg *stan.Msg) {
			ckDone := fmt.Sprintf("%s#%d", opt.DurableName, msg.Sequence)
			if _, isDone := cache.DispatcherDoneCache.Get(ckDone); isDone {
				return
			}

			if err := opt.Handler(msg.Data); err != nil {
				handleWorkerError(subscription, msg, err, opt)
				return
			}
			_ = msg.Ack()
			_ = cache.DispatcherDoneCache.Add(ckDone, 0, 10*time.Minute)
			log.With(log.Fields{
				"durableName": opt.DurableName,
			}).WithDebug(log.Fields{
				"message": msg,
			}).Log(log.InfoLevel, "NATS: Delivered")
		},
		stan.SetManualAckMode(),
		stan.AckWait(cfg.DispatcherAckWait),
		stan.MaxInflight(cfg.DispatcherMaxInflight),
		stan.DurableName(opt.DurableName),
		stan.StartWithLastReceived(),
	)
	return errSb
}

func handleWorkerError(subscription stan.Subscription, msg *stan.Msg, err error, opt DispatcherOption) {
	logEntry := log.With(log.Fields{
		"durableName": opt.DurableName,
	}).WithDebug(log.Fields{
		"error":   err,
		"message": msg,
	})

	if errors.Is(err, clients.ErrGateway) || errors.Is(err, clients.ErrSynchronizer) {
		ckRetry := opt.DurableName
		retry := DispatcherRetry{
			LastError: nil,
			Retry:     0,
		}
		if r, isRetry := cache.DispatcherRetryCache.Get(ckRetry); isRetry {
			retry = r.(DispatcherRetry)
		}

		var delay time.Duration
		retry.LastError = err
		retry.Retry++
		switch retry.Retry {
		case 1:
			delay = 30 * time.Second
		case 2:
			delay = 1 * time.Minute
		case 3:
			delay = 5 * time.Minute
		case 4:
			delay = 20 * time.Minute
		}

		if retry.Retry < 5 {
			cache.DispatcherRetryCache.Set(ckRetry, retry, 0)
			logEntry.WithField("retry", retry).Log(log.InfoLevel, "NATS: Not delivered: Will retry")

			go func() {
				ckWorker := opt.DurableName
				cache.DispatcherWorkersCache.Delete(ckWorker)
				subscription.Close()
				time.Sleep(delay)
				ctrlChan <- opt
			}()
		} else {
			cache.DispatcherRetryCache.Delete(ckRetry)
			logEntry.Log(log.InfoLevel, "NATS: Not delivered: Stop retrying")
		}

	} else {
		logEntry.Log(log.InfoLevel, "NATS: Not delivered: Will not retry")
	}
}
