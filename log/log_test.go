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

	Config(logFile1.Name(), 0, 1, 3, 0)
	Debug("message debug1")
	Info("message info1")
	Warn("message warn1")

	Config(logFile2.Name(), 0, 1, 2, 200*time.Millisecond)
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

func TestLogrotate(t *testing.T) {
	logFile, _ := ioutil.TempFile("", "log")
	defer os.Remove(logFile.Name())
	defer os.Remove(logFile.Name() + ".1")
	defer os.Remove(logFile.Name() + ".2")
	defer os.Remove(logFile.Name() + ".3")

	Config(logFile.Name(), 140, 3, 3, 0)
	Debug("message debug1")
	Info("message info1")
	Warn("message warn1") // expect rotation by maxSize

	log0, elog0 := ioutil.ReadFile(logFile.Name())
	log1, elog1 := ioutil.ReadFile(logFile.Name() + ".1")
	assert.NoError(t, elog0)
	assert.NoError(t, elog1)
	assert.Contains(t, string(log0), "warn1")
	assert.Contains(t, string(log1), "info1")
	assert.Contains(t, string(log1), "debug1")

	time.Sleep(200 * time.Millisecond)
	Debug("message debug2")
	Info("message info2") // expect rotation by maxSize
	Warn("message warn2")
	Debug("message debug3") // expect rotation by maxSize

	log0, elog0 = ioutil.ReadFile(logFile.Name())
	log1, elog1 = ioutil.ReadFile(logFile.Name() + ".1")
	log2, elog2 := ioutil.ReadFile(logFile.Name() + ".2")
	log3, elog3 := ioutil.ReadFile(logFile.Name() + ".3")
	assert.NoError(t, elog0)
	assert.NoError(t, elog1)
	assert.NoError(t, elog2)
	assert.NoError(t, elog3)

	assert.Contains(t, string(log3), "debug1")
	assert.Contains(t, string(log3), "info1")
	assert.Contains(t, string(log2), "warn1")
	assert.Contains(t, string(log2), "debug2")
	assert.Contains(t, string(log1), "info2")
	assert.Contains(t, string(log1), "warn2")
	assert.Contains(t, string(log0), "debug3")
}

func TestSanitizeEntries(t *testing.T) {
	tmpFile, _ := ioutil.TempFile("", "log")
	defer os.Remove(tmpFile.Name())
	Config(tmpFile.Name(), 0, 1, 3, 0)
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
