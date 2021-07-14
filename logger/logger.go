package logger

import (
	"container/ring"
	"io"
	"os"
	"regexp"
	"sync"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

var (
	logFile    io.WriteCloser
	lastErrors = &LogBuffer{
		Level: zerolog.ErrorLevel,
		Ring:  ring.New(10),
	}
	formatter = &zerolog.ConsoleWriter{
		Out: os.Stdout,
		// NoColor:    true,
		TimeFormat: time.RFC3339,
	}
	filter = &FilterWriter{
		LevelWriter: zerolog.MultiLevelWriter(formatter, lastErrors),
		Re: map[*regexp.Regexp][]byte{
			regexp.MustCompile(`((?i:password|token)"[^:]*:[^"]*)"(?:[^\\"]*(?:\\")*[\\]*)*"`): []byte(`${1}"***"`),
		},
	}
)

// FilterWriter implements sanitizing writes by Regexp map
type FilterWriter struct {
	zerolog.LevelWriter
	mu sync.Mutex
	Re map[*regexp.Regexp][]byte
}

// Write implements io.Writer interface
func (w *FilterWriter) Write(p []byte) (int, error) {
	return w.WriteLevel(zerolog.NoLevel, p)
}

// WriteLevel implements zerolog.LevelWriter interface
func (w *FilterWriter) WriteLevel(l zerolog.Level, p []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	for reg, repl := range w.Re {
		p = reg.ReplaceAll(p, repl)
	}
	return w.LevelWriter.WriteLevel(l, p)
}

// LogBuffer collects writes if level passed
type LogBuffer struct {
	mu    sync.Mutex
	Level zerolog.Level
	Ring  *ring.Ring
}

// Records returns collected writes
func (lb *LogBuffer) Records() []LogRecord {
	rec := []LogRecord{}
	lb.Ring.Do(func(p interface{}) {
		if p != nil {
			rec = append(rec, p.(LogRecord))
		}
	})
	return rec
}

// Write implements io.Writer interface
func (lb *LogBuffer) Write(p []byte) (int, error) {
	return len(p), nil
}

// WriteLevel implements zerolog.LevelWriter interface
func (lb *LogBuffer) WriteLevel(l zerolog.Level, p []byte) (int, error) {
	lb.mu.Lock()
	defer lb.mu.Unlock()
	if l >= lb.Level {
		/* store the copy as source could be updated */
		cp := make([]byte, len(p))
		copy(cp, p)
		lb.Ring.Value = LogRecord{cp, l}
		lb.Ring = lb.Ring.Next()
	}
	return len(p), nil
}

// LogRecord wraps JSON-like data from logger
type LogRecord struct {
	buf []byte
	lvl zerolog.Level
}

// MarshalJSON implements Marshaller interface
func (p LogRecord) MarshalJSON() ([]byte, error) { return p.buf, nil }

// Option defines logger option type
type Option func()

// SetLogger sets logger with options
func SetLogger(opts ...Option) {
	/* prevent writes */
	log.Logger = zerolog.Nop()
	/* reset to defaults */
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnixMs
	if logFile != nil {
		logFile.Close()
		logFile = nil
	}
	/* apply options */
	r0 := lastErrors.Ring
	for _, opt := range opts {
		opt()
	}
	r0.Do(func(p interface{}) {
		lastErrors.Ring.Value = p
		lastErrors.Ring = lastErrors.Ring.Next()
	})
	if logFile != nil {
		formatter.Out = zerolog.MultiLevelWriter(os.Stdout, logFile)
	}
	/* set logger */
	log.Logger = zerolog.New(filter).
		With().Timestamp().Caller().
		Logger()
}

// WithLastErrors sets count of buffered writes
func WithLastErrors(n int) Option {
	return func() {
		*lastErrors = LogBuffer{
			Level: zerolog.ErrorLevel,
			Ring:  ring.New(n),
		}
	}
}

// WithLevel sets level option
func WithLevel(l zerolog.Level) Option {
	return func() { zerolog.SetGlobalLevel(l) }
}

// WithLogFile sets filelog option
func WithLogFile(w io.WriteCloser) Option {
	return func() {
		if logFile != nil {
			logFile.Close()
		}
		logFile = w
	}
}

// WithTimeFormat sets time format option
func WithTimeFormat(s string) Option {
	return func() { formatter.TimeFormat = s }
}

// LastErrors returns last error writes
func LastErrors() []LogRecord {
	return lastErrors.Records()
}

// WriteLogBuffer writes buffered data to current logger
func WriteLogBuffer(lb *LogBuffer) {
	lvl := zerolog.GlobalLevel()
	lb.Ring.Do(func(p interface{}) {
		if p != nil {
			p := p.(LogRecord)
			if p.lvl >= lvl {
				filter.WriteLevel(p.lvl, p.buf)
			}
		}
	})
}
