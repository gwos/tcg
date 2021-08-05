package nats

import (
	"fmt"
	"sync"
	"time"

	natsd "github.com/nats-io/nats-server/v2/server"
	stand "github.com/nats-io/nats-streaming-server/server"
	"github.com/nats-io/nats-streaming-server/stores"
	"github.com/nats-io/stan.go"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

// Define NATS IDs
const (
	clusterID    = "tcg-cluster"
	dispatcherID = "tcg-dispatcher"
	publisherID  = "tcg-publisher"
)

var (
	ErrNATS       = fmt.Errorf("nats error")
	ErrDispatcher = fmt.Errorf("%w: dispatcher", ErrNATS)

	s = new(state)
)

type state struct {
	sync.Mutex

	config     Config
	stanServer *stand.StanServer
	// if a client is too slow the server will eventually cut them off by closing the connection
	connDispatcher stan.Conn
	connPublisher  stan.Conn
}

// Config defines NATS configurable options
type Config struct {
	AckWait             time.Duration
	MaxInflight         int
	MaxPubAcksInflight  int
	MaxPayload          int32
	MaxPendingBytes     int
	MaxPendingMsgs      int
	MonitorPort         int
	StoreDir            string
	StoreType           string
	StoreMaxAge         time.Duration
	StoreMaxBytes       int64
	StoreBufferSize     int
	StoreReadBufferSize int
}

// DispatcherOption defines subscription
type DispatcherOption struct {
	DurableName string
	Subject     string
	Handler     func([]byte) error
}

// StartServer runs NATS
func StartServer(config Config) error {
	natsOpts := stand.DefaultNatsServerOptions.Clone()
	natsOpts.MaxPayload = config.MaxPayload
	natsOpts.HTTPHost = "127.0.0.1"
	natsOpts.HTTPPort = config.MonitorPort
	natsOpts.Port = natsd.RANDOM_PORT

	stanOpts := stand.GetDefaultOptions().Clone()
	stanOpts.ID = clusterID
	stanOpts.FilestoreDir = config.StoreDir
	switch config.StoreType {
	case "MEMORY":
		stanOpts.StoreType = stores.TypeMemory
	case "FILE":
		stanOpts.StoreType = stores.TypeFile
	default:
		stanOpts.StoreType = stores.TypeFile
	}
	stanOpts.StoreLimits.MaxAge = config.StoreMaxAge
	stanOpts.StoreLimits.MaxBytes = config.StoreMaxBytes
	stanOpts.FileStoreOpts.BufferSize = config.StoreBufferSize
	stanOpts.FileStoreOpts.ReadBufferSize = config.StoreReadBufferSize

	s.Lock()
	defer s.Unlock()

	s.config = config
	if s.stanServer == nil {
		if stanServer, err := stand.RunServerWithOpts(stanOpts, natsOpts); err == nil {
			s.stanServer = stanServer
			log.Info().
				Func(func(e *zerolog.Event) {
					if zerolog.GlobalLevel() <= zerolog.DebugLevel {
						e.Interface("stanOpts", stanOpts).
							Interface("natsOpts", natsOpts)
					}
				}).
				Msgf("nats started at: %s", s.stanServer.ClientURL())
		} else {
			log.Warn().
				Err(err).
				Interface("stanOpts", stanOpts).
				Interface("natsOpts", natsOpts).
				Msg("nats RunServerWithOpts failed")
			return err
		}
	}
	return nil
}

// StopServer shutdowns NATS
func StopServer() {
	s.Lock()
	defer s.Unlock()

	if s.connDispatcher != nil {
		_ = s.connDispatcher.Close()
		s.connDispatcher = nil
	}
	if s.connPublisher != nil {
		_ = s.connPublisher.Close()
		s.connPublisher = nil
	}
	if s.stanServer != nil {
		s.stanServer.Shutdown()
		s.stanServer = nil
	}
}

// StartDispatcher connects to stan and adds durable subscriptions
func StartDispatcher(options []DispatcherOption) error {
	if err := StopDispatcher(); err != nil {
		return err
	}
	d := getDispatcher()
	d.Lock()
	defer d.Unlock()

	if d.connDispatcher == nil {
		if d.stanServer == nil {
			err := fmt.Errorf("%v: unavailable", ErrNATS)
			log.Warn().Err(err).Msg("nats dispatcher failed")
			return err
		}
		var err error
		if d.connDispatcher, err = stan.Connect(
			clusterID,
			dispatcherID,
			stan.NatsURL(d.stanServer.ClientURL()),
			stan.SetConnectionLostHandler(func(c stan.Conn, e error) {
				log.Warn().Err(e).Msg("nats dispatcher invoked ConnectionLostHandler")
				d.Lock()
				_ = c.Close()
				d.connDispatcher = nil
				d.Unlock()
				StartDispatcher(options)
			}),
		); err != nil {
			log.Warn().Err(err).Msg("nats dispatcher failed to connect")
			return err
		}
	}

	d.durables.Flush()
	for _, opt := range options {
		if err := d.retryDurable(opt); err != nil {
			return err
		}
	}
	return nil
}

// StopDispatcher ends dispatching
func StopDispatcher() error {
	d := getDispatcher()
	d.Lock()
	defer d.Unlock()

	var err error
	if d.connDispatcher != nil {
		err = d.connDispatcher.Close()
		d.connDispatcher = nil
	}
	d.durables.Flush()
	d.msgsDone.Flush()
	d.retryes.Flush()
	return err
}

// Publish adds message in queue
func Publish(subject string, msg []byte) error {
	s.Lock()
	defer s.Unlock()

	if s.connPublisher == nil {
		if s.stanServer == nil {
			err := fmt.Errorf("%v: unavailable", ErrNATS)
			log.Warn().Err(err).Msg("nats publisher failed")
			return err
		}
		var err error
		if s.connPublisher, err = stan.Connect(
			clusterID,
			publisherID,
			stan.NatsURL(s.stanServer.ClientURL()),
			stan.MaxPubAcksInflight(s.config.MaxPubAcksInflight),
			stan.SetConnectionLostHandler(func(c stan.Conn, e error) {
				log.Warn().Err(e).Msg("nats publisher invoked ConnectionLostHandler")
				s.Lock()
				_ = c.Close()
				s.connPublisher = nil
				s.Unlock()
			}),
		); err != nil {
			log.Warn().Err(err).Msg("nats publisher failed to connect")
			return err
		}
	}
	return s.connPublisher.Publish(subject, msg)
}
