package mapping

import (
	"encoding/json"
	"errors"
	"fmt"
	"regexp"

	"gopkg.in/yaml.v3"
)

var (
	ErrMapping = errors.New("mapping error")

	ErrMappingMissedTag     = fmt.Errorf("%w: %v", ErrMapping, "missed tag")
	ErrMappingMismatchedTag = fmt.Errorf("%w: %v", ErrMapping, "mismatched tag")
)

type Mapping struct {
	Tag string `yaml:"tag"`
	// Match tag value with regexp
	Matcher string `yaml:"matcher"`
	// Expand Template with matches
	Template string `yaml:"template"`

	matcher *regexp.Regexp
}

// UnmarshalJSON implements json.Unmarshaler.
func (p *Mapping) UnmarshalJSON(input []byte) error {
	type plain Mapping
	t := plain(*p)
	var err error
	if err = json.Unmarshal(input, &t); err != nil {
		return err
	}
	if t.matcher, err = regexp.Compile(t.Matcher); err != nil {
		return err
	}
	*p = Mapping(t)
	return nil
}

// UnmarshalYAML implements the yaml.Unmarshaler interface.
func (p *Mapping) UnmarshalYAML(value *yaml.Node) error {
	type plain Mapping
	t := plain(*p)
	var err error
	if err = value.Decode(&t); err != nil {
		return err
	}
	if t.matcher, err = regexp.Compile(t.Matcher); err != nil {
		return err
	}
	*p = Mapping(t)
	return nil
}

type Mappings []Mapping

func (p Mappings) Apply(tags map[string]string) (string, error) {
	result := []byte{}
	for _, mapping := range p {
		if mapping.Tag == "" {
			result = append(result, []byte(mapping.Template)...)
		} else if content, ok := tags[mapping.Tag]; ok {
			matches := mapping.matcher.FindAllStringSubmatchIndex(content, -1)
			if matches == nil {
				return "", fmt.Errorf("%w: %v", ErrMappingMismatchedTag, mapping.Tag)
			}
			for _, submatches := range matches {
				result = mapping.matcher.ExpandString(result, mapping.Template, content, submatches)
			}
		} else {
			return "", fmt.Errorf("%w: %v", ErrMappingMissedTag, mapping.Tag)
		}
	}
	return string(result), nil
}
