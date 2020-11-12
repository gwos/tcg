package nats

import (
	"errors"
	"fmt"
	"math"
	"sync"
	"time"

	"github.com/gwos/tcg/cache"
	"github.com/gwos/tcg/clients"
	"github.com/gwos/tcg/log"
	natsd "github.com/nats-io/nats-server/v2/server"
	stand "github.com/nats-io/nats-streaming-server/server"
	"github.com/nats-io/nats-streaming-server/stores"
	// "github.com/nats-io/stan.go"
	stan "github.com/nats-io/go-nats-streaming"
)

// Define NATS IDs
const (
	clusterID    = "tcg-cluster"
	dispatcherID = "tcg-dispatcher"
	publisherID  = "tcg-publisher"
)

var (
	mu      = &sync.Mutex{}
	natsURL = "" //  ensures connections to proper NATS instance

	cfg            Config
	connDispatcher stan.Conn
	connPublisher  stan.Conn
	ctrlChan       chan DispatcherOption
	onceDispatcher sync.Once
	natsServer     *natsd.Server
	stanServer     *stand.StanServer
)

// Config defines NATS configurable options
type Config struct {
	DispatcherAckWait     time.Duration
	DispatcherMaxInflight int
	MaxPubAcksInflight    int
	FilestoreDir          string
	StoreType             string
	StoreMaxAge           time.Duration
	StoreMaxBytes         int64
	StoreBufferSize       int
	StoreReadBufferSize   int
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

// StartServer runs NATS
func StartServer(config Config) error {
	var err error
	cfg = config

	natsOpts := stand.DefaultNatsServerOptions.Clone()
	natsOpts.MaxPayload = math.MaxInt32

	stanOpts := stand.GetDefaultOptions().Clone()
	stanOpts.ID = clusterID
	stanOpts.FilestoreDir = cfg.FilestoreDir
	switch cfg.StoreType {
	case "MEMORY":
		stanOpts.StoreType = stores.TypeMemory
	case "FILE":
		stanOpts.StoreType = stores.TypeFile
	default:
		stanOpts.StoreType = stores.TypeFile
	}
	stanOpts.StoreLimits.MaxAge = cfg.StoreMaxAge
	stanOpts.StoreLimits.MaxBytes = cfg.StoreMaxBytes
	stanOpts.FileStoreOpts.BufferSize = cfg.StoreBufferSize
	// stanOpts.FileStoreOpts.ReadBufferSize = cfg.StoreReadBufferSize

	mu.Lock()
	defer mu.Unlock()

	if len(natsURL) == 0 {
		nopts := &natsd.Options{}
		nopts.Host = "127.0.0.1"
		nopts.Port = -1 //  automatically chosen
		nopts.MaxPayload = math.MaxInt32
		// Create the NATS Server
		natsServer = natsd.New(nopts)
		// Start it as a go routine
		go natsServer.Start()
		// Wait for it to be able to accept connections
		if !natsServer.ReadyForConnections(10 * time.Second) {
			err = fmt.Errorf("[NATS]: Start NATS Server failed")
			log.Warn(err)
			return err
		}
		log.Info("[NATS]: Start listen: ", natsServer.Addr())
		// Now run the server with the streaming and streaming/nats options
		stanOpts.NATSServerURL = fmt.Sprintf("nats://%s", natsServer.Addr())
		if stanServer, err = stand.RunServerWithOpts(stanOpts, natsOpts); err != nil {
			log.With(log.Fields{
				"error":    err,
				"stanOpts": stanOpts,
				"natsOpts": natsOpts,
			}).Log(log.WarnLevel, "[NATS]: RunServerWithOpts failed")
			return err
		}
		natsURL = stanOpts.NATSServerURL
	}
	return nil
}

// StopServer shutdowns NATS
func StopServer() {
	mu.Lock()
	defer mu.Unlock()

	natsURL = ""
	if connDispatcher != nil {
		_ = connDispatcher.Close()
		connDispatcher = nil
	}
	if connPublisher != nil {
		_ = connPublisher.Close()
		connPublisher = nil
	}
	if stanServer != nil {
		stanServer.Shutdown()
		stanServer = nil
	}
	if natsServer != nil {
		natsServer.Shutdown()
		natsServer = nil
	}
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
	mu.Lock()
	defer mu.Unlock()

	if len(natsURL) == 0 {
		err = fmt.Errorf("[NATS]: unavailable")
		log.Warn(err)
		return err
	}
	if connDispatcher == nil {
		if connDispatcher, err = stan.Connect(
			clusterID,
			dispatcherID,
			stan.NatsURL(natsURL),
		); err != nil {
			return err
		}
	}
	cache.DispatcherWorkersCache.Flush()
	for _, opt := range options {
		ctrlChan <- opt
	}
	return nil
}

// StopDispatcher ends dispatching
func StopDispatcher() error {
	var err error
	mu.Lock()
	defer mu.Unlock()

	if connDispatcher != nil {
		err = connDispatcher.Close()
		connDispatcher = nil
	}
	return err
}

// Publish adds message in queue
func Publish(subject string, msg []byte) error {
	var err error
	mu.Lock()
	defer mu.Unlock()

	if len(natsURL) == 0 {
		err = fmt.Errorf("[NATS]: unavailable")
		log.Warn(err)
		return err
	}
	if connPublisher == nil {
		if connPublisher, err = stan.Connect(
			clusterID,
			publisherID,
			stan.NatsURL(natsURL),
			stan.MaxPubAcksInflight(cfg.MaxPubAcksInflight),
		); err != nil {
			log.With(log.Fields{
				"error":   err,
				"natsURL": natsURL,
			}).Log(log.WarnLevel, "[NATS]: connPublisher failed")
			return err
		}
	}
	return connPublisher.Publish(subject, msg)
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
					}).Log(log.WarnLevel, "[NATS]: startWorker failed")
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
			}).Log(log.InfoLevel, "[NATS]: Delivered")
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
			logEntry.WithField("retry", retry).Log(log.InfoLevel, "[NATS]: Not delivered: Will retry")

			go func() {
				ckWorker := opt.DurableName
				cache.DispatcherWorkersCache.Delete(ckWorker)
				_ = subscription.Close()
				time.Sleep(delay)
				ctrlChan <- opt
			}()
		} else {
			cache.DispatcherRetryCache.Delete(ckRetry)
			logEntry.Log(log.InfoLevel, "[NATS]: Not delivered: Stop retrying")
		}

	} else {
		logEntry.Log(log.InfoLevel, "[NATS]: Not delivered: Will not retry")
	}
}
