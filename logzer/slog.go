package logzer

import (
	"context"
	"log/slog"
	"runtime"
	"sync"
	"time"

	"github.com/rs/zerolog"
	zlog "github.com/rs/zerolog/log"
)

// SLogHandler translates slog.Record into zerolog.Event
// inspired by https://github.com/golang/example/blob/master/slog-handler-guide/README.md
type SLogHandler struct {
	attrs  []slog.Attr
	groups []string

	once sync.Once

	// CallerSkipFrame is the frame count from Handle to the log call site,
	// used when the call site can't be found by the record PC
	CallerSkipFrame int
	GroupsFieldName string
}

func (h *SLogHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return zerolog.GlobalLevel() <= zerologLevel(level)
}

// zerologLevel maps slog levels, others are taken as info
func zerologLevel(level slog.Level) zerolog.Level {
	switch level {
	case slog.LevelDebug:
		return zerolog.DebugLevel
	case slog.LevelWarn:
		return zerolog.WarnLevel
	case slog.LevelError:
		return zerolog.ErrorLevel
	default:
		return zerolog.InfoLevel
	}
}

func (h *SLogHandler) Handle(ctx context.Context, r slog.Record) error {
	h.once.Do(func() {
		if h.GroupsFieldName == "" {
			h.GroupsFieldName = "logger"
		}
	})

	e := zlog.WithLevel(zerologLevel(r.Level))

	attr2e := func(attr slog.Attr) bool {
		switch attr.Value.Kind() {
		case slog.KindAny:
			_ = e.Any(attr.Key, attr.Value.Any())
		case slog.KindBool:
			_ = e.Bool(attr.Key, attr.Value.Bool())
		case slog.KindDuration:
			_ = e.Str(attr.Key, attr.Value.Duration().Truncate(time.Millisecond).String())
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

	e.CallerSkipFrame(h.callerSkip(r.PC)).Msg(r.Message)

	return nil
}

// callerSkip returns the frame count from Handle to the frame of pc,
// so the caller stays right however many handlers wrap this one.
// Frames are compared by function and line, which counts inlined frames
// the same way zerolog's runtime.Caller does.
//
//go:noinline
func (h *SLogHandler) callerSkip(pc uintptr) int {
	if pc == 0 {
		return h.CallerSkipFrame
	}
	want, _ := runtime.CallersFrames([]uintptr{pc}).Next()
	var pcs [64]uintptr
	// skip runtime.Callers and callerSkip, so the first frame is Handle
	frames := runtime.CallersFrames(pcs[:runtime.Callers(2, pcs[:])])
	for i := 0; ; i++ {
		f, more := frames.Next()
		if f.Function == want.Function && f.Line == want.Line {
			return i
		}
		if !more {
			return h.CallerSkipFrame
		}
	}
}

func (h *SLogHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	nested := h.clone()
	nested.attrs = append(nested.attrs, attrs...)
	return nested
}

func (h *SLogHandler) WithGroup(name string) slog.Handler {
	nested := h.clone()
	nested.groups = append(nested.groups, name)
	return nested
}

func (h *SLogHandler) clone() *SLogHandler {
	nested := &SLogHandler{CallerSkipFrame: h.CallerSkipFrame, GroupsFieldName: h.GroupsFieldName}
	nested.attrs = append(nested.attrs, h.attrs...)
	nested.groups = append(nested.groups, h.groups...)
	return nested
}
