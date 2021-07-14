package logger

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"testing"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/stretchr/testify/assert"
)

func TestLogFilter(t *testing.T) {
	logFile, _ := ioutil.TempFile("", "log")
	assert.NoError(t, logFile.Close())
	defer os.Remove(logFile.Name())

	SetLogger(WithLogFile(&LogFile{
		FilePath: logFile.Name(),
		MaxSize:  0,
		Rotate:   0,
	}))
	payload1, _ := json.Marshal(struct{ Password, Token string }{
		Password: `PASS
		WORD`,
		Token: `TOK-EN`,
	})
	payload2, _ := json.Marshal(map[string]string{
		"somePassword": `"some"PASS"`,
		"someToken":    `"some""TOK`,
	})
	log.Info().
		Str("password", "PASSword").
		Dict("dict", zerolog.Dict().Str("token", "TOKen")).
		RawJSON("payload1", []byte(payload1)).
		RawJSON("payload2", []byte(payload2)).
		RawJSON("payload3", []byte(`{"password": "PASSWORD", "token": "TOKEN"}`)).
		RawJSON("payload4", []byte(`{"somePassword":"PASS\"\"\nWORD","Token":"TOK\\EN"}`)).
		Msg("message")

	content, err := ioutil.ReadFile(logFile.Name())
	assert.NoError(t, err)
	assert.NotContains(t, string(content), "PASS")
	assert.NotContains(t, string(content), "TOK")
	assert.Contains(t, string(content), `"Password":"***"`)
	assert.Contains(t, string(content), `"Token":"***"`)
	assert.Contains(t, string(content), `"somePassword":"***"`)
	assert.Contains(t, string(content), `"someToken":"***"`)
	assert.Contains(t, string(content), `"password":"***"`)
	assert.Contains(t, string(content), `"token":"***"`)
}

func TestLogRotate(t *testing.T) {
	logFile, _ := ioutil.TempFile("", "log")
	defer os.Remove(logFile.Name())
	defer os.Remove(logFile.Name() + ".1")
	defer os.Remove(logFile.Name() + ".2")
	defer os.Remove(logFile.Name() + ".3")

	SetLogger(WithLogFile(&LogFile{
		FilePath: logFile.Name(),
		MaxSize:  200,
		Rotate:   3,
	}))

	log.Debug().Msg("message debug1")
	log.Info().Msg("message info1")
	log.Warn().Msg("message warn1") // expect rotation by maxSize

	log0, elog0 := ioutil.ReadFile(logFile.Name())
	log1, elog1 := ioutil.ReadFile(logFile.Name() + ".1")
	assert.NoError(t, elog0)
	assert.NoError(t, elog1)
	assert.Contains(t, string(log1), "debug1")
	assert.Contains(t, string(log1), "info1")
	assert.Contains(t, string(log0), "warn1")

	log.Warn().Msg("message debug2")
	log.Warn().Msg("message info2") // expect rotation by maxSize
	log.Warn().Msg("message warn2")
	log.Warn().Msg("message debug3") // expect rotation by maxSize
	log.Warn().Msg("message info3")
	log.Warn().Msg("message warn3") // expect rotation by maxSize

	log0, elog0 = ioutil.ReadFile(logFile.Name())
	log1, elog1 = ioutil.ReadFile(logFile.Name() + ".1")
	log2, elog2 := ioutil.ReadFile(logFile.Name() + ".2")
	log3, elog3 := ioutil.ReadFile(logFile.Name() + ".3")

	// t.Logf("\n#log0\n%s\n", log0)
	// t.Logf("\n#log1\n%s\n", log1)
	// t.Logf("\n#log2\n%s\n", log2)
	// t.Logf("\n#log3\n%s\n", log3)

	assert.NoError(t, elog0)
	assert.NoError(t, elog1)
	assert.NoError(t, elog2)
	assert.NoError(t, elog3)
	assert.Contains(t, string(log3), "warn1")
	assert.Contains(t, string(log3), "debug2")
	assert.Contains(t, string(log2), "info2")
	assert.Contains(t, string(log2), "warn2")
	assert.Contains(t, string(log1), "debug3")
	assert.Contains(t, string(log1), "info3")
	assert.Contains(t, string(log0), "warn3")
}
