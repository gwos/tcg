package utils

import (
	"os/exec"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_decrypt(t *testing.T) {

	t.Run("invoke_perl", func(t *testing.T) {
		if _, err := exec.LookPath("perl"); err != nil {
			t.Skip("perl is not avaliable:", err)
		}

		s, err := decrypt("48465a464055524b1955130d0e441709")
		assert.NoError(t, err)
		assert.Equal(t, "1234567890abcdef", s)
	})
}
