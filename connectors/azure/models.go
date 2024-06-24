package azure

import (
	"encoding/json"
	"time"

	"github.com/gwos/tcg/sdk/mapping"
	"github.com/gwos/tcg/sdk/transit"
	"github.com/rs/zerolog/log"
)

type ExtConfig struct {
	Ownership     transit.HostOwnershipType `json:"ownership,omitempty"`
	CheckInterval time.Duration             `json:"checkIntervalMinutes"`
	HostGroup     string                    `json:"customHostGroup"`

	AzureTenantID       string `json:"azureTenantId"`
	AzureClientID       string `json:"azureClientId"`
	AzureClientSecret   string `json:"azureClientSecret"`
	AzureSubscriptionID string `json:"azureSubscriptionId"`

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

// Prepare compiles mappings
func (m *GWMapping) Prepare() {
	var hg, hn mapping.Mappings

	for i := range m.Service {
		if err := m.Service[i].Compile(); err != nil {
			log.Warn().Err(err).Interface("mapping", m.Service[i]).Msg("failed to prepare mapping")
			continue
		}
		hg = append(hg, m.Service[i])
	}

	for i := range m.Host {
		if err := m.Host[i].Compile(); err != nil {
			log.Warn().Err(err).Interface("mapping", m.Host[i]).Msg("failed to prepare mapping")
			continue
		}
		hn = append(hn, m.Host[i])
	}

	m.Service, m.Host = hg, hn
}
