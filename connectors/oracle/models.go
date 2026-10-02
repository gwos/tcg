package oracle

import (
	"encoding/json"
	"time"

	"github.com/rs/zerolog/log"

	"github.com/gwos/tcg/sdk/mapping"
	"github.com/gwos/tcg/sdk/transit"
)

type ExtConfig struct {
	Ownership     transit.HostOwnershipType `json:"ownership,omitempty"`
	CheckInterval time.Duration             `json:"checkIntervalMinutes"`
	HostGroup     string                    `json:"customHostGroup"`
	HostPrefix    string                    `json:"customHostPrefix,omitempty"`

	OracleTenancyOCID     string `json:"oracleTenancyOCID"`
	OracleUserOCID        string `json:"oracleUserOCID"`
	OraclePrivateKey      string `json:"oraclePrivateKey"`
	OracleFingerprint     string `json:"oracleFingerprint"`
	OracleRegion          string `json:"oracleRegion"`
	OracleAggregationType string `json:"oracleAggregationType"`

	GWMapping
}

// UnmarshalJSON implements json.Unmarshaler.
func (cfg *ExtConfig) UnmarshalJSON(input []byte) error {
	type plain ExtConfig
	c := plain(*cfg)
	if err := json.Unmarshal(input, &c); err != nil {
		return err
	}
	if c.CheckInterval != cfg.CheckInterval {
		c.CheckInterval = c.CheckInterval * time.Minute
	}
	*cfg = ExtConfig(c)
	return nil
}

type GWMapping struct {
	Host    mapping.Mappings `json:"mapHostname"`
	Service mapping.Mappings `json:"mapService"`
}

// Prepare compiles mappings and drops the invalid ones
func (m *GWMapping) Prepare() {
	var err error
	if m.Service, err = m.Service.CompileValid(); err != nil {
		log.Warn().Err(err).Msg("failed to prepare service mappings")
	}
	if m.Host, err = m.Host.CompileValid(); err != nil {
		log.Warn().Err(err).Msg("failed to prepare host mappings")
	}
}
