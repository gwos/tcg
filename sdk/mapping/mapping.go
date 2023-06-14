package mapping

import (
	"errors"
	"fmt"
	"regexp"
)

var (
	ErrMapping = errors.New("mapping error")

	ErrMappingCompile       = fmt.Errorf("%w: %v", ErrMapping, "compile")
	ErrMappingMissedTag     = fmt.Errorf("%w: %v", ErrMapping, "missed tag")
	ErrMappingMismatchedTag = fmt.Errorf("%w: %v", ErrMapping, "mismatched tag")
)

type Mapping struct {
	Tag string `json:"tag"`
	// Match tag value with regexp
	Matcher string `json:"matcher"`
	// Expand Template with matches
	Template string `json:"template"`

	matcher *regexp.Regexp
}

// NewMapping returns new mapping.
func NewMapping(tag, matcher, template string) *Mapping {
	if m, err := regexp.Compile(matcher); err != nil {
		return nil
	} else {
		return &Mapping{Tag: tag, Matcher: matcher, Template: template, matcher: m}
	}
}

// Compile compiles matcher.
func (p *Mapping) Compile() error {
	if matcher, err := regexp.Compile(p.Matcher); err != nil {
		return err
	} else {
		p.matcher = matcher
	}
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

// Compile compiles mappings matchers.
func (p Mappings) Compile() error {
	for i := range p {
		if err := p[i].Compile(); err != nil {
			return fmt.Errorf("%w [%d:%v]: %v", ErrMappingCompile, i, p[i].Tag, err)
		}
	}
	return nil
}
