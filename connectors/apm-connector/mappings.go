package main

import (
	"encoding/json"
	"net"
	"net/url"
	"regexp"
	"strings"

	"github.com/gwos/tcg/sdk/transit"
)

// unmarshalMappings decodes the input bytes into a transit.Mappings struct
func unmarshalMappings(input []byte) (*transit.Mappings, error) {
	var connector struct {
		Mappings transit.Mappings `json:"mappings"`
	}
	err := json.Unmarshal(input, &connector)
	if err != nil {
		return nil, err
	}

	for _, mapping := range connector.Mappings.HostLabel {
		mapping.Regexp = regexp.MustCompile(mapping.Source)
	}
	for _, mapping := range connector.Mappings.ServiceLabel {
		mapping.Regexp = regexp.MustCompile(mapping.Source)
	}

	return &connector.Mappings, nil
}

func applyResourceMapping(resource string) (bool, string) {
	for _, mapping := range mappings.Resource {
		if mapping.Source == resource && mapping.Enabled {

			if strings.Contains(mapping.Destination, "$1") {
				u, err := url.Parse(resource)
				if err != nil {
					return false, ""
				}
				host := u.Host
				if strings.Contains(host, ":") {
					host, _, err = net.SplitHostPort(u.Host)
					if err != nil {
						return false, ""
					}
				}
				return true, strings.ReplaceAll(mapping.Destination, "$1", host)
			}
			return true, mapping.Destination
		}
	}
	return false, ""
}

func applyLabelMapping(mappings []transit.Mapping, labels map[string]string) (bool, string) {
	for i := len(mappings) - 1; i != -1; i-- {
		for key, value := range labels {
			if mappings[i].Regexp.Match([]byte(key)) && mappings[i].Enabled {
				return true, strings.ReplaceAll(mappings[i].Destination, "$1", value)
			}
		}
	}
	return false, ""
}
