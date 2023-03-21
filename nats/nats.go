package nats

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/nats-io/nats-server/v2/server"
	"github.com/nats-io/nats.go"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

// Define NATS IDs
const (
	// tcgClusterID  = "tcg-cluster"
	tcgStreamName = "tcg-stream"

	subjDowntime         = "downtime"
	subjEvents           = "events"
	subjInventoryMetrics = "inventory-metrics"
)

var (
	ErrNATS       = fmt.Errorf("nats error")
	ErrDispatcher = fmt.Errorf("%w: dispatcher", ErrNATS)

	s        = new(state)
	subjects = []string{subjDowntime, subjEvents, subjInventoryMetrics}
)

type state struct {
	sync.Mutex

	config Config
	server *server.Server
	// if a client is too slow the server will eventually cut them off by closing the connection
	jsDispatcher nats.JetStreamContext
	jsPublisher  nats.JetStreamContext
}

// Config defines NATS configurable options
type Config struct {
	AckWait             time.Duration
	MaxInflight         int
	MaxPubAcksInflight  int
	MaxPayload          int32
	MonitorPort         int
	StoreDir            string
	StoreType           string
	StoreMaxAge         time.Duration
	StoreMaxBytes       int64
	StoreMaxMsgs        int
	StoreBufferSize     int
	StoreReadBufferSize int

	ConfigFile string
}

// DispatcherOption defines subscription
type DispatcherOption struct {
	Durable string
	Subject string
	Handler func([]byte) error
}

// StartServer runs NATS
func StartServer(config Config) error {
	opts := new(server.Options)

	if config.ConfigFile != "" {
		if _, err := os.Open(config.ConfigFile); err != nil {
			return errors.New("invalid path to config file specified")
		}
		if err := opts.ProcessConfigFile(config.ConfigFile); err != nil {
			return err
		}
	} else {
		// opts.Cluster.Name = tcgClusterID
		opts.StoreDir = config.StoreDir
		opts.JetStream = true
		opts.JetStreamLimits = server.JSLimitOpts{}
		opts.JetStreamMaxStore = config.StoreMaxBytes
		opts.MaxPayload = config.MaxPayload
		opts.HTTPHost = "127.0.0.1"
		opts.HTTPPort = config.MonitorPort
		opts.Host = "127.0.0.1"
		opts.Port = server.RANDOM_PORT

		opts.Debug = zerolog.GlobalLevel() <= zerolog.DebugLevel
	}

	s.Lock()
	defer s.Unlock()

	s.config = config
	if s.server == nil {
		if natsServer, err := server.NewServer(opts); err == nil {
			s.server = natsServer
		} else {
			log.Warn().
				Err(err).
				Interface("natsOpts", *opts).
				Msg("nats NewServer failed")
			return err
		}
	}

	if zerolog.GlobalLevel() <= zerolog.DebugLevel {
		s.server.ConfigureLogger()
	}

	s.server.Start()
	log.Info().
		Func(func(e *zerolog.Event) {
			if zerolog.GlobalLevel() <= zerolog.DebugLevel {
				e.Interface("natsOpts", *opts)
			}
		}).
		Msgf("nats started at: %s", s.server.ClientURL())

	return nil
}

// StopServer shutdowns NATS
func StopServer() {
	s.Lock()
	defer s.Unlock()

	if s.jsDispatcher != nil {
		s.jsDispatcher = nil
	}
	if s.jsPublisher != nil {
		s.jsPublisher = nil
	}
	if s.server != nil {
		s.server.Shutdown()
		s.server = nil
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

	if d.jsDispatcher == nil {
		if d.server == nil {
			err := fmt.Errorf("%v: unavailable", ErrNATS)
			log.Warn().Err(err).Msg("nats dispatcher failed")
			return err
		}

		nc, err := nats.Connect(
			d.server.ClientURL(),
			nats.RetryOnFailedConnect(true),
		)
		if err != nil {
			log.Warn().Err(err).Msg("nats dispatcher failed to connect")
			return err
		}

		js, err := nc.JetStream(
			nats.DirectGet(),
		)
		if err != nil {
			return err
		}

		if _, err = js.StreamInfo(tcgStreamName); err != nil {
			if err != nats.ErrStreamNotFound {
				return err
			}

			if _, err = js.AddStream(&nats.StreamConfig{
				Name:      tcgStreamName,
				Subjects:  subjects,
				Retention: nats.LimitsPolicy,
				Storage: func() nats.StorageType {
					if strings.ToUpper(d.config.StoreType) == "MEMORY" {
						return nats.MemoryStorage
					}
					return nats.FileStorage
				}(),
				MaxAge:      d.config.StoreMaxAge,
				AllowDirect: true,
			}); err != nil {
				return err
			}
		}

		d.jsDispatcher = js
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

	subs := d.durables.Items()
	for _, value := range subs {
		_ = value.Object.(*nats.Subscription).Unsubscribe()
	}

	d.jsDispatcher = nil
	d.durables.Flush()
	d.msgsDone.Flush()
	d.retries.Flush()
	return nil
}

// Publish adds message in queue
func Publish(subject string, msg []byte) error {
	if len(msg) > int(s.config.MaxPayload) {
		return fmt.Errorf("%v: payload too big for limit %v: %v",
			subject, s.config.MaxPayload, len(msg))
	}

	s.Lock()
	defer s.Unlock()

	if s.jsPublisher == nil {
		if s.server == nil {
			err := fmt.Errorf("%v: unavailable", ErrNATS)
			log.Warn().Err(err).Msg("nats publisher failed")
			return err
		}

		nc, err := nats.Connect(
			s.server.ClientURL(),
			nats.RetryOnFailedConnect(true),
		)
		if err != nil {
			log.Warn().Err(err).Msg("nats publisher failed to connect")
			return err
		}

		js, err := nc.JetStream()
		if err != nil {
			log.Warn().Err(err).Msg("nats publisher failed to JetStream")
			return err
		}

		s.jsPublisher = js
	}

	_, err := s.jsPublisher.Publish(subject, msg)

	return err
}
