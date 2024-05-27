package utils

import (
	"encoding/json"
	"regexp"

	"github.com/gwos/tcg/sdk/transit"
)

// UnmarshalMappings decodes the input bytes into a transit.Mappings struct
func UnmarshalMappings(input []byte) (*transit.Mappings, error) {
	var connector struct {
		Mappings transit.Mappings `json:"mappings"`
	}
	if err := json.Unmarshal(input, &connector); err != nil {
		return nil, err
	}

	for i, mapping := range connector.Mappings.HostLabel {
		connector.Mappings.HostLabel[i].Regexp = regexp.MustCompile(mapping.Source)
	}
	for i, mapping := range connector.Mappings.ServiceLabel {
		connector.Mappings.ServiceLabel[i].Regexp = regexp.MustCompile(mapping.Source)
	}

	return &connector.Mappings, nil
}
