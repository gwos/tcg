package nats

import (
	"context"
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
	streamName = "tcg-stream"
)

var (
	ErrNATS       = fmt.Errorf("nats error")
	ErrDispatcher = fmt.Errorf("%w: dispatcher", ErrNATS)

	s        = new(state)
	subjects = []string{"tcg.>"}
)

type state struct {
	sync.Mutex

	config Config
	server *server.Server
	// if a client is too slow the server will eventually cut them off by closing the connection
	ncDispatcher *nats.Conn
	ncPublisher  *nats.Conn
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
		opts.HTTPHost = "127.0.0.1"
		opts.HTTPPort = config.MonitorPort
		opts.Host = "127.0.0.1"
		opts.Port = server.RANDOM_PORT
		opts.StoreDir = config.StoreDir
		opts.MaxPayload = config.MaxPayload

		opts.JetStream = true
		opts.JetStreamLimits = server.JSLimitOpts{}
		opts.JetStreamMaxStore = config.StoreMaxBytes

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
				Msg("nats failed NewServer")
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

	nc, err := nats.Connect(s.server.ClientURL())
	if err != nil {
		log.Warn().Err(err).Msg("nats failed Connect")
		return err
	}
	s.ncPublisher = nc

	return defineStream(nc, streamName, subjects)
}

func defineStream(nc *nats.Conn, streamName string, subjects []string) error {
	js, err := nc.JetStream(nats.DirectGet())
	if err != nil {
		log.Warn().Err(err).Msg("nats failed JetStream")
		return err
	}

	if _, err = js.StreamInfo(streamName); err == nats.ErrStreamNotFound {
		if _, err = js.AddStream(&nats.StreamConfig{
			Name:     streamName,
			Subjects: subjects,
			Storage: func(arg string) nats.StorageType {
				switch strings.ToUpper(arg) {
				case "MEMORY":
					return nats.MemoryStorage
				default:
					return nats.FileStorage
				}
			}(s.config.StoreType),
			AllowDirect: true,
			MaxAge:      s.config.StoreMaxAge,
			Retention:   nats.LimitsPolicy,
		}); err != nil {
			log.Warn().Err(err).Msg("nats failed AddStream")
			return err
		}
	} else if err != nil {
		log.Warn().Err(err).Msg("nats failed StreamInfo")
		return err
	}

	return nil
}

// StopServer shutdowns NATS
func StopServer() {
	s.Lock()
	defer s.Unlock()

	if s.ncPublisher != nil {
		s.ncPublisher.Close()
		s.ncPublisher = nil
	}
	if s.ncDispatcher != nil {
		s.ncDispatcher.Close()
		s.ncDispatcher = nil
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

	if d.server == nil {
		err := fmt.Errorf("%v: unavailable", ErrNATS)
		log.Warn().Err(err).Msg("nats dispatcher failed")
		return err
	}
	if d.ncDispatcher == nil {
		nc, err := nats.Connect(s.server.ClientURL())
		if err != nil {
			log.Warn().Err(err).Msg("nats dispatcher failed Connect")
			return err
		}
		d.ncDispatcher = nc
	}

	d.retries.Flush()
	ctx, cancel := context.WithCancel(context.Background())
	d.cancel = cancel
	for _, opt := range options {
		d.OpenDurable(ctx, opt)
	}
	return nil
}

// StopDispatcher ends dispatching
func StopDispatcher() error {
	d := getDispatcher()
	d.Lock()
	defer d.Unlock()

	if d.cancel != nil {
		d.cancel()
		d.cancel = nil
	}
	if d.ncDispatcher != nil {
		d.ncDispatcher.Close()
		d.ncDispatcher = nil
	}

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

	if s.server == nil {
		err := fmt.Errorf("%v: unavailable", ErrNATS)
		log.Warn().Err(err).Msg("nats publisher failed")
		return err
	}
	if s.ncPublisher == nil {
		nc, err := nats.Connect(s.server.ClientURL())
		if err != nil {
			log.Warn().Err(err).Msg("nats publisher failed Connect")
			return err
		}
		s.ncPublisher = nc
	}

	return s.ncPublisher.Publish(subject, msg)
}
