package log

import (
	nested "github.com/antonfisher/nested-logrus-formatter"
	"github.com/sirupsen/logrus"
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
func Config(level int) {
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
	case 3:
		logger.SetLevel(logrus.DebugLevel)
	}
}
