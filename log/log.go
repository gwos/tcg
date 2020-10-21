package log

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/patrickmn/go-cache"
	"github.com/sirupsen/logrus"
)

// Fields Type to pass when we want to call WithFields for structured logging
type Fields map[string]interface{}

// Levels to pass when we want call Log on WithFields
const (
	ErrorLevel = logrus.ErrorLevel
	WarnLevel  = logrus.WarnLevel
	InfoLevel  = logrus.InfoLevel
	DebugLevel = logrus.DebugLevel
)

const timestampFormat = "2006-01-02 15:04:05"

var (
	logger = logrus.New()
	once   = sync.Once{}
	/* define regex matchers for sanitizing */
	sanReJSON = regexp.MustCompile(`((?i:password|token)"[^:]*:[^"]*)"(?:[^\\"]*(?:\\")*[\\]*)*"`)
	sanReMaps = regexp.MustCompile(`((?i:password|token):)(?:[^\s\]]*)`)
	/* define hooks */
	preFormatterHook = &genHook{
		levels:  logrus.AllLevels,
		handler: preformat,
	}
	writerHook = &multiWriterHook{
		cache:    cache.New(10*time.Minute, 10*time.Second),
		consolid: 0,
		file:     nil,
		once:     sync.Once{},
		writer:   os.Stdout,
	}
)

// Info makes entries in the log on Info level
func Info(args ...interface{}) {
	logger.Info(args...)
}

// Warn makes entries in the log on Warn level
func Warn(args ...interface{}) {
	logger.Warn(args...)
}

// Debug makes entries in the log on Debug level
func Debug(args ...interface{}) {
	logger.Debug(args...)
}

// Error makes entries in the log on Error level
func Error(args ...interface{}) {
	logger.Error(args...)
}

// Config configures logger
func Config(filePath string, level int, consolid time.Duration) {
	once.Do(func() {
		logger.AddHook(preFormatterHook)
		logger.AddHook(writerHook)
		logger.SetFormatter(&Formatter{
			TimestampFormat: timestampFormat,
		})
		logger.SetOutput(ioutil.Discard)
	})

	switch level {
	case 0:
		logger.SetLevel(logrus.ErrorLevel)
	case 1:
		logger.SetLevel(logrus.WarnLevel)
	case 2:
		logger.SetLevel(logrus.InfoLevel)
	default:
		logger.SetLevel(logrus.DebugLevel)
	}

	if writerHook.file != nil {
		writerHook.file.Close()
	}
	if len(filePath) > 0 {
		if file, err := os.OpenFile(filePath,
			os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644); err == nil {
			writerHook.writer = io.MultiWriter(os.Stdout, file)
			writerHook.file = file
		}
	}
	writerHook.consolid = consolid
}

// Entry wraps logrus.Entry
type Entry struct {
	*logrus.Entry
}

// With adds a struct of fields to the log entry
func With(fields Fields) *Entry {
	return &Entry{
		logger.WithFields(logrus.Fields(fields)),
	}
}

// WithDebug adds a struct of fields to the log entry
func (entry *Entry) WithDebug(fields Fields) *Entry {
	if logger.IsLevelEnabled(DebugLevel) {
		entry.Entry = entry.WithFields(logrus.Fields(fields))
	}
	return entry
}

// WithInfo adds a struct of fields to the log entry
func (entry *Entry) WithInfo(fields Fields) *Entry {
	if logger.IsLevelEnabled(InfoLevel) {
		entry.Entry = entry.WithFields(logrus.Fields(fields))
	}
	return entry
}

type multiWriterHook struct {
	cache    *cache.Cache
	consolid time.Duration
	file     *os.File
	once     sync.Once
	writer   io.Writer
}

func (h *multiWriterHook) onEvicted() func(string, interface{}) {
	return func(ck string, i interface{}) {
		v := i.(uint16)
		if v > 0 {
			_, _ = fmt.Fprintf(h.writer, "%s [consolidate: %d more entries last %.f seconds] %s\n",
				time.Now().Format(timestampFormat),
				v, h.consolid.Seconds(), ck)
		}
	}
}

// Levels implements logrus.Hook.Level interface
func (*multiWriterHook) Levels() []logrus.Level {
	return logrus.AllLevels
}

// Fire implements logrus.Hook.Fire interface
func (h *multiWriterHook) Fire(entry *logrus.Entry) error {
	h.once.Do(func() {
		h.cache.OnEvicted(h.onEvicted())
	})
	/* define cache key */
	var dataKeys []string
	for k := range entry.Data {
		dataKeys = append(dataKeys, k)
	}
	sort.Strings(dataKeys)
	ck := fmt.Sprintf("[%s] %s ", entry.Level, entry.Message)
	for _, k := range dataKeys {
		ck = fmt.Sprintf("%s#%s", ck, k)
	}
	/* workaround on https://github.com/patrickmn/go-cache/issues/48 */
	h.cache.DeleteExpired()
	/* inc hits if cached */
	if _, ok := h.cache.Get(ck); ok {
		_ = h.cache.Increment(ck, 1)
	} else {
		/* skip caching if consolidation off */
		if h.consolid.Milliseconds() > 0 {
			_ = h.cache.Add(ck, uint16(0), h.consolid)
		}
		output, _ := entry.Logger.Formatter.Format(entry)
		_, _ = h.writer.Write(output)
	}
	/* debug hits */
	// fmt.Println("\n##", ck, entry.Time, "\n##:", entry.Data)
	return nil
}

type genHook struct {
	handler func(Entry) error
	levels  []logrus.Level
}

// Levels implements logrus.Hook.Level interface
func (h *genHook) Levels() []logrus.Level {
	return h.levels
}

// Fire implements logrus.Hook.Fire interface
func (h *genHook) Fire(entry *logrus.Entry) error {
	return h.handler(Entry{entry})
}

// SetHook adds a hook to the logger hooks
func SetHook(handler func(Entry) error, levels ...logrus.Level) {
	if len(levels) == 0 {
		levels = logrus.AllLevels
	}
	logger.AddHook(&genHook{handler, levels})
}

// Formatter implements logrus.Formatter
type Formatter struct {
	TimestampFormat string
}

// Format implements logrus.Formatter.Format
func (f *Formatter) Format(entry *logrus.Entry) ([]byte, error) {
	var formattedFields interface{}
	if entry.Context != nil {
		formattedFields = entry.Context.Value(entry)
	}
	colors := []int{31, 31, 31, 33, 36, 37}
	buf := &bytes.Buffer{}
	fmt.Fprintf(buf, "%s \x1b[%dm[%s] %s%s\x1b[0m\n",
		entry.Time.Format(f.TimestampFormat),
		colors[entry.Level],
		strings.ToUpper(entry.Level.String()),
		formattedFields,
		entry.Message,
	)
	return buf.Bytes(), nil
}

func preformat(entry Entry) error {
	formattedData := ""
	for k, v := range entry.Data {
		s := fmt.Sprintf("%+v", v)
		if s == "" || s == "{}" || s == "[]" || s == "map[]" || s == "<nil>" {
			continue
		}
		/* sanitize attached json */
		s = sanReJSON.ReplaceAllString(s, `${1}"***"`)
		/* sanitize attached maps */
		s = sanReMaps.ReplaceAllString(s, `${1}***`)
		entry.Data[k] = s
		formattedData += fmt.Sprintf("[%s] ", s)
	}
	/* keep formatted data for reuse */
	ctx := entry.Context
	if ctx == nil {
		ctx = context.Background()
	}
	entry.Context = context.WithValue(ctx, entry.Entry, formattedData)
	return nil
}
