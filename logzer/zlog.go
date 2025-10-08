package logzer

import (
	"bytes"

	"github.com/rs/zerolog"
)

// StrOrJSON is shortcut for creating zerolog.LogObjectMarshaler
// that produces `RawJSON` or `string` field for provided key
// based on quick check for JSON-like content
//
// e.EmbedObject(logzer.StrOrJSON("response", body))
func StrOrJSON(k string, s []byte) ZlogStrOrJSON {
	return ZlogStrOrJSON{Key: k, Bytes: bytes.TrimSpace(s)}
}

// ZlogStrOrJSON implements zerolog.LogObjectMarshaler interface
// that produces `RawJSON` or `string` field for provided key
// based on quick check for JSON-like content
type ZlogStrOrJSON struct {
	Key   string
	Bytes []byte
}

// MarshalZerologObject implements zerolog.LogObjectMarshaler interface
func (s ZlogStrOrJSON) MarshalZerologObject(e *zerolog.Event) {
	if s.IsJSON() {
		e.RawJSON(s.Key, s.Bytes)
	} else {
		e.Str(s.Key, string(s.Bytes))
	}
}

// IsJSON does quick check for JSON-like content
func (s ZlogStrOrJSON) IsJSON() bool {
	if len(s.Bytes) >= 2 &&
		((s.Bytes[0] == '{' && s.Bytes[len(s.Bytes)-1] == '}') ||
			(s.Bytes[0] == '[' && s.Bytes[len(s.Bytes)-1] == ']')) {
		return true
	}
	return false
}
