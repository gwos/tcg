package config

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"github.com/gwos/tng/log"
	"github.com/kelseyhightower/envconfig"
	"golang.org/x/crypto/nacl/secretbox"
	"gopkg.in/yaml.v3"
	"io"
	"io/ioutil"
	"math"
	"os"
	"path"
	"strings"
	"sync"
)

var once sync.Once
var cfg *Config

// ConfigStringConstant defines string constant type
type ConfigStringConstant string

// ConfigEnv defines environment variable for config file path, overrides the ConfigName
// ConfigName defines default filename for look in work directory if ConfigEnv is empty
// EnvConfigPrefix defines name prefix for environment variables
//   for example: TNG_CONNECTOR_NATSSTORETYPE
const (
	ConfigEnv           ConfigStringConstant = "TNG_CONFIG"
	ConfigName                               = "tng_config.yaml"
	EnvConfigPrefix                          = "TNG"
	SecKeyEnv                                = "TNG_SECKEY"
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

// Connector defines TNG Connector configuration
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
	// NatsHost accepts value for combined "host:port"
	// used as `strings.Split(natsHost, ":")`
	NatsHost string `yaml:"natsHost"`
	// LogConsPeriod accepts number of seconds
	// if 0 turn off consolidation
	LogConsPeriod int `yaml:"logConsPeriod"`
	// LogFile accepts file path to log in addition to stdout
	LogFile          string   `yaml:"logFile"`
	LogLevel         LogLevel `yaml:"logLevel"`
	Enabled          bool     `yaml:"enabled"`
	InstallationMode string   `yaml:"installationMode,omitempty"`
}

// ConnectorDTO defines TNG Connector configuration
type ConnectorDTO struct {
	AgentID       string        `json:"agentId"`
	AppName       string        `json:"appName"`
	AppType       string        `json:"appType"`
	TngURL        string        `json:"tngUrl"`
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
	if s := os.Getenv(string(SecKeyEnv)); s != "" {
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
		s := os.Getenv(string(SecKeyEnv))
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

// Config defines TNG Agent configuration
type Config struct {
	Connector     *Connector    `yaml:"connector"`
	DSConnection  *DSConnection `yaml:"dsConnection"`
	GWConnections GWConnections `yaml:"gwConnections"`
}

// GetConfig implements Singleton pattern
func GetConfig() *Config {
	once.Do(func() {
		// set defaults
		cfg = &Config{
			Connector: &Connector{
				ControllerAddr:   ":8099",
				LogConsPeriod:    0,
				LogLevel:         1,
				NatsAckWait:      30,
				NatsMaxInflight:  math.MaxInt32,
				NatsFilestoreDir: "natsstore",
				NatsStoreType:    "FILE",
				NatsHost:         "127.0.0.1:4222",
			},
			DSConnection: &DSConnection{},
		}

		configPath := os.Getenv(string(ConfigEnv))
		if configPath == "" {
			wd, err := os.Getwd()
			if err != nil {
				log.Warn(err)
				wd = ""
			}
			configPath = path.Join(wd, ConfigName)
		}

		if data, err := ioutil.ReadFile(configPath); err != nil {
			log.Warn(err)
		} else {
			if err := yaml.Unmarshal(data, cfg); err != nil {
				log.Warn(err)
			}
		}

		if err := envconfig.Process(EnvConfigPrefix, cfg); err != nil {
			log.Warn(err)
		}

		log.Config(cfg.Connector.LogFile, int(cfg.Connector.LogLevel), cfg.Connector.LogConsPeriod)
	})
	return cfg
}

func (cfg *Config) loadConnector(data []byte) (*ConnectorDTO, error) {
	var dto ConnectorDTO
	if err := json.Unmarshal(data, &dto); err != nil {
		log.Error(err.Error())
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
		log.Error(err.Error())
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

// LoadConnectorDTO loads ConnectorDTO into Config
func (cfg *Config) LoadConnectorDTO(data []byte) (*ConnectorDTO, error) {
	/* load as ConnectorDTO */
	dto, err := cfg.loadConnector(data)
	if err != nil {
		return nil, err
	}
	/* load as struct with advanced prefixes field */
	if err := cfg.loadAdvancedPrefixes(data); err != nil {
		return nil, err
	}

	log.Config(cfg.Connector.LogFile, int(cfg.Connector.LogLevel), cfg.Connector.LogConsPeriod)

	if cfg.IsConfiguringPMC() {
		cfg.Connector.InstallationMode = string(InstallationModePMC)
	}

	if output, err := yaml.Marshal(cfg); err != nil {
		log.Warn(err)
	} else {
		configPath := os.Getenv(string(ConfigEnv))
		if configPath == "" {
			wd, err := os.Getwd()
			if err != nil {
				log.Warn(err)
				wd = ""
			}
			configPath = path.Join(wd, ConfigName)
		}
		if err := ioutil.WriteFile(configPath, output, 0644); err != nil {
			log.Warn(err)
		}
	}

	return dto, nil
}

// IsConfiguringPMC checks configuration stage
func (cfg *Config) IsConfiguringPMC() bool {
	return os.Getenv(string(InstallationModeEnv)) == string(InstallationModePMC) &&
		cfg.Connector.InstallationMode != string(InstallationModePMC)
}

// Decrypt decrypts small messages
// golang.org/x/crypto/nacl/secretbox
func Decrypt(message, secret []byte) ([]byte, error) {
	var nonce [24]byte
	var secretKey [32]byte
	secretKey = sha256.Sum256([]byte(secret))
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
	secretKey = sha256.Sum256([]byte(secret))
	if _, err := io.ReadFull(rand.Reader, nonce[:]); err != nil {
		return nil, err
	}
	return secretbox.Seal(nonce[:], message, &nonce, &secretKey), nil
}
