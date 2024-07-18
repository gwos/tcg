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
