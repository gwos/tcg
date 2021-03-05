package log

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"os"
	"testing"
	"time"

	"github.com/patrickmn/go-cache"
	"github.com/stretchr/testify/assert"
)

func TestConfig(t *testing.T) {
	logFile1, _ := ioutil.TempFile("", "log1")
	logFile2, _ := ioutil.TempFile("", "log2")
	defer os.Remove(logFile1.Name())
	defer os.Remove(logFile2.Name())

	writerHook.cache = cache.New(10*time.Minute, 100*time.Millisecond)

	Config(logFile1.Name(), 1<<20, 30, 3, 0)
	Debug("message debug1")
	Info("message info1")
	Warn("message warn1")

	Config(logFile2.Name(), 1<<20, 30, 2, 200*time.Millisecond)
	Debug("message debug")
	Info("message info")
	Info("message info")
	Info("message info")

	log1, err1 := ioutil.ReadFile(logFile1.Name())
	assert.NoError(t, err1)
	assert.Equal(t, 3, bytes.Count(log1, []byte("\n")))

	log2, err2 := ioutil.ReadFile(logFile2.Name())
	assert.NoError(t, err2)
	assert.Equal(t, 1, bytes.Count(log2, []byte("\n")))

	time.Sleep(400 * time.Millisecond)
	log2, err2 = ioutil.ReadFile(logFile2.Name())
	assert.NoError(t, err2)
	assert.Equal(t, 2, bytes.Count(log2, []byte("\n")))
}

func TestSanitizeEntries(t *testing.T) {
	tmpFile, _ := ioutil.TempFile("", "log")
	defer os.Remove(tmpFile.Name())
	Config(tmpFile.Name(), 1<<20, 30, 3, 0)
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
