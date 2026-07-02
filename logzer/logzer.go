// Package logzer provides customized logger based on zerolog
package logzer

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
)

var (
	logFile   io.WriteCloser
	errBuffer = &LogBuffer{
		Level: zerolog.ErrorLevel,
		Size:  10,
	}
	formatter = &zerolog.ConsoleWriter{
		Out:        os.Stdout,
		NoColor:    true,
		TimeFormat: time.RFC3339,
	}
	filter = &FilterWriter{
		LevelWriter: zerolog.MultiLevelWriter(formatter, errBuffer),
		Re: map[*regexp.Regexp][]byte{
			// (1) Structured JSON: a key ending in password/token,
			// then its value (string | array | number | bool | null) -> "***".
			// The key is fully bracketed so a match can't span into the next field.
			regexp.MustCompile(`("[^"]*(?i:password|token)"\s*:\s*)` +
				`("(?:[^"\\]|\\.)*"|\[[^\]]*\]|true|false|null|-?\d[\d.eE+-]*)`): []byte(`${1}"***"`),
			// (2) Free text: the word password/token, a separator, then the value run.
			// The value run excludes whitespace and JSON structural chars (" ' , : { } [ ])
			// so masking prose can't corrupt a JSON line.
			// Note: an object-valued secret ("password":{...}) is not handled.
			regexp.MustCompile(`(?i)(password|token)([:=\s]+)([^\s"',:{}\[\]]+)`): []byte(`${1}${2}***`),
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
	if w.Condense <= 0 {
		/* condensing disabled: skip cache/regex work entirely */
		return w.LevelWriter.WriteLevel(lvl, p)
	}
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

func (w *CondenseWriter) onEvicted() func(string, any) {
	return func(ck string, i any) {
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
				return strconv.AppendInt(dst, ts.UnixMilli(), 10)
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

// Clear removes all collected records.
func (lb *LogBuffer) Clear() {
	lb.once.Do(func() { lb.ring = ring.New(lb.Size) })
	lb.mu.Lock()
	defer lb.mu.Unlock()
	lb.ring = ring.New(lb.Size)
}

// Resize reconfigures the buffer to keep up to n records, discarding any already collected.
// Safe for concurrent use; unlike assigning a new struct it does not copy the embedded lock.
func (lb *LogBuffer) Resize(n int) {
	lb.once.Do(func() {}) // consume lazy-init so it won't clobber the new ring
	lb.mu.Lock()
	defer lb.mu.Unlock()
	lb.Size = n
	lb.ring = ring.New(n)
}

// Records returns collected writes
func (lb *LogBuffer) Records() []LogRecord {
	lb.once.Do(func() { lb.ring = ring.New(lb.Size) })
	lb.mu.Lock()
	defer lb.mu.Unlock()

	rec := make([]LogRecord, 0, lb.Size)
	lb.ring.Do(func(p any) {
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
	lb.once.Do(func() { lb.ring = ring.New(lb.Size) })
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

// Option defines writer option type
type Option func()

// NewLoggerWriter returns writer
func NewLoggerWriter(opts ...Option) io.Writer {
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
		_, _ = errBuffer.WriteLevel(p.lvl, p.buf)
	}
	if logFile != nil {
		formatter.Out = zerolog.MultiLevelWriter(os.Stdout, logFile)
	}
	/* return writer */
	return condenser
}

// WithLastErrors sets count of buffered writes
func WithLastErrors(n int) Option {
	return func() { errBuffer.Resize(n) }
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

// WithColors sets formatter option
func WithColors(b bool) Option {
	return func() { formatter.NoColor = !b }
}

// WithTimeFormat sets formatter option
func WithTimeFormat(s string) Option {
	return func() { formatter.TimeFormat = s }
}

// IsDebugEnabled defines debugging
func IsDebugEnabled() bool { return zerolog.GlobalLevel() <= zerolog.DebugLevel }

// LastErrors returns last error writes
func LastErrors() []LogRecord {
	return errBuffer.Records()
}

// ClearLastErrors removes buffered error records.
func ClearLastErrors() {
	errBuffer.Clear()
}

// WriteLogBuffer writes buffered data to current logger
func WriteLogBuffer(lb *LogBuffer) {
	lvl := zerolog.GlobalLevel()
	for _, p := range lb.Records() {
		if p.lvl >= lvl {
			_, _ = filter.WriteLevel(p.lvl, p.buf)
		}
	}
}
