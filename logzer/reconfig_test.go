package logzer

import (
	"bytes"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/rs/zerolog"
)

// TestReconfigureWhileLogging reconfigures the writer, as a config reload does,
// while other goroutines keep logging: no line may be lost and the race detector must stay quiet.
func TestReconfigureWhileLogging(t *testing.T) {
	path := filepath.Join(t.TempDir(), "log")
	opts := func() []Option {
		return []Option{
			WithCondense(0),
			WithColors(false),
			WithTimeFormat(time.RFC3339),
			WithLastErrors(10),
			WithLevel(zerolog.InfoLevel),
			WithLogFile(&LogFile{FilePath: path}),
		}
	}
	logger := zerolog.New(NewLoggerWriter(opts()...))

	const writers, lines = 4, 200
	var wg sync.WaitGroup
	for range writers {
		wg.Go(func() {
			for range lines {
				logger.Info().Msg("line")
			}
		})
	}
	done := make(chan struct{})
	go func() {
		defer close(done)
		for range 50 {
			NewLoggerWriter(opts()...)
		}
	}()
	wg.Wait()
	<-done

	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if n := bytes.Count(content, []byte("\n")); n != writers*lines {
		t.Errorf("logged lines = %d; want %d", n, writers*lines)
	}
}

// TestReconfigureWithoutLogFile drops the log file on reconfiguration, as a reload that clears it does:
// later lines must not reach the old file.
func TestReconfigureWithoutLogFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "log")
	logger := zerolog.New(NewLoggerWriter(WithLevel(zerolog.InfoLevel), WithLogFile(&LogFile{FilePath: path})))
	logger.Info().Msg("to file")

	logger = zerolog.New(NewLoggerWriter(WithLevel(zerolog.InfoLevel)))
	logger.Info().Msg("to stdout only")

	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(content, []byte("stdout only")) {
		t.Errorf("log file content = %q", content)
	}
}
