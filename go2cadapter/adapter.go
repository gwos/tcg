package go2cadapter

import (
	"time"

	"github.com/gwos/tcg/logger"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

var logBuf interface{}

func SetLogBuffer() {
	logBuf = &logger.LogBuffer{
		Level: zerolog.TraceLevel,
		Size:  16,
	}
	log.Logger = zerolog.New(logBuf.(*logger.LogBuffer)).
		With().Timestamp().Caller().Logger()
}

func UseLogBuffer() {
	logger.WriteLogBuffer(logBuf.(*logger.LogBuffer))
}

func SetLoggerWithOptions(
	condense time.Duration,
	filePath string,
	fileMaxSize int64,
	fileRotate int,
	level int,
	noColor bool,
	timeFormat string,
) {
	opts := []logger.Option{
		logger.WithCondense(condense),
		logger.WithLastErrors(10),
		logger.WithLevel([...]zerolog.Level{3, 2, 1, 0}[level]),
		logger.WithNoColor(noColor),
		logger.WithTimeFormat(timeFormat),
	}
	if filePath != "" {
		opts = append(opts, logger.WithLogFile(&logger.LogFile{
			FilePath: filePath,
			MaxSize:  fileMaxSize,
			Rotate:   fileRotate,
		}))
	}
	logger.SetLogger(opts...)
}
