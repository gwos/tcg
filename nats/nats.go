package nats

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/nats-io/nats-server/v2/logger"
	"github.com/nats-io/nats-server/v2/server"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/shirou/gopsutil/v3/disk"
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

	onPause    bool
	onPauseBuf *RBuf
}

// Config defines NATS configurable options
type Config struct {
	AckWait            time.Duration
	LogColors          bool
	MaxInflight        int
	MaxPubAcksInflight int
	MaxPayload         int32
	MonitorPort        int
	StoreDir           string
	StoreType          string
	StoreMaxAge        time.Duration
	StoreMaxBytes      int64
	StoreMaxMsgs       int64

	ConfigFile string
}

// DispatcherOption defines subscription
type DispatcherOption struct {
	Durable string
	Subject string
	Handler func(context.Context, []byte) error
}

// StartServer runs NATS
func StartServer(config Config) error {
	if s.server != nil {
		log.Info().
			Msgf("nats already started at: %s", s.server.ClientURL())
		return nil
	}

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

	s.server.SetLogger(logger.NewStdLogger(true, false, false, config.LogColors, false),
		zerolog.GlobalLevel() <= zerolog.DebugLevel, false)

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

	return defineStream(nc)
}

func defineStream(nc *nats.Conn) error {
	storage := func(arg string) jetstream.StorageType {
		switch strings.ToUpper(arg) {
		case "MEMORY":
			return jetstream.MemoryStorage
		default:
			return jetstream.FileStorage
		}
	}(s.config.StoreType)
	sc := jetstream.StreamConfig{
		Name:        streamName,
		Subjects:    subjects,
		Storage:     storage,
		AllowDirect: true,
		MaxAge:      s.config.StoreMaxAge,
		MaxBytes:    s.config.StoreMaxBytes,
		MaxMsgs:     s.config.StoreMaxMsgs,
		Retention:   jetstream.LimitsPolicy,
	}

	js, err := jetstream.New(nc)
	if err != nil {
		log.Err(err).Msg("nats failed JetStream")
		return err
	}

	ctx := context.Background()
	fn, fnDesc := js.UpdateStream, "UpdateStream"
	stream, err := js.Stream(ctx, streamName)
	if err == jetstream.ErrStreamNotFound {
		fn, fnDesc = js.CreateStream, "CreateStream"
	} else if err != nil {
		log.Err(err).Msg("nats failed Stream")
		return err
	} else {
		info, err := stream.Info(ctx)
		if err != nil {
			log.Err(err).Msg("nats failed JetStream Info")
			return err
		}
		if equalStreamConfig(sc, info.Config) {
			return nil
		}
	}

	_, err = fn(ctx, sc)
	if err == nil {
		return nil
	} else if !isJSStorageErr(err) || sc.Storage != jetstream.FileStorage {
		log.Err(err).
			Interface("config", sc).
			Msgf("nats failed %v", fnDesc)
		return err
	}

	/* retry with smaller storage, 5/8 that smaller then 3/4
	NATS Server allows up to 75% of available storage.
	https://github.com/nats-io/nats-server/blob/v2.9.19/server/disk_avail.go */
	u, errUsage := disk.Usage(s.config.StoreDir)
	if errUsage != nil {
		log.Err(err).
			Interface("config", sc).
			Interface("diskUsage", errUsage).
			Msgf("nats failed %v, could not repair due to disk.Usage error", fnDesc)
		return err
	}
	if u.Free/8*5 > uint64(sc.MaxBytes) {
		log.Err(err).
			Interface("config", sc).
			Interface("diskUsage", u).
			Msgf("nats failed %v, could not repair due to unexpected disk.Free", fnDesc)
		return err
	}
	if u.Free < 1024*1024 {
		log.Err(err).
			Interface("config", sc).
			Interface("diskUsage", u).
			Msgf("nats failed %v, could not repair due to low disk.Free", fnDesc)
		return err
	}

	mb := int64(u.Free) / 8 * 5
	origCfg, origErr := sc, err
	sc.MaxBytes = mb
	_, err = fn(ctx, sc)
	log.Err(err).
		Interface("originalError", origErr).
		Interface("originalConfig", origCfg).
		Interface("reducedMaxBytes", mb).
		Interface("disk.Free", u.Free).
		Msgf("nats retrying %v with smaller storage", fnDesc)
	return err
}

func equalStreamConfig(c1, c2 jetstream.StreamConfig) bool {
	return c1.MaxAge == c2.MaxAge &&
		c1.MaxBytes == c2.MaxBytes &&
		c1.MaxMsgs == c2.MaxMsgs &&
		c1.Storage == c2.Storage
}

func isJSStorageErr(err error) bool {
	// Note: there is some inconsistency in public nats constants
	codes := map[jetstream.ErrorCode]string{
		10023: "nats.JSErrCodeInsufficientResourcesErr",
		10047: "nats.JSStorageResourcesExceededErr", // missed as public const
	}
	var apiErr *jetstream.APIError
	if errors.As(err, &apiErr) {
		_, ok := codes[apiErr.ErrorCode]
		return ok
	}
	return false
}

// StopServer shutdowns NATS
func StopServer() {
	s.Lock()
	defer s.Unlock()

	if s.ncPublisher != nil {
		if err := s.ncPublisher.Drain(); err != nil {
			log.Warn().Err(err).Msg("could not drain nats publisher connection")
			s.ncPublisher.Close()
		}
		s.ncPublisher = nil
	}
	if s.ncDispatcher != nil {
		if err := s.ncDispatcher.Drain(); err != nil {
			log.Warn().Err(err).Msg("could not drain nats dispatcher connection")
			s.ncDispatcher.Close()
		}
		s.ncDispatcher = nil
	}
	if s.server != nil {
		s.server.Shutdown()
		s.server = nil
	}
	log.Info().Msg("nats stopped")
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
		err := fmt.Errorf("%w: unavailable", ErrNATS)
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

	d.Flush()
	ctx, cancel := context.WithCancel(context.Background())
	d.cancel = cancel
	for _, opt := range options {
		d.OpenDurable(ctx, opt)
	}

	log.Info().Msg("dispatcher started")
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

	ze := log.Info()
	if d.ncDispatcher != nil {
		js, _ := d.ncDispatcher.JetStream()
		info, _ := js.StreamInfo(streamName)
		ze = ze.Interface("streamState", info.State)

		if err := d.ncDispatcher.Drain(); err != nil {
			log.Warn().Err(err).Msg("could not drain nats dispatcher connection")
			d.ncDispatcher.Close()
		}
		d.ncDispatcher = nil
	}

	ze.Msg("dispatcher stopped")
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

	if s.onPause {
		_, err := s.onPauseBuf.WriteMsg(subject, msg)
		return err
	}

	if s.server == nil {
		err := fmt.Errorf("%w: unavailable", ErrNATS)
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

func Pause() error {
	s.Lock()
	defer s.Unlock()
	if s.onPause {
		return nil
	}
	s.onPause = true
	s.onPauseBuf = &RBuf{Size: 200}
	return nil
}

func Unpause() error {
	s.Lock()
	defer s.Unlock()
	if !s.onPause {
		return nil
	}
	s.onPause = false
	for _, bufMsg := range s.onPauseBuf.Records() {
		if err := s.ncPublisher.Publish(bufMsg.subj, bufMsg.msg); err != nil {
			log.Warn().Err(err).Str("subject", bufMsg.subj).
				Msg("could not publish buffered message")
		}
	}
	return nil
}
