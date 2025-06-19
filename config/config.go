package config

import (
	"encoding/json"
	"fmt"
	stdlog "log"
	"log/slog"
	"os"
	"path"
	"strings"
	"sync"
	"time"

	"github.com/gwos/tcg/logzer"
	"github.com/gwos/tcg/nats"
	"github.com/gwos/tcg/sdk/clients"
	sdklog "github.com/gwos/tcg/sdk/log"
	"github.com/gwos/tcg/sdk/transit"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	tracesdk "go.opentelemetry.io/otel/sdk/trace"
	"gopkg.in/yaml.v3"
)

var (
	once sync.Once
	cfg  *Config

	// Suppress provides ability to globally suppress the submission of all data.
	// Not for any production use-case, but strictly troubleshooting:
	// this would be useful in troubleshooting to isolate synch issues to malformed perf data
	// (which is a really common problem)
	Suppress struct {
		Downtimes bool `env:"SUPPRESS_DOWNTIMES"`
		Events    bool `env:"SUPPRESS_EVENTS"`
		Inventory bool `env:"SUPPRESS_INVENTORY"`
		Metrics   bool `env:"SUPPRESS_METRICS"`
	}
)

const (
	InstallationModeEnv = "INSTALLATION_MODE"
	InstallationModeCMC = "CHILD_MANAGED_CHILD"
	InstallationModePMC = "PARENT_MANAGED_CHILD"
	InstallationModeP   = "PARENT"
	InstallationModeS   = "STANDALONE"

	ParentInstanceNameEnv = "PARENT_INSTANCE_NAME"

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
	Trace
)

func (l LogLevel) String() string {
	return [...]string{"Error", "Warn", "Info", "Debug", "Trace"}[l]
}

type Nats struct {
	// NatsAckWait is the time the NATS server will wait before resending a message
	// Should be greater then the GWClient request duration
	NatsAckWait time.Duration `env:"NATSACKWAIT" yaml:"-"`
	// designates the maximum number of outstanding acknowledgements
	// (messages that have been delivered but not acknowledged)
	// that NATS Streaming will allow for a given subscription.
	// When this limit is reached, NATS Streaming will suspend delivery of messages
	// to this subscription until the number of unacknowledged messages falls below the specified limit
	NatsMaxInflight int `env:"NATSMAXINFLIGHT" yaml:"-"`
	// NatsMaxPubAcksInflight accepts number of unacknowledged messages
	// that a publisher may have in-flight at any given time.
	// When this maximum is reached, further async publish calls will block
	// until the number of unacknowledged messages falls below the specified limit
	NatsMaxPubAcksInflight int   `env:"NATSMAXPUBACKSINFLIGHT" yaml:"-"`
	NatsMaxPayload         int32 `env:"NATSMAXPAYLOAD" yaml:"-"`
	// NatsMonitorPort enables monitoring on http port useful for debug
	// curl 'localhost:8222/streaming/channelsz?limit=0&offset=0&subs=1'
	// More info: https://docs.nats.io/nats-streaming-concepts/monitoring
	NatsMonitorPort int    `env:"NATSMONITORPORT" yaml:"-"`
	NatsStoreDir    string `env:"NATSSTOREDIR" yaml:"natsFilestoreDir"`
	// NatsStoreType accepts "FILE"|"MEMORY"
	NatsStoreType string `env:"NATSSTORETYPE" yaml:"natsStoreType"`
	// How long messages are kept
	NatsStoreMaxAge time.Duration `env:"NATSSTOREMAXAGE" yaml:"natsStoreMaxAge"`
	// How many bytes are allowed per-channel
	NatsStoreMaxBytes int64 `env:"NATSSTOREMAXBYTES" yaml:"natsStoreMaxBytes"`
	// How many messages are allowed per-channel
	NatsStoreMaxMsgs int64 `env:"NATSSTOREMAXMSGS" yaml:"natsStoreMaxMsgs"`
	// NatsServerConfigFile is used to override yaml values for
	// NATS server configuration (debug only).
	NatsServerConfigFile string `env:"NATSSERVERCONFIGFILE" yaml:"natsServerConfigFile"`
}

// Hashsum calculates FNV non-cryptographic hash suitable for checking the equality
func (c Nats) Hashsum() ([]byte, error) {
	return Hashsum(c)
}

// Connector defines TCG Connector configuration
// see GetConfig() for defaults
type Connector struct {
	transit.AgentIdentity `yaml:",inline"`

	BatchEvents   time.Duration `env:"BATCHEVENTS" yaml:"batchEvents"`
	BatchMetrics  time.Duration `env:"BATCHMETRICS" yaml:"batchMetrics"`
	BatchMaxBytes int           `env:"BATCHMAXBYTES" yaml:"batchMaxBytes"`

	// ControllerAddr accepts value for combined "host:port"
	// used as `http.Server{Addr}`
	ControllerAddr     string `env:"CONTROLLERADDR" yaml:"controllerAddr"`
	ControllerCertFile string `env:"CONTROLLERCERTFILE" yaml:"controllerCertFile"`
	ControllerKeyFile  string `env:"CONTROLLERKEYFILE" yaml:"controllerKeyFile"`
	// ControllerPin accepts value from environment
	// provides local access for debug
	ControllerPin string `env:"CONTROLLERPIN" yaml:"-"`
	// Custom HTTP configuration
	ControllerReadTimeout  time.Duration `env:"CONTROLLERREADTIMEOUT" yaml:"-"`
	ControllerWriteTimeout time.Duration `env:"CONTROLLERWRITETIMEOUT" yaml:"-"`
	ControllerStartTimeout time.Duration `env:"CONTROLLERSTARTTIMEOUT" yaml:"-"`
	ControllerStopTimeout  time.Duration `env:"CONTROLLERSTOPTIMEOUT" yaml:"-"`

	Enabled            bool   `env:"ENABLED" yaml:"enabled"`
	InstallationMode   string `env:"INSTALLATIONMODE" yaml:"installationMode,omitempty"`
	IsDynamicInventory bool   `env:"ISDYNAMICINVENTORY" yaml:"-"`
	// GWEncode defines using HTTPEncode in Groundwork client: child|force|off
	// enabled for child by default
	GWEncode string `env:"GWENCODE" yaml:"-"`

	// LogCondense accepts time duration for condensing similar records
	// if 0 turn off condensing
	LogCondense time.Duration `env:"LOGCONDENSE" yaml:"logCondense"`
	// LogFile accepts file path to log in addition to stdout
	LogFile        string `env:"LOGFILE" yaml:"logFile"`
	LogFileMaxSize int64  `env:"LOGFILEMAXSIZE" yaml:"logFileMaxSize"`
	// Log files are rotated count times before being removed.
	// If count is 0, old versions are removed rather than rotated.
	LogFileRotate int      `env:"LOGFILEROTATE" yaml:"logFileRotate"`
	LogLevel      LogLevel `env:"LOGLEVEL" yaml:"logLevel"`
	LogColors     bool     `env:"LOGCOLORS" yaml:"logColors"`
	LogTimeFormat string   `env:"LOGTIMEFORMAT" yaml:"logTimeFormat"`

	Nats `yaml:",inline"`

	RetryDelays []time.Duration `env:"RETRYDELAYS" yaml:"-"`

	TransportStartRndDelay int `env:"TRANSPORTSTARTRNDDELAY" yaml:"-"`

	ExportProm bool `env:"EXPORTPROM" yaml:"-"`

	ExportTransitDir string `env:"EXPORTTRANSITDIR" yaml:"-"`
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

// AsClient returns as clients type
func (c *DSConnection) AsClient() clients.DSConnection {
	return (clients.DSConnection)(*c)
}

// GWConnection defines Groundwork Connection configuration
type GWConnection clients.GWConnection

// AsClient returns as clients type
func (c *GWConnection) AsClient() clients.GWConnection {
	return (clients.GWConnection)(*c)
}

// MarshalYAML implements yaml.Marshaler interface
// overrides the password field
func (c GWConnection) MarshalYAML() (any, error) {
	type plain GWConnection
	p := plain(c)
	if s := os.Getenv(SecKeyEnv); s != "" {
		encrypted, err := Encrypt([]byte(p.Password), []byte(s))
		if err != nil {
			return nil, err
		}
		p.Password = fmt.Sprintf("%s%x", SecVerPrefix, encrypted)
	}
	return p, nil
}

// UnmarshalYAML implements the yaml.Unmarshaler interface.
// overrides the password field
func (c *GWConnection) UnmarshalYAML(unmarshal func(any) error) error {
	type plain GWConnection
	if err := unmarshal((*plain)(c)); err != nil {
		return err
	}
	if strings.HasPrefix(c.Password, SecVerPrefix) {
		s := os.Getenv(SecKeyEnv)
		if s == "" {
			return fmt.Errorf("unmarshaler error: %s SecKeyEnv is empty", SecVerPrefix)
		}
		var encrypted []byte
		if _, err := fmt.Sscanf(c.Password, SecVerPrefix+"%x", &encrypted); err != nil {
			return err
		}
		decrypted, err := Decrypt(encrypted, []byte(s))
		if err != nil {
			return err
		}
		c.Password = string(decrypted)
	}
	return nil
}

// GWConnections defines a set of configurations
type GWConnections []GWConnection

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
		if err := node.Decode(&(*cc)[i]); err != nil {
			return err
		}
	}
	return nil
}

// Config defines TCG Agent configuration
type Config struct {
	Connector     Connector     `envPrefix:"CONNECTOR_" yaml:"connector"`
	DSConnection  DSConnection  `envPrefix:"DSCONNECTION_" yaml:"dsConnection"`
	GWConnections GWConnections `envPrefix:"GWCONNECTIONS_" yaml:"gwConnections"`
}

func defaults() Config {
	return Config{
		Connector: Connector{
			BatchEvents:            0,
			BatchMetrics:           0,
			BatchMaxBytes:          1024 * 1024, // 1MB
			ControllerAddr:         ":8099",
			ControllerReadTimeout:  time.Second * 10,
			ControllerWriteTimeout: time.Second * 20,
			ControllerStartTimeout: time.Second * 4,
			ControllerStopTimeout:  time.Second * 4,
			LogCondense:            0,
			LogFileMaxSize:         1024 * 1024 * 10, // 10MB
			LogFileRotate:          5,
			LogLevel:               1,
			LogColors:              false,
			LogTimeFormat:          time.RFC3339,
			Nats: Nats{
				NatsAckWait:            time.Second * 30,
				NatsMaxInflight:        4,
				NatsMaxPubAcksInflight: 4,
				NatsMaxPayload:         1024 * 1024 * 8, // 8MB github.com/nats-io/nats-server/releases/tag/v2.3.4
				NatsMonitorPort:        0,
				NatsStoreDir:           "natsstore",
				NatsStoreType:          "FILE",
				NatsStoreMaxAge:        time.Hour * 24 * 10,     // 10days
				NatsStoreMaxBytes:      1024 * 1024 * 1024 * 20, // 20GB
				NatsStoreMaxMsgs:       1_000_000,               // 1 000 000
				NatsServerConfigFile:   "",
			},
			RetryDelays: []time.Duration{time.Second * 30, time.Second * 30, time.Second * 30, time.Second * 30, time.Second * 30,
				time.Second * 30, time.Second * 30, time.Second * 30, time.Minute * 1, time.Minute * 5, time.Minute * 20},
			TransportStartRndDelay: 60,
		},
		// create disabled connections to support partial setting with struct-path
		// 4 items should be enough
		GWConnections: GWConnections{{}, {}, {}, {}},
		DSConnection:  DSConnection{},
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
		}
		if err := applyEnv(cfg, &Suppress); err != nil {
			log.Warn().Err(err).
				Msg("could not apply env vars")
		}
		logSuppress := func(b bool, str string) {
			if b {
				log.Error().Msgf("TCG will suppress %v due to env var is active", str)
			}
		}
		logSuppress(Suppress.Downtimes, "Downtimes")
		logSuppress(Suppress.Events, "Events")
		logSuppress(Suppress.Inventory, "Inventory")
		logSuppress(Suppress.Metrics, "Metrics")

		/* process PMC */
		if cfg.IsPMC() {
			cfg.Connector.InstallationMode = InstallationModePMC
			cfg.DSConnection.HostName = os.Getenv(ParentInstanceNameEnv)
		}
		/* prepare gwConnections */
		gwEncode := strings.ToLower(cfg.Connector.GWEncode)
		for i := range cfg.GWConnections {
			cfg.GWConnections[i].IsDynamicInventory = cfg.Connector.IsDynamicInventory
			cfg.GWConnections[i].HTTPEncode = gwEncode == "force" ||
				(gwEncode != "off" && cfg.GWConnections[i].IsChild)
		}
		/* init logger and flush buffer */
		cfg.initLogger()
		logzer.WriteLogBuffer(logBuf)
		/* update other deps */
		nats.RetryDelays = cfg.Connector.RetryDelays
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
			} `json:"prefixes,omitzero"`
		} `json:"advanced,omitzero"`
	}
	if err := json.Unmarshal(data, &s); err != nil {
		log.Err(err).Msg("could not parse advanced")
		return err
	}
	for i := range cfg.GWConnections {
		cfg.GWConnections[i].PrefixResourceNames = false
		cfg.GWConnections[i].ResourceNamePrefix = ""
		for _, p := range s.Advanced.Prefixes {
			if cfg.GWConnections[i].ID == p.GWConnectionID && p.Prefix != "" {
				cfg.GWConnections[i].PrefixResourceNames = true
				cfg.GWConnections[i].ResourceNamePrefix = p.Prefix
			}
		}
	}
	return nil
}

func (cfg *Config) loadDynamicInventoryFlag(_ []byte) error {
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
	case "CHECKER", "APM", "EVENTS", "TCGAZURE":
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
		if err := applyEnv(newCfg); err != nil {
			log.Warn().Err(err).
				Msg("could not apply env vars")
		}
	}

	/* process PMC */
	if cfg.IsPMC() {
		newCfg.Connector.InstallationMode = InstallationModePMC
		newCfg.DSConnection.HostName = os.Getenv(ParentInstanceNameEnv)
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
	cfg.GWConnections = newCfg.GWConnections

	/* update logger */
	cfg.initLogger()

	/* update other deps */
	nats.RetryDelays = cfg.Connector.RetryDelays

	return dto, nil
}

// IsPMC checks configuration
func (cfg *Config) IsPMC() bool {
	return os.Getenv(InstallationModeEnv) == InstallationModePMC
}

// InitTracerProvider inits provider
func (cfg Config) InitTracerProvider() (*tracesdk.TracerProvider, error) {
	return initOTLP(fmt.Sprintf("%s:%s:%s",
		cfg.Connector.AppType, cfg.Connector.AppName, cfg.Connector.AgentID))
}

// Hashsum calculates FNV non-cryptographic hash suitable for checking the equality
func (cfg Config) Hashsum() ([]byte, error) {
	return Hashsum(cfg)
}

func (cfg Config) initLogger() {
	if cfg.Connector.LogLevel > 4 {
		cfg.Connector.LogLevel = 4
	}
	lvl := [...]zerolog.Level{3, 2, 1, 0, -1}[cfg.Connector.LogLevel]
	if lvl <= zerolog.DebugLevel {
		cfg.Connector.LogCondense = 0
	}
	opts := []logzer.Option{
		logzer.WithColors(cfg.Connector.LogColors),
		logzer.WithCondense(cfg.Connector.LogCondense),
		logzer.WithLastErrors(10),
		logzer.WithLevel(lvl),
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
	/* adapt SDK logger */
	sdklog.Logger = slog.New(&logzer.SLogHandler{CallerSkipFrame: 3})
	/* set as standard logger output */
	stdlog.SetFlags(0)
	stdlog.SetOutput(log.Logger)
}
