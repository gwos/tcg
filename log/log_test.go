package log

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSanitizeEntries(t *testing.T) {
	tmpFile, _ := ioutil.TempFile("", "log")
	defer os.Remove(tmpFile.Name())
	Config(tmpFile.Name(), 3, 0)
	payload1, _ := json.Marshal(struct{ Password, Token string }{
		Password: `PASS
		WORD`,
		Token: `TOK-EN`,
	})
	payload2, _ := json.Marshal(map[string]string{
		"somePassword": `"some"PASS"`,
		"someToken":    `"some""TOK`,
	})
	With(Fields{
		"payload1": string(payload1),
		"payload2": string(payload2),
		"payload3": `{"password": "PASSWORD", "token": "TOKEN"}`,
		"payload4": `{"somePassword":"PASS\"\"\nWORD","Token":"TOK\\EN"}`,
	}).Info("msg")
	content, err := ioutil.ReadFile(tmpFile.Name())
	assert.NoError(t, err)
	assert.NotContains(t, string(content), "PASS")
	assert.NotContains(t, string(content), "TOK")
	assert.Contains(t, string(content), `{"Password":"***","Token":"***"}`)
	assert.Contains(t, string(content), `{"somePassword":"***","someToken":"***"}`)
	assert.Contains(t, string(content), `{"password": "***", "token": "***"}`)
	assert.Contains(t, string(content), `{"somePassword":"***","Token":"***"}`)
}
