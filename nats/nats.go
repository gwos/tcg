package nats

import (
	"context"
	"errors"
	"expvar"
	"fmt"
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

	s        = new(state)
	subjects = []string{"tcg.>"}

	xClientURL = expvar.NewString("tcgNatsClientURL")
	xStats     = expvar.NewMap("tcgNatsStats")
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
	Subject string
	Handler func(context.Context, NatsMsg) error
}

type NatsMsg struct {
	*nats.Msg
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

	if info, err := js.StreamInfo(streamName); err == nil {
		if err := doStream(js, &info.Config, &sc); err != nil {
			return err
		}
	} else if err == nats.ErrStreamNotFound {
		if err := doStream(js, nil, &sc); err != nil {
			return err
		}
	} else {
		log.Err(err).Msg("nats failed StreamInfo")
		return err
	}

	return nil
}

func doStream(js nats.JetStreamContext, curCfg, newCfg *nats.StreamConfig) error {
	fn, fnDesc := js.AddStream, "AddStream"
	if curCfg != nil {
		if equalStreamConfig(*curCfg, *newCfg) {
			return nil
		}
		fn, fnDesc = js.UpdateStream, "UpdateStream"
	}

	_, err := fn(newCfg)

	chkAPIErr := func(err error) bool {
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

	if newCfg.Storage == nats.FileStorage && chkAPIErr(err) {
		/* retry with smaller storage, 5/8 that smaller then 3/4
		NATS Server allows up to 75% of available storage.
		https://github.com/nats-io/nats-server/blob/v2.9.19/server/disk_avail.go */
		u, errUsage := disk.Usage(s.config.StoreDir)
		if errUsage != nil {
			log.Err(err).
				Str("config", fmt.Sprintf("%+v", *newCfg)).
				Str("diskUsage", errUsage.Error()).
				Msgf("nats failed %v, could not repair due to disk.Usage error", fnDesc)
			return err
		}
		if u.Free/8*5 > uint64(newCfg.MaxBytes) {
			log.Err(err).
				Str("config", fmt.Sprintf("%+v", *newCfg)).
				Str("diskUsage", fmt.Sprintf("%+v", *u)).
				Msgf("nats failed %v, could not repair due to unexpected disk.Free", fnDesc)
			return err
		}
		if u.Free < 1024*1024 {
			log.Err(err).
				Str("config", fmt.Sprintf("%+v", *newCfg)).
				Str("diskUsage", fmt.Sprintf("%+v", *u)).
				Msgf("nats failed %v, could not repair due to low disk.Free", fnDesc)
			return err
		}

		mb := int64(u.Free) / 8 * 5
		origCfg := *newCfg
		newCfg.MaxBytes = mb
		_, err2 := fn(newCfg)
		log.Err(err2).
			Str("originalError", err.Error()).
			Str("originalConfig", fmt.Sprintf("%+v", origCfg)).
			Int64("reducedMaxBytes", mb).
			Uint64("disk.Free", u.Free).
			Msgf("nats retrying %v with smaller storage", fnDesc)
		return err2
	}

	log.Err(err).
		Str("config", fmt.Sprintf("%+v", *newCfg)).
		Msgf("nats %v", fnDesc)
	return err
}

func equalStreamConfig(c1, c2 nats.StreamConfig) bool {
	return c1.MaxAge == c2.MaxAge &&
		c1.MaxBytes == c2.MaxBytes &&
		c1.MaxMsgs == c2.MaxMsgs &&
		c1.Storage == c2.Storage &&
		equalStrs(c1.Subjects, c2.Subjects)
}

func equalStrs(ss1, ss2 []string) bool {
	if len(ss1) != len(ss2) {
		return false
	}
	ms := make(map[string]bool, len(ss1))
	for _, s := range ss1 {
		ms[s] = false
	}
	for _, s := range ss2 {
		if _, ok := ms[s]; !ok {
			return false
		}
	}
	return true
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
	if d.ncDispatcher != nil {
		d.ncDispatcher.Close()
		d.ncDispatcher = nil
	}

	log.Info().Msg("dispatcher stopped")
	return nil
}

// Publish adds payload in queue with optional key-value headers
func Publish(subj string, data []byte, headers ...string) error {
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

	if len(headers) < 2 {
		return s.ncPublisher.Publish(subj, data)
	}

	msg := nats.NewMsg(subj)
	msg.Data = data
	for i := 0; i < len(headers)-1; i += 2 {
		msg.Header.Add(headers[i], headers[i+1])
	}
	return s.ncPublisher.PublishMsg(msg)
}

func IsStartedDispatcher() bool {
	return s != nil && s.ncDispatcher != nil
}

func IsStartedServer() bool {
	return s != nil && s.server != nil
}
