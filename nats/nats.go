package nats

import (
	"context"
	"errors"
	"expvar"
	"fmt"
	"maps"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/nats-io/nats-server/v2/logger"
	"github.com/nats-io/nats-server/v2/server"
	"github.com/nats-io/nats.go"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/shirou/gopsutil/v4/disk"
)

// Define NATS IDs
const (
	streamName = "tcg-stream"
)

var (
	ErrNATS       = fmt.Errorf("nats error")
	ErrDispatcher = fmt.Errorf("%w: dispatcher", ErrNATS)
	ErrPayloadLim = fmt.Errorf("%w: payload oversized limit", ErrNATS)

	subjects = []string{"tcg.>"}

	xClientURL = expvar.NewString("tcgNatsClientURL")
	xStats     = expvar.NewMap("tcgNatsStats")

	s = new(state)
	_ = func() int { s.pub = make(chan *nats.Msg, 2000); return 0 }()
)

type state struct {
	sync.Mutex

	config Config
	server *server.Server
	// if a client is too slow the server will eventually cut them off by closing the connection
	ncDispatcher *nats.Conn
	ncPublisher  *nats.Conn

	cancel context.CancelFunc
	pub    chan *nats.Msg
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

// DurableCfg defines subscription
type DurableCfg struct {
	Durable string
	Handler func(context.Context, *nats.Msg) error
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
		opts.NoSigs = true
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
				Str("natsOpts", fmt.Sprintf("%+v", *opts)).
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
				e.Str("natsOpts", fmt.Sprintf("%+v", *opts))
			}
		}).
		Msgf("nats started at: %s", s.server.ClientURL())
	xClientURL.Set(s.server.ClientURL())

	nc, err := nats.Connect(s.server.ClientURL())
	if err != nil {
		log.Err(err).Msg("nats failed Connect")
		return err
	}
	s.ncPublisher = nc

	if err := defineStream(nc); err != nil {
		log.Err(err).Msg("nats failed defineStream")
		return err
	}

	ctx, cancel := context.WithCancel(context.Background())
	s.cancel = cancel
	go handlePubchan(ctx)
	return nil
}

func defineStream(nc *nats.Conn) error {
	storage := func(arg string) nats.StorageType {
		switch strings.ToUpper(arg) {
		case "MEMORY":
			return nats.MemoryStorage
		default:
			return nats.FileStorage
		}
	}(s.config.StoreType)
	sc := nats.StreamConfig{
		Name:        streamName,
		Subjects:    subjects,
		Storage:     storage,
		AllowDirect: true,
		MaxAge:      s.config.StoreMaxAge,
		MaxBytes:    s.config.StoreMaxBytes,
		MaxMsgs:     s.config.StoreMaxMsgs,
		Retention:   nats.LimitsPolicy,
	}

	js, err := nc.JetStream(nats.DirectGet())
	if err != nil {
		log.Err(err).Msg("nats failed JetStream")
		return err
	}

	fn, fnDesc := js.UpdateStream, "UpdateStream"
	if info, err := js.StreamInfo(streamName); err == nil {
		if equalStreamConfig(sc, info.Config) {
			return nil
		}
	} else if err == nats.ErrStreamNotFound {
		fn, fnDesc = js.AddStream, "AddStream"
	} else {
		log.Err(err).Msg("nats failed StreamInfo")
		return err
	}

	_, err = fn(&sc)
	if err == nil {
		return nil
	} else if !isJSStorageErr(err) || sc.Storage != nats.FileStorage {
		log.Err(err).
			Str("config", fmt.Sprintf("%+v", sc)).
			Msgf("nats failed %v", fnDesc)
		return err
	}

	/* retry with smaller storage, 5/8 that smaller then 3/4
	NATS Server allows up to 75% of available storage.
	https://github.com/nats-io/nats-server/blob/v2.9.19/server/disk_avail.go */
	u, errUsage := disk.Usage(s.config.StoreDir)
	if errUsage != nil {
		log.Err(err).
			Str("config", fmt.Sprintf("%+v", sc)).
			Str("diskUsage", errUsage.Error()).
			Msgf("nats failed %v, could not repair due to disk.Usage error", fnDesc)
		return err
	}
	if u.Free/8*5 > uint64(sc.MaxBytes) {
		log.Err(err).
			Str("config", fmt.Sprintf("%+v", sc)).
			Str("diskUsage", fmt.Sprintf("%+v", *u)).
			Msgf("nats failed %v, could not repair due to unexpected disk.Free", fnDesc)
		return err
	}
	if u.Free < 1024*1024 {
		log.Err(err).
			Str("config", fmt.Sprintf("%+v", sc)).
			Str("diskUsage", fmt.Sprintf("%+v", *u)).
			Msgf("nats failed %v, could not repair due to low disk.Free", fnDesc)
		return err
	}

	origCfg, origErr := sc, err
	mb := int64(u.Free) / 8 * 5
	sc.MaxBytes = mb
	_, err = fn(&sc)
	log.Err(err).
		Str("originalError", origErr.Error()).
		Str("originalConfig", fmt.Sprintf("%+v", origCfg)).
		Int64("reducedMaxBytes", mb).
		Uint64("disk.Free", u.Free).
		Msgf("nats retrying %v with smaller storage", fnDesc)
	return err
}

func equalStreamConfig(c1, c2 nats.StreamConfig) bool {
	return c1.MaxAge == c2.MaxAge &&
		c1.MaxBytes == c2.MaxBytes &&
		c1.MaxMsgs == c2.MaxMsgs &&
		c1.Storage == c2.Storage
}

func isJSStorageErr(err error) bool {
	// Note: there is some inconsistency in public nats constants
	codes := map[nats.ErrorCode]string{
		10023: "nats.JSErrCodeInsufficientResourcesErr",
		10047: "nats.JSStorageResourcesExceededErr", // missed as public const
	}
	var apiErr *nats.APIError
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

	s.cancel()
	var wg sync.WaitGroup
	if s.ncPublisher != nil {
		wg.Add(1)
		s.ncPublisher.SetClosedHandler(func(*nats.Conn) { wg.Done() })
		if err := s.ncPublisher.Drain(); err != nil {
			log.Warn().Err(err).Msg("could not drain nats publisher connection")
		}
		s.ncPublisher = nil
	}
	if s.ncDispatcher != nil {
		wg.Add(1)
		s.ncDispatcher.SetClosedHandler(func(*nats.Conn) { wg.Done() })
		if err := s.ncDispatcher.Drain(); err != nil {
			log.Warn().Err(err).Msg("could not drain nats dispatcher connection")
		}
		s.ncDispatcher = nil
	}
	wg.Wait()

	if s.server != nil {
		s.server.Shutdown()
		s.server = nil
	}
	log.Info().Msg("nats stopped")
	xClientURL.Set("")
}

// StartDispatcher connects to stan and adds durable subscriptions
func StartDispatcher(options []DurableCfg) error {
	if err := StopDispatcher(); err != nil {
		return err
	}
	d := getDispatcher()
	d.Lock()
	defer d.Unlock()

	if d.server == nil {
		err := fmt.Errorf("%w: unavailable", ErrNATS)
		log.Err(err).Msg("nats dispatcher failed")
		return err
	}
	if d.ncDispatcher == nil {
		nc, err := nats.Connect(s.server.ClientURL())
		if err != nil {
			log.Err(err).Msg("nats dispatcher failed Connect")
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
	var wg sync.WaitGroup
	if d.ncDispatcher != nil {
		if js, err := d.ncDispatcher.JetStream(); err == nil {
			if info, err := js.StreamInfo(streamName); err == nil {
				ze = ze.Str("streamState", fmt.Sprintf("%+v", info.State))
			}
		}
		wg.Add(1)
		s.ncDispatcher.SetClosedHandler(func(*nats.Conn) { wg.Done() })
		if err := d.ncDispatcher.Drain(); err != nil {
			log.Warn().Err(err).Msg("could not drain nats dispatcher connection")
		}
		d.ncDispatcher = nil
	}
	wg.Wait()
	ze.Msg("dispatcher stopped")
	return nil
}

// Pub sends NATS message in buffered channel
func Pub(subj string, data []byte, header http.Header) error {
	if len(data) > int(s.config.MaxPayload) {
		err := fmt.Errorf("%w: %v / %v / %v",
			ErrPayloadLim, subj, s.config.MaxPayload, len(data))
		log.Err(err).Msg("nats publisher failed")
		return err
	}
	msg := nats.NewMsg(subj)
	msg.Data = data
	maps.Copy(msg.Header, header)
	// use goroutine as L2 buffer
	go func(msg *nats.Msg) { s.pub <- msg }(msg)
	return nil
}

// Publish sends NATS message
//
// Deprecated: Use Pub
func Publish(subj string, data []byte, header http.Header) error {
	if len(data) > int(s.config.MaxPayload) {
		err := fmt.Errorf("%w: %v / %v / %v",
			ErrPayloadLim, subj, s.config.MaxPayload, len(data))
		log.Err(err).Msg("nats publisher failed")
		return err
	}

	s.Lock()
	defer s.Unlock()

	if s.server == nil {
		err := fmt.Errorf("%w: unavailable", ErrNATS)
		log.Err(err).Msg("nats publisher failed")
		return err
	}
	if s.ncPublisher == nil {
		nc, err := nats.Connect(s.server.ClientURL())
		if err != nil {
			log.Err(err).Msg("nats publisher failed Connect")
			return err
		}
		s.ncPublisher = nc
	}

	msg := nats.NewMsg(subj)
	msg.Data = data
	maps.Copy(msg.Header, header)
	return s.ncPublisher.PublishMsg(msg)
}

func IsStartedDispatcher() bool {
	return s != nil && s.ncDispatcher != nil
}

func IsStartedServer() bool {
	return s != nil && s.server != nil
}

func handlePubchan(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case msg := <-s.pub:
			if err := s.ncPublisher.PublishMsg(msg); err != nil {
				log.Warn().Err(err).
					Str("header", fmt.Sprintf("%+v", msg.Header)).
					Msg("nats failed PublishMsg: reconnecting")
				if nc, err := nats.Connect(s.server.ClientURL()); err == nil {
					s.ncPublisher = nc
					if err := s.ncPublisher.PublishMsg(msg); err != nil {
						log.Warn().Err(err).
							Str("header", fmt.Sprintf("%+v", msg.Header)).
							Msg("nats failed PublishMsg")
					}
				} else {
					log.Warn().Err(err).Msg("nats failed reConnect")
					time.Sleep(time.Second)
				}
			}
		}
	}
}
