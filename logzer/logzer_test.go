package logzer

import (
	"bytes"
	"encoding/json"
	"os"
	"testing"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/stretchr/testify/assert"
)

func TestLogCondense(t *testing.T) {
	logFile, _ := os.CreateTemp("", "log")
	defer os.Remove(logFile.Name())

	zerolog.TimeFieldFormat = zerolog.TimeFormatUnixMs
	w := NewLoggerWriter(
		WithCondense(time.Millisecond*2),
		WithLogFile(&LogFile{FilePath: logFile.Name()}))
	log.Logger = zerolog.New(w).
		With().Timestamp().Caller().
		Logger()
	logfun := func(lvl zerolog.Level, msg string) { log.WithLevel(lvl).Msg(msg) }
	logfun(zerolog.DebugLevel, "message debug")
	logfun(zerolog.InfoLevel, "message info")
	logfun(zerolog.InfoLevel, "message info") // expect condense
	logfun(zerolog.InfoLevel, "message info") // expect condense

	content, err := os.ReadFile(logFile.Name())
	assert.NoError(t, err)
	assert.Equal(t, 2, bytes.Count(content, []byte("\n")))

	time.Sleep(time.Millisecond * 400) // expect condense output
	content, err = os.ReadFile(logFile.Name())
	assert.NoError(t, err)
	assert.Equal(t, 3, bytes.Count(content, []byte("\n")))
	assert.Contains(t, string(content), `condense`)
}

func TestLogFilter(t *testing.T) {
	logFile, _ := os.CreateTemp("", "log")
	assert.NoError(t, logFile.Close())
	defer os.Remove(logFile.Name())

	zerolog.TimeFieldFormat = zerolog.TimeFormatUnixMs
	w := NewLoggerWriter(WithLogFile(&LogFile{FilePath: logFile.Name()}))
	log.Logger = zerolog.New(w).
		With().Timestamp().Caller().
		Logger()
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
		RawJSON("payload1", payload1).
		RawJSON("payload2", payload2).
		RawJSON("payload3", []byte(`{"password": "PASSWORD", "token": "TOKEN"}`)).
		RawJSON("payload4", []byte(`{"somePassword":"PASS\"\"\nWORD","Token":"TOK\\EN"}`)).
		Msg("message")

	content, err := os.ReadFile(logFile.Name())
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
	logFile, _ := os.CreateTemp("", "log")
	defer os.Remove(logFile.Name())
	defer os.Remove(logFile.Name() + ".1")
	defer os.Remove(logFile.Name() + ".2")
	defer os.Remove(logFile.Name() + ".3")

	zerolog.TimeFieldFormat = zerolog.TimeFormatUnixMs
	w := NewLoggerWriter(WithLogFile(&LogFile{
		FilePath: logFile.Name(),
		MaxSize:  180,
		Rotate:   3,
	}))
	log.Logger = zerolog.New(w).
		With().Timestamp().Caller().
		Logger()

	log.Debug().Msg("message debug1")
	log.Info().Msg("message info1")
	log.Warn().Msg("message warn1") // expect rotation by maxSize

	log0, err0 := os.ReadFile(logFile.Name())
	log1, err1 := os.ReadFile(logFile.Name() + ".1")
	assert.NoError(t, err0)
	assert.NoError(t, err1)
	assert.Contains(t, string(log1), "debug1")
	assert.Contains(t, string(log1), "info1")
	assert.Contains(t, string(log0), "warn1")

	log.Warn().Msg("message debug2")
	log.Warn().Msg("message info2") // expect rotation by maxSize
	log.Warn().Msg("message warn2")
	log.Warn().Msg("message debug3") // expect rotation by maxSize
	log.Warn().Msg("message info3")
	log.Warn().Msg("message warn3") // expect rotation by maxSize

	log0, err0 = os.ReadFile(logFile.Name())
	log1, err1 = os.ReadFile(logFile.Name() + ".1")
	log2, err2 := os.ReadFile(logFile.Name() + ".2")
	log3, err3 := os.ReadFile(logFile.Name() + ".3")

	// println("\n#log0\n", string(log0))
	// println("\n#log1\n", string(log1))
	// println("\n#log2\n", string(log2))
	// println("\n#log3\n", string(log3))

	assert.NoError(t, err0)
	assert.NoError(t, err1)
	assert.NoError(t, err2)
	assert.NoError(t, err3)
	assert.Contains(t, string(log3), "warn1")
	assert.Contains(t, string(log3), "debug2")
	assert.Contains(t, string(log2), "info2")
	assert.Contains(t, string(log2), "warn2")
	assert.Contains(t, string(log1), "debug3")
	assert.Contains(t, string(log1), "info3")
	assert.Contains(t, string(log0), "warn3")
}
