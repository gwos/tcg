package config

import (
	"encoding/json"
	"hash/fnv"
)

// Hashsum calculates FNV non-cryptographic hash suitable for checking the equality
func Hashsum(args ...any) ([]byte, error) {
	h := fnv.New128()
	for _, arg := range args {
		s, err := json.Marshal(arg)
		if err != nil {
			return nil, err
		}
		_, _ = h.Write(s)
	}
	return h.Sum(nil), nil
}
