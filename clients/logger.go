package clients

import (
	"bytes"
	"fmt"
	"io"
	stdlog "log"
	"os"
)

/* by default there is logging based on stdlog with disabled debug */
var (
	flags       = stdlog.Lmsgprefix | stdlog.Lshortfile | stdlog.LUTC
	stdlogError = stdlog.New(os.Stderr, "ERR:", flags)
	stdlogWarn  = stdlog.New(os.Stderr, "WRN:", flags)
	stdlogInfo  = stdlog.New(os.Stderr, "INF:", flags)
	stdlogDebug = stdlog.New(os.Stderr, "DBG:", flags)

	// LogError do struct logging with Error level
	LogError StructLogFn = func(fields interface{}, format string, v ...interface{}) {
		log2stdlog(stdlogError, fields, format, v...)
	}
	// LogWarn do struct logging with Warning level
	LogWarn StructLogFn = func(fields interface{}, format string, v ...interface{}) {
		log2stdlog(stdlogWarn, fields, format, v...)
	}
	// LogInfo do struct logging with Info level
	LogInfo StructLogFn = func(fields interface{}, format string, v ...interface{}) {
		log2stdlog(stdlogInfo, fields, format, v...)
	}
	// LogDebug do struct logging with Debug level
	LogDebug StructLogFn = func(fields interface{}, format string, v ...interface{}) {
		if IsDebugEnabled() {
			log2stdlog(stdlogDebug, fields, format, v...)
		}
	}
	// IsDebugEnabled defines debugging
	IsDebugEnabled = func() bool { return false }
)

// StructLogFn defines log function
// fields arg can be of type: map[string]interface{}, []interface{},
//   interface{LogFields()(map[string]interface{}, map[string][]byte)}
type StructLogFn func(fields interface{}, format string, v ...interface{})

// SetLogger sets logging options in one call
func SetLogger(
	logError StructLogFn,
	logWarn StructLogFn,
	logInfo StructLogFn,
	logDebug StructLogFn,
	isDebugEnabled func() bool,
) {
	LogError = logError
	LogWarn = logWarn
	LogInfo = logInfo
	LogDebug = logDebug
	IsDebugEnabled = isDebugEnabled
}

func log2stdlog(logger *stdlog.Logger, fields interface{}, format string, v ...interface{}) {
	if logger == nil || logger.Writer() == io.Discard {
		return
	}
	buf := &bytes.Buffer{}
	if ff, ok := fields.(interface {
		LogFields() (map[string]interface{}, map[string][]byte)
	}); ok {
		m1, m2 := ff.LogFields()
		for k, v := range m1 {
			fmt.Fprintf(buf, `%s:%s `, k, v)
		}
		for k, v := range m2 {
			fmt.Fprintf(buf, `%s:%s `, k, v)
		}
	} else if ff, ok := fields.(map[string]interface{}); ok {
		for k, v := range ff {
			fmt.Fprintf(buf, `%s:%s `, k, v)
		}
	} else if ff, ok := fields.([]interface{}); ok {
		for _, v := range ff {
			fmt.Fprintf(buf, `%s `, v)
		}
	}
	if format != "" {
		fmt.Fprintf(buf, format, v...)
	}
	logger.Output(2, buf.String())
}

// obj defines a short alias
type obj map[string]interface{}
