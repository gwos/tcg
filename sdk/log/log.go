package log

import (
	"log/slog"
	"os"
)

// Logger defines slog logger and can be adapted to external log
var Logger = slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
	AddSource: true,
	Level:     slog.LevelInfo,
}).WithGroup("tcg.sdk"))

// XLogger is the extended SDK logger: it masks secrets and writes into the current Logger
var XLogger = slog.New(&maskHandler{})
