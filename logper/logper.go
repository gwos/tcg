// Package logper provides logger wrapper preventing external dependencies in library
package logper

import (
	"bytes"
	"fmt"
	"io"
	stdlog "log"
)

// By default there is logging based on stdlog with disabled debug
var (
	// Error does struct logging with Error level
	Error,
	// Warn does struct logging with Warning level
	Warn,
	// Info does struct logging with Info level
	Info,
	// Debug does struct logging with Debug level
	Debug = func(flags int) (LogFn, LogFn, LogFn, LogFn) {
		loggerErr := stdlog.New(stdlog.Writer(), "ERR ", flags)
		loggerWrn := stdlog.New(stdlog.Writer(), "WRN ", flags)
		loggerInf := stdlog.New(stdlog.Writer(), "INF ", flags)
		loggerDbg := stdlog.New(stdlog.Writer(), "DBG ", flags)
		return func(fields interface{}, format string, a ...interface{}) {
				log2stdlog(loggerErr, fields, format, a...)
			},
			func(fields interface{}, format string, a ...interface{}) {
				log2stdlog(loggerWrn, fields, format, a...)
			},
			func(fields interface{}, format string, a ...interface{}) {
				log2stdlog(loggerInf, fields, format, a...)
			},
			func(fields interface{}, format string, a ...interface{}) {
				if IsDebugEnabled() {
					log2stdlog(loggerDbg, fields, format, a...)
				}
			}
	}(stdlog.Lmsgprefix | stdlog.Lshortfile | stdlog.Ldate | stdlog.Ltime | stdlog.LUTC)

	// IsDebugEnabled defines debugging
	IsDebugEnabled = func() bool { return false }
)

// LogFn defines log function
// fields arg can be of type: map[string]interface{}, []interface{},
//   interface{LogFields()(map[string]interface{}, map[string][]byte)}
type LogFn func(fields interface{}, format string, a ...interface{})

// SetLogger sets logging options in one call
func SetLogger(
	logError LogFn,
	logWarn LogFn,
	logInfo LogFn,
	logDebug LogFn,
	isDebugEnabled func() bool,
) {
	Error = logError
	Warn = logWarn
	Info = logInfo
	Debug = logDebug
	IsDebugEnabled = isDebugEnabled
}

func log2stdlog(logger *stdlog.Logger, fields interface{}, format string, a ...interface{}) {
	if logger == nil || logger.Writer() == io.Discard {
		return
	}
	buf := &bytes.Buffer{}
	if ff, ok := fields.(interface {
		LogFields() (map[string]interface{}, map[string][]byte)
	}); ok {
		m1, m2 := ff.LogFields()
		for k, v := range m1 {
			fmt.Fprintf(buf, "%s:%v ", k, v)
		}
		for k, v := range m2 {
			fmt.Fprintf(buf, "%s:%s ", k, v)
		}
	} else if ff, ok := fields.(map[string]interface{}); ok {
		for k, v := range ff {
			fmt.Fprintf(buf, "%s:%v ", k, v)
		}
	} else if ff, ok := fields.([]interface{}); ok {
		for _, v := range ff {
			fmt.Fprintf(buf, "%v ", v)
		}
	}
	if format != "" {
		fmt.Fprintf(buf, format, a...)
	}
	logger.Output(4, buf.String())
}
