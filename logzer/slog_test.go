package logzer

import (
	"context"
	"log/slog"
	"os"
	"runtime"
	"strconv"
	"testing"

	sdklog "github.com/gwos/tcg/sdk/log"
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

func TestSLogHandlerCaller(t *testing.T) {
	logFile, _ := os.CreateTemp("", "log")
	assert.NoError(t, logFile.Close())
	defer os.Remove(logFile.Name())

	w := NewLoggerWriter(WithLogFile(&LogFile{FilePath: logFile.Name()}))
	log.Logger = zerolog.New(w).
		With().Timestamp().Caller().
		Logger()

	orig := sdklog.Logger
	defer func() { sdklog.Logger = orig }()
	// a wrong skip count is ignored while the record PC resolves the caller
	sdklog.Logger = slog.New(&SLogHandler{CallerSkipFrame: 1})

	line := func() string {
		_, _, l, _ := runtime.Caller(1)
		return "slog_test.go:" + strconv.Itoa(l+1)
	}
	want := []string{line()}
	sdklog.XLogger.LogAttrs(context.TODO(), slog.LevelWarn, "__xlogger__ message",
		slog.Any("header", map[string][]string{"Authorization": {"Bearer abc"}}))
	want = append(want, line())
	sdklog.XLogger.Warn("__xlogger__ warn")
	want = append(want, line())
	sdklog.Logger.Warn("__sdklogger__ warn")
	want = append(want, line())
	sdklog.XLogger.With("k", "v").WithGroup("g").Warn("__xlogger__ nested")

	content, err := os.ReadFile(logFile.Name())
	assert.NoError(t, err)
	assert.Contains(t, string(content), `__xlogger__ message`)
	assert.NotContains(t, string(content), `abc`)
	for _, w := range want {
		assert.Contains(t, string(content), w+" ")
	}
}
