package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path"
	"strings"
	"sync"
	"time"

	"github.com/gwos/tcg/logzer"
	"github.com/gwos/tcg/sdk/clients"
	"github.com/gwos/tcg/sdk/logper"
	"github.com/gwos/tcg/sdk/transit"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	tracesdk "go.opentelemetry.io/otel/sdk/trace"
	"gopkg.in/yaml.v3"
)

var (
	once sync.Once
	cfg  *Config

	liblogger zerolog.Logger
)

const (
	InstallationModeEnv = "INSTALLATION_MODE"
	InstallationModeCMC = "CHILD_MANAGED_CHILD"
	InstallationModePMC = "PARENT_MANAGED_CHILD"
	InstallationModeP   = "PARENT"
	InstallationModeS   = "STANDALONE"

	SecVerPrefix = "_v1_"
)

// LogLevel defines levels in logrus-style
type LogLevel int

// Enum levels
const (
	Error LogLevel = iota
	Warn
	Info
	Debug
)

func (l LogLevel) String() string {
	return [...]string{"Error", "Warn", "Info", "Debug"}[l]
}

// Connector defines TCG Connector configuration
// see GetConfig() for defaults
type Connector struct {
	transit.AgentIdentity `yaml:",inline"`

	BatchEvents   time.Duration `yaml:"batchEvents"`
	BatchMetrics  time.Duration `yaml:"batchMetrics"`
	BatchMaxBytes int           `yaml:"batchMaxBytes"`

	// ControllerAddr accepts value for combined "host:port"
	// used as `http.Server{Addr}`
	ControllerAddr     string `yaml:"controllerAddr"`
	ControllerCertFile string `yaml:"controllerCertFile"`
	ControllerKeyFile  string `yaml:"controllerKeyFile"`
	// ControllerPin accepts value from environment
	// provides local access for debug
	ControllerPin string `yaml:"-"`
	// Custom HTTP configuration
	ControllerReadTimeout  time.Duration `yaml:"-"`
	ControllerWriteTimeout time.Duration `yaml:"-"`
	ControllerStartTimeout time.Duration `yaml:"-"`
	ControllerStopTimeout  time.Duration `yaml:"-"`

	Enabled            bool   `yaml:"enabled"`
	InstallationMode   string `yaml:"installationMode,omitempty"`
	IsDynamicInventory bool   `yaml:"-"`
	// GWEncode defines using HTTPEncode in Groundwork client: child|force|off
	// enabled for child by default
	GWEncode string `yaml:"-"`

	// LogCondense accepts time duration for condensing similar records
	// if 0 turn off condensing
	LogCondense time.Duration `yaml:"logCondense"`
	// LogFile accepts file path to log in addition to stdout
	LogFile        string `yaml:"logFile"`
	LogFileMaxSize int64  `yaml:"logFileMaxSize"`
	// Log files are rotated count times before being removed.
	// If count is 0, old versions are removed rather than rotated.
	LogFileRotate int      `yaml:"logFileRotate"`
	LogLevel      LogLevel `yaml:"logLevel"`
	LogNoColor    bool     `yaml:"logNoColor"`
	LogTimeFormat string   `yaml:"logTimeFormat"`

	// NatsAckWait is the time the NATS server will wait before resending a message
	// Should be greater then the GWClient request duration
	NatsAckWait time.Duration `yaml:"-"`
	// designates the maximum number of outstanding acknowledgements
	// (messages that have been delivered but not acknowledged)
	// that NATS Streaming will allow for a given subscription.
	// When this limit is reached, NATS Streaming will suspend delivery of messages
	// to this subscription until the number of unacknowledged messages falls below the specified limit
	NatsMaxInflight int `yaml:"-"`
	// NatsMaxPubAcksInflight accepts number of unacknowledged messages
	// that a publisher may have in-flight at any given time.
	// When this maximum is reached, further async publish calls will block
	// until the number of unacknowledged messages falls below the specified limit
	NatsMaxPubAcksInflight int   `yaml:"-"`
	NatsMaxPayload         int32 `yaml:"-"`
	// NatsMaxPendingBytes. Deprecated. Use NatsMaxInflight instead.
	// sets the limits for pending msgs and bytes for the internal low-level NATS Subscription.
	// Zero is not allowed. Any negative value means that the given metric is not limited.
	NatsMaxPendingBytes int `yaml:"-"`
	// NatsMaxPendingMsgs. Deprecated. Use NatsMaxInflight instead.
	// sets the limits for pending msgs and bytes for the internal low-level NATS Subscription.
	// Zero is not allowed. Any negative value means that the given metric is not limited.
	NatsMaxPendingMsgs int `yaml:"-"`
	// NatsMonitorPort enables monitoring on http port useful for debug
	// curl 'localhost:8222/streaming/channelsz?limit=0&offset=0&subs=1'
	// More info: https://docs.nats.io/nats-streaming-concepts/monitoring
	NatsMonitorPort int    `yaml:"-"`
	NatsStoreDir    string `yaml:"natsFilestoreDir"`
	// NatsStoreType accepts "FILE"|"MEMORY"
	NatsStoreType string `yaml:"natsStoreType"`
	// How long messages are kept
	NatsStoreMaxAge time.Duration `yaml:"natsStoreMaxAge"`
	// How many bytes are allowed per-channel
	NatsStoreMaxBytes int64 `yaml:"natsStoreMaxBytes"`
	// How many messages are allowed per-channel
	NatsStoreMaxMsgs int `yaml:"natsStoreMaxMsgs"`
	// NatsStoreBufferSize for FileStore type
	// size (in bytes) of the buffer used during file store operations
	NatsStoreBufferSize int `yaml:"-"`
	// NatsStoreReadBufferSize for FileStore type
	// size of the buffer to preload messages
	NatsStoreReadBufferSize int `yaml:"-"`
}

// ConnectorDTO defines TCG Connector configuration
type ConnectorDTO struct {
	AgentID       string        `json:"agentId"`
	AppName       string        `json:"appName"`
	AppType       string        `json:"appType"`
	TcgURL        string        `json:"tcgUrl"`
	LogLevel      LogLevel      `json:"logLevel"`
	Enabled       bool          `json:"enabled"`
	DSConnection  DSConnection  `json:"dalekservicesConnection"`
	GWConnections GWConnections `json:"groundworkConnections"`
	// TODO: extend LoadConnectorDTO to handle more fields
	// MonitorConnection MonitorConnectionDto
	// MetricsProfile    MetricsProfileDto
}

// DSConnection defines DalekServices Connection configuration
type DSConnection clients.DSConnection

// GWConnection defines Groundwork Connection configuration
type GWConnection clients.GWConnection

// MarshalYAML implements yaml.Marshaler interface
// overrides the password field
func (con GWConnection) MarshalYAML() (interface{}, error) {
	type plain GWConnection
	c := plain(con)
	if s := os.Getenv(SecKeyEnv); s != "" {
		encrypted, err := Encrypt([]byte(c.Password), []byte(s))
		if err != nil {
			return nil, err
		}
		c.Password = fmt.Sprintf("%s%x", SecVerPrefix, encrypted)
	}
	return c, nil
}

// UnmarshalYAML implements the yaml.Unmarshaler interface.
// overrides the password field
func (con *GWConnection) UnmarshalYAML(unmarshal func(interface{}) error) error {
	type plain GWConnection
	if err := unmarshal((*plain)(con)); err != nil {
		return err
	}
	if strings.HasPrefix(con.Password, SecVerPrefix) {
		s := os.Getenv(SecKeyEnv)
		if s == "" {
			return fmt.Errorf("unmarshaler error: %s SecKeyEnv is empty", SecVerPrefix)
		}
		var encrypted []byte
		fmt.Sscanf(con.Password, SecVerPrefix+"%x", &encrypted)
		decrypted, err := Decrypt(encrypted, []byte(s))
		if err != nil {
			return err
		}
		con.Password = string(decrypted)
	}
	return nil
}

// GWConnections defines a set of configurations
type GWConnections []*GWConnection

// UnmarshalYAML implements the yaml.Unmarshaler interface.
// Applies decode to items in collection for setting only fields present in yaml.
// Note (as for gopkg.in/yaml.v3 v3.0.0-20210107192922-496545a6307b):
//
//	if yaml defines an empty set this method is not called but target is truncated.
func (cc *GWConnections) UnmarshalYAML(value *yaml.Node) error {
	for i, node := range value.Content {
		if len(*cc) < i+1 {
			*cc = append(*cc, GWConnections{{}}...)
		}
		if err := node.Decode((*cc)[i]); err != nil {
			return err
		}
	}
	return nil
}

// Config defines TCG Agent configuration
type Config struct {
	Connector     Connector     `yaml:"connector"`
	DSConnection  DSConnection  `yaml:"dsConnection"`
	GWConnections GWConnections `yaml:"gwConnections"`
	Jaegertracing Jaegertracing `yaml:"jaegertracing"`
}

func defaults() Config {
	return Config{
		Connector: Connector{
			BatchEvents:             0,
			BatchMetrics:            0,
			BatchMaxBytes:           1024 * 1024, // 1MB
			ControllerAddr:          ":8099",
			ControllerReadTimeout:   time.Second * 10,
			ControllerWriteTimeout:  time.Second * 20,
			ControllerStartTimeout:  time.Second * 4,
			ControllerStopTimeout:   time.Second * 4,
			LogCondense:             0,
			LogFileMaxSize:          1024 * 1024 * 10, // 10MB
			LogFileRotate:           5,
			LogLevel:                1,
			LogNoColor:              false,
			LogTimeFormat:           time.RFC3339,
			NatsAckWait:             time.Second * 30,
			NatsMaxInflight:         1024,
			NatsMaxPubAcksInflight:  1024,
			NatsMaxPayload:          1024 * 1024 * 64, // 64MB github.com/nats-io/nats-server/releases/tag/v2.3.4
			NatsMaxPendingBytes:     -1,
			NatsMaxPendingMsgs:      1024,
			NatsMonitorPort:         0,
			NatsStoreDir:            "natsstore",
			NatsStoreType:           "FILE",
			NatsStoreMaxAge:         time.Hour * 24 * 10,     // 10days
			NatsStoreMaxBytes:       1024 * 1024 * 1024 * 50, // 50GB
			NatsStoreMaxMsgs:        1000000,                 // 1 000 000
			NatsStoreBufferSize:     1024 * 1024 * 2,         // 2MB
			NatsStoreReadBufferSize: 1024 * 1024 * 2,         // 2MB
		},
		// create disabled connections to support partial setting with struct-path
		// 4 items should be enough
		GWConnections: GWConnections{{}, {}, {}, {}},
		DSConnection:  DSConnection{},
		Jaegertracing: Jaegertracing{},
	}
}

// GetConfig implements Singleton pattern
func GetConfig() *Config {
	once.Do(func() {
		/* buffer the logging while configuring */
		logBuf := &logzer.LogBuffer{
			Level: zerolog.TraceLevel,
			Size:  16,
		}
		log.Logger = zerolog.New(logBuf).
			With().Timestamp().Caller().Logger()
		log.Info().Msgf("Build info: %s / %s", buildTag, buildTime)

		/* merge defaults, file, and env */
		applyFlags()
		cfg = new(Config)
		*cfg = defaults()
		if data, err := os.ReadFile(cfg.ConfigPath()); err != nil {
			log.Warn().Err(err).
				Str("configPath", cfg.ConfigPath()).
				Msg("could not read config")
		} else {
			if err := yaml.Unmarshal(data, cfg); err != nil {
				log.Err(err).
					Str("configData", string(data)).
					Str("configPath", cfg.ConfigPath()).
					Msg("could not parse config")
			}

			if data, err := yaml.Marshal(cfg); err == nil {
				data = applyEnv(data)
				if err := yaml.Unmarshal(data, cfg); err != nil {
					log.Err(err).
						Str("configData", string(data)).
						Msg("could not apply env vars")
				}
			} else {
				log.Warn().Err(err).
					Msg("could not apply env vars")
			}
		}

		/* init logger and flush buffer */
		cfg.initLogger()
		logzer.WriteLogBuffer(logBuf)
	})
	return cfg
}

// ConfigPath returns config file path
func (cfg Config) ConfigPath() string {
	configPath := os.Getenv(ConfigEnv)
	if configPath == "" {
		configPath = ConfigName
		if wd, err := os.Getwd(); err == nil {
			configPath = path.Join(wd, ConfigName)
		}
	}
	return configPath
}

func (cfg *Config) loadConnector(data []byte) (*ConnectorDTO, error) {
	var dto ConnectorDTO
	if err := json.Unmarshal(data, &dto); err != nil {
		log.Err(err).Msg("could not parse connector")
		return nil, err
	}
	cfg.Connector.AgentID = dto.AgentID
	cfg.Connector.AppName = dto.AppName
	cfg.Connector.AppType = dto.AppType
	cfg.Connector.LogLevel = dto.LogLevel
	cfg.Connector.Enabled = dto.Enabled
	cfg.GWConnections = dto.GWConnections
	if len(dto.DSConnection.HostName) != 0 {
		cfg.DSConnection.HostName = dto.DSConnection.HostName
	}
	return &dto, nil
}

func (cfg *Config) loadAdvancedPrefixes(data []byte) error {
	var s struct {
		Advanced struct {
			Prefixes []struct {
				GWConnectionID int    `json:"groundworkConnectionId"`
				Prefix         string `json:"prefix"`
			} `json:"prefixes,omitempty"`
		} `json:"advanced,omitempty"`
	}
	if err := json.Unmarshal(data, &s); err != nil {
		log.Err(err).Msg("could not parse advanced")
		return err
	}
	for _, c := range cfg.GWConnections {
		c.PrefixResourceNames = false
		c.ResourceNamePrefix = ""
		for _, p := range s.Advanced.Prefixes {
			if c.ID == p.GWConnectionID && p.Prefix != "" {
				c.PrefixResourceNames = true
				c.ResourceNamePrefix = p.Prefix
			}
		}
	}
	return nil
}

func (cfg *Config) loadDynamicInventoryFlag(data []byte) error {
	/* TODO: Support dynamic flag in UI
	var s struct {
		Connection struct {
			Extensions struct {
				IsDynamicInventory bool `json:"isDynamicInventory"`
			} `json:"extensions,omitempty"`
		} `json:"monitorConnection,omitempty"`
	}

	if err := json.Unmarshal(data, &s); err != nil {
		log.Error("|config.go| : [loadDynamicInventoryFlag] : ", err.Error())
		return err
	} */

	switch cfg.Connector.AppType {
	case "CHECKER", "APM":
		cfg.Connector.IsDynamicInventory = true
	default:
		cfg.Connector.IsDynamicInventory = false
	}

	return nil
}

// LoadConnectorDTO loads ConnectorDTO into Config
func (cfg *Config) LoadConnectorDTO(data []byte) (*ConnectorDTO, error) {
	newCfg := new(Config)
	*newCfg = defaults()
	/* load config file */
	if data, err := os.ReadFile(newCfg.ConfigPath()); err != nil {
		log.Warn().Err(err).
			Str("configPath", newCfg.ConfigPath()).
			Msg("could not read config")
	} else {
		if err := yaml.Unmarshal(data, newCfg); err != nil {
			log.Warn().Err(err).
				Str("configData", string(data)).
				Str("configPath", newCfg.ConfigPath()).
				Msg("could not parse config")
		}
	}
	/* load as ConnectorDTO */
	dto, err := newCfg.loadConnector(data)
	if err != nil {
		return nil, err
	}
	/* load as struct with advanced prefixes field */
	if err := newCfg.loadAdvancedPrefixes(data); err != nil {
		return nil, err
	}
	/* load as struct with dynamic inventory flag */
	if err := newCfg.loadDynamicInventoryFlag(data); err != nil {
		return nil, err
	}

	if output, err := yaml.Marshal(newCfg); err != nil {
		log.Err(err).
			Str("configData", string(output)).
			Msg("could not prepare config for writing")
	} else {
		/* override config file */
		if err := os.WriteFile(newCfg.ConfigPath(), output, 0644); err != nil {
			log.Err(err).
				Str("configData", string(output)).
				Str("configPath", newCfg.ConfigPath()).
				Msg("could not write config")
		}
		/* load environment */
		output = applyEnv(output)
		if err := yaml.Unmarshal(output, newCfg); err != nil {
			log.Err(err).
				Str("configData", string(output)).
				Msg("could not apply env vars")
		}
	}

	/* process PMC */
	if cfg.IsConfiguringPMC() {
		newCfg.Connector.InstallationMode = InstallationModePMC
	}
	/* prepare gwConnections */
	gwEncode := strings.ToLower(newCfg.Connector.GWEncode)
	for i := range newCfg.GWConnections {
		newCfg.GWConnections[i].IsDynamicInventory = newCfg.Connector.IsDynamicInventory
		newCfg.GWConnections[i].HTTPEncode = gwEncode == "force" ||
			(gwEncode != "off" && newCfg.GWConnections[i].IsChild)
	}
	/* update config */
	cfg.Connector = newCfg.Connector
	cfg.DSConnection = newCfg.DSConnection
	cfg.Jaegertracing = newCfg.Jaegertracing
	cfg.GWConnections = newCfg.GWConnections

	/* update logger */
	cfg.initLogger()

	return dto, nil
}

// IsConfiguringPMC checks configuration stage
func (cfg *Config) IsConfiguringPMC() bool {
	return os.Getenv(InstallationModeEnv) == InstallationModePMC &&
		cfg.Connector.InstallationMode != InstallationModePMC
}

// InitTracerProvider inits provider
func (cfg Config) InitTracerProvider() (*tracesdk.TracerProvider, error) {
	return initJaegertracing(cfg.Jaegertracing, fmt.Sprintf("%s:%s:%s",
		cfg.Connector.AppType, cfg.Connector.AppName, cfg.Connector.AgentID))
}

func (cfg Config) initLogger() {
	opts := []logzer.Option{
		logzer.WithCondense(cfg.Connector.LogCondense),
		logzer.WithLastErrors(10),
		logzer.WithLevel([...]zerolog.Level{3, 2, 1, 0}[cfg.Connector.LogLevel]),
		logzer.WithNoColor(cfg.Connector.LogNoColor),
		logzer.WithTimeFormat(cfg.Connector.LogTimeFormat),
	}
	if cfg.Connector.LogFile != "" {
		opts = append(opts, logzer.WithLogFile(&logzer.LogFile{
			FilePath: cfg.Connector.LogFile,
			MaxSize:  cfg.Connector.LogFileMaxSize,
			Rotate:   cfg.Connector.LogFileRotate,
		}))
	}

	/* prevent writes in global logger */
	log.Logger = zerolog.Nop()
	/* reset to defaults */
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnixMs
	/* apply options */
	w := logzer.NewLoggerWriter(opts...)
	/* set global logger */
	log.Logger = zerolog.New(w).
		With().Timestamp().Caller().
		Logger()
	/* set wrapped logger */
	liblogger = zerolog.New(w).
		With().Timestamp().CallerWithSkipFrameCount(4).
		Logger()

	logper.SetLogger(
		func(fields interface{}, format string, a ...interface{}) {
			log2zerolog(zerolog.ErrorLevel, fields, format, a...)
		},
		func(fields interface{}, format string, a ...interface{}) {
			log2zerolog(zerolog.WarnLevel, fields, format, a...)
		},
		func(fields interface{}, format string, a ...interface{}) {
			log2zerolog(zerolog.InfoLevel, fields, format, a...)
		},
		func(fields interface{}, format string, a ...interface{}) {
			if zerolog.GlobalLevel() <= zerolog.DebugLevel {
				log2zerolog(zerolog.DebugLevel, fields, format, a...)
			}
		},
		func() bool { return zerolog.GlobalLevel() <= zerolog.DebugLevel },
	)
}

func log2zerolog(lvl zerolog.Level, fields interface{}, format string, a ...interface{}) {
	e := liblogger.WithLevel(lvl)
	if ff, ok := fields.(interface {
		LogFields() (map[string]interface{}, map[string][]byte)
	}); ok {
		m1, m2 := ff.LogFields()
		e.Fields(m1)
		for k, v := range m2 {
			e.RawJSON(k, v)
		}
	} else if _, ok := fields.(map[string]interface{}); ok {
		e.Fields(fields)
	} else if _, ok := fields.([]interface{}); ok {
		e.Fields(fields)
	}
	e.Msgf(format, a...)
}
