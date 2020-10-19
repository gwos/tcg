package config

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"math"
	"os"
	"path"
	"strings"
	"sync"
	"time"

	"github.com/gwos/tcg/log"
	"github.com/kelseyhightower/envconfig"
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

// ConfigStringConstant defines string constant type
type ConfigStringConstant string

// ConfigEnv defines environment variable for config file path, overrides the ConfigName
// ConfigName defines default filename for look in work directory if ConfigEnv is empty
// EnvConfigPrefix defines name prefix for environment variables
//   for example: TCG_CONNECTOR_NATSSTORETYPE
const (
	ConfigEnv           ConfigStringConstant = "TCG_CONFIG"
	ConfigName                               = "tcg_config.yaml"
	EnvConfigPrefix                          = "TCG"
	SecKeyEnv                                = "TCG_SECKEY"
	SecVerPrefix                             = "_v1_"
	InstallationModeEnv                      = "INSTALLATION_MODE"
	InstallationModeCMC                      = "CHILD_MANAGED_CHILD"
	InstallationModePMC                      = "PARENT_MANAGED_CHILD"
	InstallationModeP                        = "PARENT"
	InstallationModeS                        = "STANDALONE"
)

// LogLevel defines levels for logrus
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
	AgentID string `yaml:"agentId"`
	AppName string `yaml:"appName"`
	AppType string `yaml:"appType"`
	// ControllerAddr accepts value for combined "host:port"
	// used as `http.Server{Addr}`
	ControllerAddr     string `yaml:"controllerAddr"`
	ControllerCertFile string `yaml:"controllerCertFile"`
	ControllerKeyFile  string `yaml:"controllerKeyFile"`
	// ControllerPin accepts value from environment
	// provides local access for debug
	ControllerPin string `yaml:"-"`
	// NatsAckWait accepts number of seconds
	// should be greater then the GWClient request duration
	NatsAckWait int64 `yaml:"natsAckWait"`
	// NatsMaxInflight accepts number of unacknowledged messages
	// that a publisher may have in-flight at any given time.
	// When this maximum is reached, further async publish calls will block
	// until the number of unacknowledged messages falls below the specified limit
	NatsMaxInflight  int    `yaml:"natsMaxInflight"`
	NatsFilestoreDir string `yaml:"natsFilestoreDir"`
	// NatsStoreType accepts "FILE"|"MEMORY"
	NatsStoreType string `yaml:"natsStoreType"`
	// LogConsPeriod accepts number of seconds
	// if 0 turn off consolidation
	LogConsPeriod int `yaml:"logConsPeriod"`
	// LogFile accepts file path to log in addition to stdout
	LogFile            string   `yaml:"logFile"`
	LogLevel           LogLevel `yaml:"logLevel"`
	Enabled            bool     `yaml:"enabled"`
	InstallationMode   string   `yaml:"installationMode,omitempty"`
	IsDynamicInventory bool     `yaml:"-"`
}

// ConnectorDTO defines TCG Connector configuration
type ConnectorDTO struct {
	AgentID       string        `json:"agentId"`
	AppName       string        `json:"appName"`
	AppType       string        `json:"appType"`
	TcgURL        string        `json:"tcgUrl"`
	LogConsPeriod int           `json:"logConsPeriod"`
	LogLevel      LogLevel      `json:"logLevel"`
	Enabled       bool          `json:"enabled"`
	DSConnection  DSConnection  `json:"dalekservicesConnection"`
	GWConnections GWConnections `json:"groundworkConnections"`
	// TODO: extend LoadConnectorDTO to handle more fields
	// MonitorConnection MonitorConnectionDto
	// MetricsProfile    MetricsProfileDto
}

// GWConnection defines Groundwork Connection configuration
type GWConnection struct {
	ID int `yaml:"id"`
	// HostName accepts value for combined "host:port"
	// used as `url.URL{HostName}`
	HostName            string `yaml:"hostName"`
	UserName            string `yaml:"userName"`
	Password            string `yaml:"password"`
	Enabled             bool   `yaml:"enabled"`
	IsChild             bool   `yaml:"isChild"`
	DisplayName         string `yaml:"displayName"`
	MergeHosts          bool   `yaml:"mergeHosts"`
	LocalConnection     bool   `yaml:"localConnection"`
	DeferOwnership      string `yaml:"deferOwnership"`
	PrefixResourceNames bool   `yaml:"prefixResourceNames"`
	ResourceNamePrefix  string `yaml:"resourceNamePrefix"`
	SendAllInventory    bool   `yaml:"sendAllInventory"`
}

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
		*cons = *(&buf)
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
type DSConnection struct {
	// HostName accepts value for combined "host:port"
	// used as `url.URL{HostName}`
	HostName string `yaml:"hostName"`
}

// Decode implements envconfig.Decoder interface
// merges incoming value with existed structure
func (con *DSConnection) Decode(value string) error {
	var overrides GWConnection
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
	// Agent defines address for communicating via UDP,
	// like jaeger-agent:6831
	// Is ignored if the Collector is specified
	Agent string `yaml:"agent"`
	// Collector defines traces endpoint,
	// in case the client should connect directly to the Collector,
	// like http://jaeger-collector:14268/api/traces
	// If specified, the AgentAddress is ignored
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
			ControllerAddr:   ":8099",
			LogConsPeriod:    0,
			LogLevel:         1,
			NatsAckWait:      30,
			NatsMaxInflight:  math.MaxInt32,
			NatsFilestoreDir: "natsstore",
			NatsStoreType:    "FILE",
		},
		DSConnection:  &DSConnection{},
		Jaegertracing: &Jaegertracing{},
	}
}

// GetConfig implements Singleton pattern
func GetConfig() *Config {
	once.Do(func() {
		logBuf := make(map[string]interface{}, 3)
		c := defaults()
		cfg = &c

		if data, err := ioutil.ReadFile(cfg.configPath()); err != nil {
			logBuf["ioutil.ReadFile"] = err
		} else {
			if err := yaml.Unmarshal(data, cfg); err != nil {
				logBuf["yaml.Unmarshal"] = err
			}
		}

		if err := envconfig.Process(EnvConfigPrefix, cfg); err != nil {
			logBuf["envconfig.Process"] = err
		}

		log.Config(
			cfg.Connector.LogFile,
			int(cfg.Connector.LogLevel),
			time.Duration(cfg.Connector.LogConsPeriod)*time.Second,
		)
		log.Info(fmt.Sprintf("Build info: %s / %s", buildTag, buildTime))
		if len(logBuf) > 0 {
			log.With(logBuf).Warn()
		}
	})
	return cfg
}

func (cfg Config) configPath() string {
	configPath := os.Getenv(string(ConfigEnv))
	if configPath == "" {
		wd, err := os.Getwd()
		if err != nil {
			log.Warn(err)
			wd = ""
		}
		configPath = path.Join(wd, ConfigName)
	}
	return configPath
}

func (cfg *Config) loadConnector(data []byte) (*ConnectorDTO, error) {
	var dto ConnectorDTO
	if err := json.Unmarshal(data, &dto); err != nil {
		log.Error("|config.go| : [loadConnector] : ", err.Error())
		return nil, err
	}
	cfg.Connector.AgentID = dto.AgentID
	cfg.Connector.AppName = dto.AppName
	cfg.Connector.AppType = dto.AppType
	cfg.Connector.LogConsPeriod = dto.LogConsPeriod
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
		log.Error("|config.go| : [loadAdvancedPrefixes] : ", err.Error())
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
	if data, err := ioutil.ReadFile(newCfg.configPath()); err != nil {
		log.Warn(err)
	} else {
		if err := yaml.Unmarshal(data, newCfg); err != nil {
			log.Warn(err)
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
		log.Warn(err)
	} else {
		if err := ioutil.WriteFile(newCfg.configPath(), output, 0644); err != nil {
			log.Warn(err)
		}
	}
	/* load environment */
	if err := envconfig.Process(EnvConfigPrefix, newCfg); err != nil {
		log.Warn(err)
	}
	/* process PMC */
	if cfg.IsConfiguringPMC() {
		newCfg.Connector.InstallationMode = InstallationModePMC
	}
	/* update config */
	*cfg.Connector = *newCfg.Connector
	*cfg.DSConnection = *newCfg.DSConnection
	*cfg.Jaegertracing = *newCfg.Jaegertracing
	cfg.GWConnections = newCfg.GWConnections

	/* update logger */
	log.Config(
		cfg.Connector.LogFile,
		int(cfg.Connector.LogLevel),
		time.Duration(cfg.Connector.LogConsPeriod)*time.Second,
	)

	return dto, nil
}

// IsConfiguringPMC checks configuration stage
func (cfg *Config) IsConfiguringPMC() bool {
	return os.Getenv(InstallationModeEnv) == InstallationModePMC &&
		cfg.Connector.InstallationMode != InstallationModePMC
}

// Decrypt decrypts small messages
// golang.org/x/crypto/nacl/secretbox
func Decrypt(message, secret []byte) ([]byte, error) {
	var nonce [24]byte
	var secretKey [32]byte
	secretKey = sha256.Sum256(secret)
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
	var secretKey [32]byte
	secretKey = sha256.Sum256(secret)
	if _, err := io.ReadFull(rand.Reader, nonce[:]); err != nil {
		return nil, err
	}
	return secretbox.Seal(nonce[:], message, &nonce, &secretKey), nil
}
