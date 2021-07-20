package utils

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_XORpass(t *testing.T) {
	s, err := Decrypt("1e0606071b07121c520e160a1b451e00164d00091a")
	assert.NoError(t, err)
	assert.Equal(t, "groundworkdevelopment", s)
	s, err = Decrypt("18150813580107114248435e5c115f5d541257")
	assert.NoError(t, err)
	assert.Equal(t, "aaaa-bbbb-1111-2222", s)

	assert.Equal(t, "1e0606071b07121c520e160a1b451e00164d00091a",
		Encrypt("groundworkdevelopment"))
	assert.Equal(t, "18150813580107114248435e5c115f5d541257",
		Encrypt("aaaa-bbbb-1111-2222"))
}
