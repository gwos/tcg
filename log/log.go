package log

import (
	"fmt"
	nested "github.com/antonfisher/nested-logrus-formatter"
	"github.com/patrickmn/go-cache"
	"github.com/sirupsen/logrus"
	"io"
	"io/ioutil"
	"os"
	"sort"
	"sync"
	"time"
)

// Fields Type to pass when we want to call WithFields for structured logging
type Fields map[string]interface{}

var once sync.Once

// Levels to pass when we want call Log on WithFields
const (
	ErrorLevel = logrus.ErrorLevel
	WarnLevel  = logrus.WarnLevel
	InfoLevel  = logrus.InfoLevel
	DebugLevel = logrus.DebugLevel
)

const timestampFormat = "2006-01-02 15:04:05"

var logger = logrus.New()

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
func Config(filePath string, level int) {
	once.Do(func() {
		ch := ckHook{
			cache.New(10*time.Minute, 10*time.Second),
			os.Stdout,
		}

		if len(filePath) > 0 {
			if logFile, err := os.OpenFile(filePath,
				os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644); err == nil {
				ch.writer = io.MultiWriter(os.Stdout, logFile)
			}
		}

		ch.cache.OnEvicted(fnOnEvicted(ch.writer))
		logger.SetOutput(ioutil.Discard)
		logger.AddHook(&ch)
	})

	logger.SetFormatter(&nested.Formatter{
		TimestampFormat: timestampFormat,
		HideKeys:        true,
		ShowFullLevel:   true,
		FieldsOrder:     []string{"component", "category"},
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
		entry = &Entry{
			entry.WithFields(logrus.Fields(fields)),
		}
	}
	return entry
}

// WithInfo adds a struct of fields to the log entry
func (entry *Entry) WithInfo(fields Fields) *Entry {
	if logger.IsLevelEnabled(InfoLevel) {
		entry = &Entry{
			entry.WithFields(logrus.Fields(fields)),
		}
	}
	return entry
}

func fnOnEvicted(w io.Writer) func(string, interface{}) {
	return func(ck string, i interface{}) {
		v := i.(uint16)
		if v > 0 {
			fmt.Fprintf(w, "%s [consolidate: %d more entries last 60 seconds] %s\n",
				time.Now().Format(timestampFormat),
				v, ck)
		}
	}
}

type ckHook struct {
	cache  *cache.Cache
	writer io.Writer
}

// Levels implements logrus.Hook.Level interface
func (*ckHook) Levels() []logrus.Level {
	return logrus.AllLevels
}

// Fire implements logrus.Hook.Fire interface
func (h *ckHook) Fire(entry *logrus.Entry) error {
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
		h.cache.Increment(ck, 1)
	} else {
		h.cache.Add(ck, uint16(0), 60*time.Second)
		output, _ := entry.Logger.Formatter.Format(entry)
		h.writer.Write(output)
	}
	/* debug hits */
	// fmt.Println("\n##", ck, entry.Time, "\n##:", entry.Data)
	return nil
}
