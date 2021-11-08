package config

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"os"
	"path"
	"strings"
	"sync"
	"time"

	"github.com/gwos/tcg/clients"
	"github.com/gwos/tcg/logger"
	"github.com/gwos/tcg/transit"
	"github.com/kelseyhightower/envconfig"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/jaeger"
	"go.opentelemetry.io/otel/sdk/resource"
	tracesdk "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.7.0"
	"golang.org/x/crypto/nacl/secretbox"
	"gopkg.in/yaml.v3"
)

// Variables to control the build info
// can be overridden by Go linker during the build step:
// go build -ldflags "-X 'github.com/gwos/tcg/config.buildTag=<TAG>' -X 'github.com/gwos/tcg/config.buildTime=`date --rfc-3339=s`'"
var (
	buildTag  = "8.x.x"
	buildTime = "Build time not provided"

	once sync.Once
	cfg  *Config
)

// BuildInfo describes the build properties
type BuildInfo struct {
	Tag  string `json:"tag"`
	Time string `json:"time"`
}

// GetBuildInfo returns the build properties
func GetBuildInfo() BuildInfo {
	return BuildInfo{buildTag, buildTime}
}

// ConfigEnv defines environment variable for config file path, overrides the ConfigName
// ConfigName defines default filename for look in work directory if ConfigEnv is empty
// EnvConfigPrefix defines name prefix for environment variables
//   for example: TCG_CONNECTOR_NATSSTORETYPE
const (
	ConfigEnv           = "TCG_CONFIG"
	ConfigName          = "tcg_config.yaml"
	EnvConfigPrefix     = "TCG"
	SecKeyEnv           = "TCG_SECKEY"
	SecVerPrefix        = "_v1_"
	InstallationModeEnv = "INSTALLATION_MODE"
	InstallationModeCMC = "CHILD_MANAGED_CHILD"
	InstallationModePMC = "PARENT_MANAGED_CHILD"
	InstallationModeP   = "PARENT"
	InstallationModeS   = "STANDALONE"
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
	// NatsMonitorPort enables monitoring on http port usefull for debug
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

// Decode implements envconfig.Decoder interface
// merges incoming value with existed structure
func (con *GWConnection) Decode(value string) error {
	var overrides GWConnection
	if err := yaml.Unmarshal([]byte(value), &overrides); err != nil {
		return err
	}
	if overrides.HostName != "" {
		con.HostName = overrides.HostName
	}
	if overrides.UserName != "" {
		con.UserName = overrides.UserName
	}
	if overrides.Password != "" {
		con.Password = overrides.Password
	}
	if overrides.DisplayName != "" {
		con.DisplayName = overrides.DisplayName
	}
	if overrides.DeferOwnership != "" {
		con.DeferOwnership = overrides.DeferOwnership
	}
	if overrides.ResourceNamePrefix != "" {
		con.ResourceNamePrefix = overrides.ResourceNamePrefix
	}
	return nil
}

// GWConnections defines a set of configurations
type GWConnections []*GWConnection

// Decode implements envconfig.Decoder interface
// merges incoming value with existing structure
func (cons *GWConnections) Decode(value string) error {
	var overrides GWConnections
	if err := yaml.Unmarshal([]byte(value), &overrides); err != nil {
		return err
	}
	if len(overrides) > len(*cons) {
		buf := GWConnections(make([]*GWConnection, len(overrides)))
		copy(buf, overrides)
		copy(buf, *cons)
		*cons = buf
	}
	for i, v := range overrides {
		if v.HostName != "" {
			(*cons)[i].HostName = v.HostName
		}
		if v.UserName != "" {
			(*cons)[i].UserName = v.UserName
		}
		if v.Password != "" {
			(*cons)[i].Password = v.Password
		}
	}
	return nil
}

// DSConnection defines DalekServices Connection configuration
type DSConnection clients.DSConnection

// Decode implements envconfig.Decoder interface
// merges incoming value with existed structure
func (con *DSConnection) Decode(value string) error {
	var overrides DSConnection
	if err := yaml.Unmarshal([]byte(value), &overrides); err != nil {
		return err
	}
	if overrides.HostName != "" {
		con.HostName = overrides.HostName
	}
	return nil
}

// Jaegertracing defines the configuration of telemetry provider
type Jaegertracing struct {
	// Agent defines address for communicating with AgentJaegerThriftCompactUDP,
	// hostport, like jaeger-agent:6831
	Agent string `yaml:"agent"`
	// Collector defines traces endpoint,
	// in case the client should connect directly to the CollectorHTTP,
	// endpoint, like http://jaeger-collector:14268/api/traces
	Collector string `yaml:"collector"`
	// Tags defines tracer-level tags, which get added to all reported spans
	Tags map[string]string `yaml:"tags"`
}

// Config defines TCG Agent configuration
type Config struct {
	Connector     *Connector     `yaml:"connector"`
	DSConnection  *DSConnection  `yaml:"dsConnection"`
	GWConnections GWConnections  `yaml:"gwConnections"`
	Jaegertracing *Jaegertracing `yaml:"jaegertracing"`
}

func defaults() Config {
	return Config{
		Connector: &Connector{
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
			NatsMaxPayload:          1024 * 1024 * 80, // 80MB
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
		DSConnection:  &DSConnection{},
		Jaegertracing: &Jaegertracing{},
	}
}

// GetConfig implements Singleton pattern
func GetConfig() *Config {
	once.Do(func() {
		/* buffer the logging while configuring */
		logBuf := &logger.LogBuffer{
			Level: zerolog.TraceLevel,
			Size:  16,
		}
		log.Logger = zerolog.New(logBuf).
			With().Timestamp().Caller().Logger()
		log.Info().Msgf("Build info: %s / %s", buildTag, buildTime)

		c := defaults()
		cfg = &c
		if data, err := ioutil.ReadFile(cfg.ConfigPath()); err != nil {
			log.Warn().Err(err).
				Str("configPath", cfg.ConfigPath()).
				Msg("could not read config")
		} else {
			if err := yaml.Unmarshal(data, cfg); err != nil {
				log.Err(err).
					Str("configPath", cfg.ConfigPath()).
					Msg("could not parse config")
			}
		}
		if err := envconfig.Process(EnvConfigPrefix, cfg); err != nil {
			log.Err(err).
				Str("EnvConfigPrefix", EnvConfigPrefix).
				Msg("could not process config environment")
		}
		cfg.initLogger()
		logger.WriteLogBuffer(logBuf)
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
	c := defaults()
	newCfg := &c
	/* load config file */
	if data, err := ioutil.ReadFile(newCfg.ConfigPath()); err != nil {
		log.Warn().Err(err).
			Str("configPath", cfg.ConfigPath()).
			Msg("could not read config")
	} else {
		if err := yaml.Unmarshal(data, newCfg); err != nil {
			log.Warn().Err(err).
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
	/* override config file */
	if output, err := yaml.Marshal(newCfg); err != nil {
		log.Err(err).
			Msg("could not prepare config for writing")
	} else {
		if err := ioutil.WriteFile(newCfg.ConfigPath(), output, 0644); err != nil {
			log.Err(err).
				Str("configPath", newCfg.ConfigPath()).
				Msg("could not write config")
		}
	}
	/* load environment */
	if err := envconfig.Process(EnvConfigPrefix, newCfg); err != nil {
		log.Err(err).
			Str("EnvConfigPrefix", EnvConfigPrefix).
			Msg("could not process config environment")
	}
	/* process PMC */
	if cfg.IsConfiguringPMC() {
		newCfg.Connector.InstallationMode = InstallationModePMC
	}
	/* apply dynamic inventory flag to gwConnections */
	for i := range newCfg.GWConnections {
		newCfg.GWConnections[i].IsDynamicInventory = newCfg.Connector.IsDynamicInventory
	}
	/* update config */
	*cfg.Connector = *newCfg.Connector
	*cfg.DSConnection = *newCfg.DSConnection
	*cfg.Jaegertracing = *newCfg.Jaegertracing
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
	return cfg.initJaegertracing()
}

// initJaegertracing inits tracing provider with Jaeger exporter
func (cfg Config) initJaegertracing() (*tracesdk.TracerProvider, error) {
	var errNotConfigured = fmt.Errorf("telemetry is not configured")
	/* Jaegertracing supports a few options to receive spans
	[https://github.com/jaegertracing/jaeger/blob/master/ports/ports.go]
	// AgentJaegerThriftCompactUDP is the default port for receiving Jaeger Thrift over UDP in compact encoding
	AgentJaegerThriftCompactUDP = 6831
	// AgentJaegerThriftBinaryUDP is the default port for receiving Jaeger Thrift over UDP in binary encoding
	AgentJaegerThriftBinaryUDP = 6832
	// AgentZipkinThriftCompactUDP is the default port for receiving Zipkin Thrift over UDP in binary encoding
	AgentZipkinThriftCompactUDP = 5775
	// CollectorGRPC is the default port for gRPC server for sending spans
	CollectorGRPC = 14250
	// CollectorHTTP is the default port for HTTP server for sending spans (e.g. /api/traces endpoint)
	CollectorHTTP = 14268

	The otel jaeger exporter supports AgentJaegerThriftCompactUDP and CollectorHTTP protocols.
	otel-v0.20.0 Note the possible mistakes in defaults:
		* "6832" for jaeger.WithAgentPort()
		* "http://localhost:14250" for jaeger.WithCollectorEndpoint()

	Checking configuration to prevent exporter run with internal defaults in environment without receiver.
	The OTEL_EXPORTER_ env vars take precedence on the TCG config (with TCG_JAEGERTRACING_ env vars).
	And the Agent entrypoint setting takes precedence on the Collector entrypoint. */
	otelExporterJaegerAgentHost := os.Getenv("OTEL_EXPORTER_JAEGER_AGENT_HOST")
	otelExporterJaegerAgentPort := os.Getenv("OTEL_EXPORTER_JAEGER_AGENT_PORT")
	otelExporterJaegerEndpoint := os.Getenv("OTEL_EXPORTER_JAEGER_ENDPOINT")
	otelExporterJaegerPassword := os.Getenv("OTEL_EXPORTER_JAEGER_PASSWORD")
	otelExporterJaegerUser := os.Getenv("OTEL_EXPORTER_JAEGER_USER")
	tcgJaegerAgent := cfg.Jaegertracing.Agent
	tcgJaegerCollector := cfg.Jaegertracing.Collector

	var endpointOption jaeger.EndpointOption
	switch {
	case len(otelExporterJaegerAgentHost)+len(otelExporterJaegerAgentPort) != 0:
		endpointOption = jaeger.WithAgentEndpoint()
	case len(otelExporterJaegerEndpoint)+len(otelExporterJaegerPassword)+len(otelExporterJaegerUser) != 0:
		endpointOption = jaeger.WithCollectorEndpoint()
	case len(tcgJaegerAgent) != 0:
		if host, port, err := net.SplitHostPort(tcgJaegerAgent); err == nil {
			endpointOption = jaeger.WithAgentEndpoint(
				jaeger.WithAgentHost(host),
				jaeger.WithAgentPort(port),
			)
		} else {
			log.Err(err).Msg("could not parse the JaegerAgent")
			return nil, err
		}
	case len(tcgJaegerCollector) != 0:
		endpointOption = jaeger.WithCollectorEndpoint(jaeger.WithEndpoint(tcgJaegerCollector))
	default:
		log.Debug().Msg(errNotConfigured.Error())
		return nil, errNotConfigured
	}

	attrs := []attribute.KeyValue{
		semconv.ServiceNameKey.String(fmt.Sprintf("%s:%s:%s",
			cfg.Connector.AppType, cfg.Connector.AppName, cfg.Connector.AgentID)),
		attribute.String("runtime", "golang"),
	}
	for k, v := range cfg.Jaegertracing.Tags {
		attrs = append(attrs, attribute.String(k, v))
	}

	/* It may be useful to look for modern API state and usage at:
	https://github.com/open-telemetry/opentelemetry-go/blob/main/example/jaeger/main.go */

	exporter, err := jaeger.New(endpointOption)
	if err != nil {
		log.Err(err).Msg("could not create exporter")
		return nil, err
	}
	tp := tracesdk.NewTracerProvider(
		/* Always be sure to batch in production */
		tracesdk.WithBatcher(exporter),
		/* Record information about this application in an Resource */
		tracesdk.WithResource(resource.NewWithAttributes(semconv.SchemaURL, attrs...)),
	)
	return tp, nil
}

func (cfg Config) initLogger() {
	opts := []logger.Option{
		logger.WithCondense(cfg.Connector.LogCondense),
		logger.WithLastErrors(10),
		logger.WithLevel([...]zerolog.Level{3, 2, 1, 0}[cfg.Connector.LogLevel]),
		logger.WithNoColor(cfg.Connector.LogNoColor),
		logger.WithTimeFormat(cfg.Connector.LogTimeFormat),
	}
	if cfg.Connector.LogFile != "" {
		opts = append(opts, logger.WithLogFile(&logger.LogFile{
			FilePath: cfg.Connector.LogFile,
			MaxSize:  cfg.Connector.LogFileMaxSize,
			Rotate:   cfg.Connector.LogFileRotate,
		}))
	}
	logger.SetLogger(opts...)

	clients.SetLogger(
		func(fields interface{}, format string, v ...interface{}) {
			log2zerolog(zerolog.ErrorLevel, fields, format, v...)
		},
		func(fields interface{}, format string, v ...interface{}) {
			log2zerolog(zerolog.WarnLevel, fields, format, v...)
		},
		func(fields interface{}, format string, v ...interface{}) {
			log2zerolog(zerolog.InfoLevel, fields, format, v...)
		},
		func(fields interface{}, format string, v ...interface{}) {
			if zerolog.GlobalLevel() <= zerolog.DebugLevel {
				log2zerolog(zerolog.DebugLevel, fields, format, v...)
			}
		},
		func() bool { return zerolog.GlobalLevel() <= zerolog.DebugLevel },
	)
}

func log2zerolog(lvl zerolog.Level, fields interface{}, format string, v ...interface{}) {
	logger := log.Logger.With().CallerWithSkipFrameCount(4).Logger()
	e := logger.WithLevel(lvl)
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
	e.Msgf(format, v...)
}

// Decrypt decrypts small messages
// golang.org/x/crypto/nacl/secretbox
func Decrypt(message, secret []byte) ([]byte, error) {
	var nonce [24]byte
	var secretKey [32]byte = sha256.Sum256(secret)
	copy(nonce[:], message[:24])
	decrypted, ok := secretbox.Open(nil, message[24:], &nonce, &secretKey)
	if !ok {
		return nil, fmt.Errorf("decryption error")
	}
	return decrypted, nil
}

// Encrypt encrypts small messages
// golang.org/x/crypto/nacl/secretbox
func Encrypt(message, secret []byte) ([]byte, error) {
	var nonce [24]byte
	var secretKey [32]byte = sha256.Sum256(secret)
	if _, err := io.ReadFull(rand.Reader, nonce[:]); err != nil {
		return nil, err
	}
	return secretbox.Seal(nonce[:], message, &nonce, &secretKey), nil
}
