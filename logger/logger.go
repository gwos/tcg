package logger

import (
	"container/ring"
	"io"
	"os"
	"regexp"
	"strconv"
	"sync"
	"time"

	"github.com/patrickmn/go-cache"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

var (
	logFile   io.WriteCloser
	errBuffer = &LogBuffer{
		Level: zerolog.ErrorLevel,
		Size:  10,
	}
	formatter = &zerolog.ConsoleWriter{
		Out:        os.Stdout,
		NoColor:    false,
		TimeFormat: time.RFC3339,
	}
	filter = &FilterWriter{
		LevelWriter: zerolog.MultiLevelWriter(formatter, errBuffer),
		Re: map[*regexp.Regexp][]byte{
			regexp.MustCompile(`((?i:password|token)"[^:]*:[^"]*)"(?:[^\\"]*(?:\\")*[\\]*)*"`): []byte(`${1}"***"`),
		},
	}
	condenser = &CondenseWriter{
		Condense:    0,
		LevelWriter: filter,
	}
)

// CondenseWriter handles similar writes by caller field
type CondenseWriter struct {
	zerolog.LevelWriter
	mu       sync.Mutex
	once     sync.Once
	cache    *cache.Cache
	callerRe *regexp.Regexp
	Condense time.Duration
}

// Write implements io.Writer interface
func (w *CondenseWriter) Write(p []byte) (int, error) {
	return w.WriteLevel(zerolog.NoLevel, p)
}

// WriteLevel implements zerolog.LevelWriter interface
func (w *CondenseWriter) WriteLevel(lvl zerolog.Level, p []byte) (int, error) {
	w.once.Do(func() {
		defaultExpiration, cleanupInterval := time.Minute*10, time.Second*10
		if w.Condense > 0 {
			defaultExpiration = w.Condense * 2
			cleanupInterval = w.Condense / 4
		}
		w.cache = cache.New(defaultExpiration, cleanupInterval)
		w.cache.OnEvicted(w.onEvicted())
		w.callerRe = regexp.MustCompile(`"` + zerolog.CallerFieldName + `":"[^"]*"`)
	})
	w.mu.Lock()
	defer w.mu.Unlock()

	/* define cache key */
	ck := string(append([]byte{byte(lvl), ':'}, w.callerRe.Find(p)...))
	/* workaround on https://github.com/patrickmn/go-cache/issues/48 */
	w.cache.DeleteExpired()
	/* inc hits if cached */
	if _, ok := w.cache.Get(ck); ok {
		_ = w.cache.Increment(ck, 1)
		return len(p), nil
	}
	/* skip caching if not condense */
	if w.Condense > 0 {
		_ = w.cache.Add(ck, uint16(0), w.Condense)
	}
	return w.LevelWriter.WriteLevel(lvl, p)
}

func (w *CondenseWriter) onEvicted() func(string, interface{}) {
	return func(ck string, i interface{}) {
		appendLvl := func(dst []byte, lvl zerolog.Level) []byte {
			dst = append(dst, '"')
			dst = append(dst, zerolog.LevelFieldName...)
			dst = append(dst, `":"`...)
			dst = append(dst, lvl.String()...)
			return append(dst, '"')
		}
		appendTS := func(dst []byte, ts time.Time) []byte {
			dst = append(dst, '"')
			dst = append(dst, zerolog.TimestampFieldName...)
			dst = append(dst, `":`...)
			switch zerolog.TimeFieldFormat {
			case zerolog.TimeFormatUnix:
				return strconv.AppendInt(dst, ts.Unix(), 10)
			case zerolog.TimeFormatUnixMs:
				return strconv.AppendInt(dst, ts.UnixNano()/1000000, 10)
			case zerolog.TimeFormatUnixMicro:
				return strconv.AppendInt(dst, ts.UnixNano()/1000, 10)
			}
			dst = append(dst, '"')
			dst = ts.AppendFormat(dst, zerolog.TimeFieldFormat)
			return append(dst, '"')
		}

		v := i.(uint16)
		if v > 0 {
			lvl, caller := zerolog.Level(ck[0]), ck[2:]
			buf := append(make([]byte, 0, 200), '{')
			buf = appendLvl(buf, lvl)
			buf = append(buf, ',')
			buf = appendTS(buf, time.Now())
			buf = append(buf, ',')
			buf = append(buf, caller...)
			buf = append(buf, `,"`...)
			buf = append(buf, zerolog.MessageFieldName...)
			buf = append(buf, `":"[condensed `...)
			buf = strconv.AppendInt(buf, int64(v), 10)
			buf = append(buf, ` more entries last `...)
			buf = strconv.AppendInt(buf, int64(w.Condense.Seconds()), 10)
			buf = append(buf, ` seconds]"}`...)
			_, _ = w.LevelWriter.WriteLevel(lvl, buf)
		}
	}
}

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
func (w *FilterWriter) WriteLevel(lvl zerolog.Level, p []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	for reg, repl := range w.Re {
		p = reg.ReplaceAll(p, repl)
	}
	return w.LevelWriter.WriteLevel(lvl, p)
}

// LogBuffer collects writes if level passed
type LogBuffer struct {
	mu    sync.Mutex
	once  sync.Once
	ring  *ring.Ring
	Level zerolog.Level
	Size  int
}

// Records returns collected writes
func (lb *LogBuffer) Records() []LogRecord {
	lb.once.Do(func() {
		lb.ring = ring.New(lb.Size)
	})
	lb.mu.Lock()
	defer lb.mu.Unlock()
	rec := []LogRecord{}
	lb.ring.Do(func(p interface{}) {
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
func (lb *LogBuffer) WriteLevel(lvl zerolog.Level, p []byte) (int, error) {
	lb.once.Do(func() {
		lb.ring = ring.New(lb.Size)
	})
	lb.mu.Lock()
	defer lb.mu.Unlock()
	if lvl >= lb.Level {
		/* store the copy as source could be updated */
		cp := make([]byte, len(p))
		copy(cp, p)
		lb.ring.Value = LogRecord{cp, lvl}
		lb.ring = lb.ring.Next()
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
	lastErrors := LastErrors()
	for _, opt := range opts {
		opt()
	}
	for _, p := range lastErrors {
		errBuffer.WriteLevel(p.lvl, p.buf)
	}
	if logFile != nil {
		formatter.Out = zerolog.MultiLevelWriter(os.Stdout, logFile)
	}
	/* set logger */
	log.Logger = zerolog.New(condenser).
		With().Timestamp().Caller().
		Logger()
}

// WithLastErrors sets count of buffered writes
func WithLastErrors(n int) Option {
	return func() {
		*errBuffer = LogBuffer{
			Level: zerolog.ErrorLevel,
			Size:  n,
		}
	}
}

// WithLevel sets level option
func WithLevel(lvl zerolog.Level) Option {
	return func() { zerolog.SetGlobalLevel(lvl) }
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

// WithCondense enables condensing similar records
func WithCondense(d time.Duration) Option {
	return func() { condenser.Condense = d }
}

// WithNoColor sets formatter option
func WithNoColor(b bool) Option {
	return func() { formatter.NoColor = b }
}

// WithTimeFormat sets formatter option
func WithTimeFormat(s string) Option {
	return func() { formatter.TimeFormat = s }
}

// LastErrors returns last error writes
func LastErrors() []LogRecord {
	return errBuffer.Records()
}

// WriteLogBuffer writes buffered data to current logger
func WriteLogBuffer(lb *LogBuffer) {
	lvl := zerolog.GlobalLevel()
	for _, p := range lb.Records() {
		if p.lvl >= lvl {
			filter.WriteLevel(p.lvl, p.buf)
		}
	}
}
