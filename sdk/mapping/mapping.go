package mapping

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
)

var (
	ErrMapping = errors.New("mapping error")

	ErrMappingCompile       = fmt.Errorf("%w: %v", ErrMapping, "compile")
	ErrMappingMissedTag     = fmt.Errorf("%w: %v", ErrMapping, "missed tag")
	ErrMappingMismatchedTag = fmt.Errorf("%w: %v", ErrMapping, "mismatched tag")
	ErrMappingEmptyResult   = fmt.Errorf("%w: %v", ErrMapping, "empty result")

	errNotCompiled = errors.New("not compiled")
)

type Mapping struct {
	Tag string `json:"tag"`
	// Match tag value with regexp
	Matcher string `json:"matcher"`
	// Expand Template with matches
	Template string `json:"template"`

	matcher *regexp.Regexp
	// keys holds the trimmed comma-separated Tag names used by ApplyOR
	keys []string
}

// NewMapping returns new mapping.
func NewMapping(tag, matcher, template string) *Mapping {
	p := &Mapping{Tag: tag, Matcher: matcher, Template: template}
	if p.Compile() != nil {
		return nil
	}
	return p
}

// Compile compiles matcher.
func (p *Mapping) Compile() error {
	matcher, err := regexp.Compile(p.Matcher)
	if err != nil {
		return err
	}
	p.matcher = matcher
	p.keys = strings.Split(p.Tag, ",")
	for i := range p.keys {
		p.keys[i] = strings.TrimSpace(p.keys[i])
	}
	return nil
}

// expand appends Template expanded for every match in content to dst.
// It reports false if content does not match.
func (p *Mapping) expand(dst []byte, content string) ([]byte, bool) {
	matches := p.matcher.FindAllStringSubmatchIndex(content, -1)
	if matches == nil {
		return dst, false
	}
	for _, submatches := range matches {
		dst = p.matcher.ExpandString(dst, p.Template, content, submatches)
	}
	return dst, true
}

// joinTags returns the comma-joined values of the keys tags.
// It reports false if any key is missing.
func (p *Mapping) joinTags(tags map[string]string) (string, bool) {
	if len(p.keys) == 1 {
		val, ok := tags[p.keys[0]]
		return val, ok
	}
	n := len(p.keys) - 1
	for _, key := range p.keys {
		val, ok := tags[key]
		if !ok {
			return "", false
		}
		n += len(val)
	}
	var sb strings.Builder
	sb.Grow(n)
	for i, key := range p.keys {
		if i > 0 {
			sb.WriteByte(',')
		}
		sb.WriteString(tags[key])
	}
	return sb.String(), true
}

type Mappings []Mapping

func (p Mappings) Apply(tags map[string]string) (string, error) {
	var result []byte
	for i := range p {
		mapping := &p[i]
		if mapping.Tag == "" {
			result = append(result, mapping.Template...)
			continue
		}
		if mapping.matcher == nil {
			return "", mapping.errCompile(i, errNotCompiled)
		}
		content, ok := tags[mapping.Tag]
		if !ok {
			return "", fmt.Errorf("%w: %v", ErrMappingMissedTag, mapping.Tag)
		}
		if result, ok = mapping.expand(result, content); !ok {
			return "", fmt.Errorf("%w: %v", ErrMappingMismatchedTag, mapping.Tag)
		}
	}
	return string(result), nil
}

func (p Mappings) ApplyOR(tags map[string]string) (string, error) {
	mismatched := false
	for i := range p {
		mapping := &p[i]
		if mapping.Tag == "" {
			return mapping.Template, nil
		}
		if mapping.matcher == nil {
			return "", mapping.errCompile(i, errNotCompiled)
		}
		content, ok := mapping.joinTags(tags)
		if !ok {
			continue
		}
		result, ok := mapping.expand(nil, content)
		if !ok {
			mismatched = true
			continue
		}
		return string(result), nil
	}

	if mismatched {
		return "", ErrMappingMismatchedTag
	}
	if len(p) > 0 {
		return "", ErrMappingMissedTag
	}
	return "", nil
}

// Lookup applies ApplyOR and tells unconfigured mappings apart from failed ones.
// It returns ok=false with a nil error when p is empty,
// an error when no mapping matches or the mapped value is empty,
// and the mapped value with ok=true otherwise.
func (p Mappings) Lookup(tags map[string]string) (string, bool, error) {
	if len(p) == 0 {
		return "", false, nil
	}
	v, err := p.ApplyOR(tags)
	if err == nil && v == "" {
		err = ErrMappingEmptyResult
	}
	return v, err == nil, err
}

// Matches reports whether any mapping applies to tags with a non-empty result.
// It reports false when p is empty.
func (p Mappings) Matches(tags map[string]string) bool {
	v, err := p.ApplyOR(tags)
	return err == nil && v != ""
}

// Compile compiles mappings matchers.
// It stops at the first invalid matcher.
func (p Mappings) Compile() error {
	for i := range p {
		if err := p[i].Compile(); err != nil {
			return p[i].errCompile(i, err)
		}
	}
	return nil
}

// CompileValid compiles mappings matchers and returns the mappings that compiled.
// The errors of the dropped mappings are joined into the returned error.
func (p Mappings) CompileValid() (Mappings, error) {
	var (
		valid Mappings
		errs  []error
	)
	for i := range p {
		if err := p[i].Compile(); err != nil {
			errs = append(errs, p[i].errCompile(i, err))
			continue
		}
		valid = append(valid, p[i])
	}
	return valid, errors.Join(errs...)
}

func (p *Mapping) errCompile(i int, err error) error {
	return fmt.Errorf("%w [%d:%v]: %v", ErrMappingCompile, i, p.Tag, err)
}

// MatchString reports whether str matches any mapping matcher, or true if p is empty.
// Mappings that are not compiled never match.
func (p Mappings) MatchString(str string) bool {
	if len(p) == 0 {
		return true
	}
	for i := range p {
		if p[i].matcher != nil && p[i].matcher.MatchString(str) {
			return true
		}
	}
	return false
}
