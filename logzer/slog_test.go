package logzer

import (
	"context"
	"log/slog"
	"os"
	"testing"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/stretchr/testify/assert"
)

func TestNewSLogHandler(t *testing.T) {
	logFile, _ := os.CreateTemp("", "log")
	assert.NoError(t, logFile.Close())
	defer os.Remove(logFile.Name())

	zerolog.TimeFieldFormat = zerolog.TimeFormatUnixMs
	w := NewLoggerWriter(WithLogFile(&LogFile{FilePath: logFile.Name()}))
	log.Logger = zerolog.New(w).
		With().Timestamp().Caller().
		Logger()

	slogger := slog.New((&SLogHandler{CallerSkipFrame: 3}).
		WithGroup("foo.bar").
		WithAttrs([]slog.Attr{{Key: "foo", Value: slog.StringValue("bar")}}).
		WithGroup("bar.foo").
		WithAttrs([]slog.Attr{{Key: "fox", Value: slog.StringValue("box")}}))

	slogger.LogAttrs(context.TODO(), slog.LevelInfo, "__slogger__ message",
		slog.String("aaa", "bbb"), slog.Int("i", 111))

	content, err := os.ReadFile(logFile.Name())
	assert.NoError(t, err)
	assert.Contains(t, string(content), `logger=["foo.bar","bar.foo"]`)
	assert.Contains(t, string(content), `__slogger__ message`)
	assert.Contains(t, string(content), `aaa=bbb`)
}
