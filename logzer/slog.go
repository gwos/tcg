package logzer

import (
	"context"
	"log/slog"
	"sync"

	"github.com/rs/zerolog"
	zlog "github.com/rs/zerolog/log"
)

// SLogHandler translates slog.Record into zerolog.Event
// inspired by https://github.com/golang/example/blob/master/slog-handler-guide/README.md
type SLogHandler struct {
	attrs  []slog.Attr
	groups []string

	once sync.Once

	CallerSkipFrame int
	GroupsFieldName string
}

func (h *SLogHandler) Enabled(ctx context.Context, level slog.Level) bool {
	l := zerolog.GlobalLevel()
	switch level {
	case slog.LevelDebug:
		return l <= zerolog.DebugLevel
	case slog.LevelInfo:
		return l <= zerolog.InfoLevel
	case slog.LevelWarn:
		return l <= zerolog.WarnLevel
	case slog.LevelError:
		return l <= zerolog.ErrorLevel
	default:
		return l <= zerolog.InfoLevel
	}
}

func (h *SLogHandler) Handle(ctx context.Context, r slog.Record) error {
	h.once.Do(func() {
		if h.GroupsFieldName == "" {
			h.GroupsFieldName = "logger"
		}
	})

	var l zerolog.Level
	switch r.Level {
	case slog.LevelDebug:
		l = zerolog.DebugLevel
	case slog.LevelInfo:
		l = zerolog.InfoLevel
	case slog.LevelWarn:
		l = zerolog.WarnLevel
	case slog.LevelError:
		l = zerolog.ErrorLevel
	default:
		l = zerolog.InfoLevel
	}
	e := zlog.WithLevel(l)

	attr2e := func(attr slog.Attr) bool {
		switch attr.Value.Kind() {
		case slog.KindAny:
			_ = e.Any(attr.Key, attr.Value.Any())
		case slog.KindBool:
			_ = e.Bool(attr.Key, attr.Value.Bool())
		case slog.KindDuration:
			_ = e.Dur(attr.Key, attr.Value.Duration())
		case slog.KindFloat64:
			_ = e.Float64(attr.Key, attr.Value.Float64())
		case slog.KindInt64:
			_ = e.Int64(attr.Key, attr.Value.Int64())
		case slog.KindString:
			_ = e.Str(attr.Key, attr.Value.String())
		case slog.KindTime:
			_ = e.Time(attr.Key, attr.Value.Time())
		case slog.KindUint64:
			_ = e.Uint64(attr.Key, attr.Value.Uint64())
		case slog.KindGroup:
			_ = e.Str(attr.Key, attr.Value.String())
		case slog.KindLogValuer:
			_ = e.Any(attr.Key, attr.Value.Any())
		}
		return true
	}

	if len(h.groups) > 0 {
		_ = e.Strs(h.GroupsFieldName, h.groups)
	}
	for _, attr := range h.attrs {
		_ = attr2e(attr)
	}
	r.Attrs(attr2e)

	e.CallerSkipFrame(h.CallerSkipFrame).Msg(r.Message)

	return nil
}

func (h *SLogHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	nested := &SLogHandler{CallerSkipFrame: h.CallerSkipFrame, GroupsFieldName: h.GroupsFieldName}
	nested.attrs = append(nested.attrs, h.attrs...)
	nested.groups = append(nested.groups, h.groups...)
	nested.attrs = append(nested.attrs, attrs...)
	return nested
}

func (h *SLogHandler) WithGroup(name string) slog.Handler {
	nested := &SLogHandler{CallerSkipFrame: h.CallerSkipFrame, GroupsFieldName: h.GroupsFieldName}
	nested.attrs = append(nested.attrs, h.attrs...)
	nested.groups = append(nested.groups, h.groups...)
	nested.groups = append(nested.groups, name)
	return nested
}
