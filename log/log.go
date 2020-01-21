package log

import (
	nested "github.com/antonfisher/nested-logrus-formatter"
	"github.com/sirupsen/logrus"
	"io"
	"os"
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
	if len(filePath) > 0 {
		if logFile, err := os.OpenFile(filePath,
			os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644); err == nil {
			logger.SetOutput(io.MultiWriter(os.Stdout, logFile))
		}
	}

	logger.SetFormatter(&nested.Formatter{
		TimestampFormat: "2006-01-02 15:04:05",
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
