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

// TestReconfigureCondense changes the condense duration between writes:
// the summary of condensed records must report the new one.
func TestReconfigureCondense(t *testing.T) {
	t.Cleanup(func() { NewLoggerWriter(WithCondense(0)) })
	path := filepath.Join(t.TempDir(), "log")
	lf := &LogFile{FilePath: path}
	logger := zerolog.New(NewLoggerWriter(
		WithCondense(time.Second), WithLevel(zerolog.InfoLevel), WithLogFile(lf))).With().Caller().Logger()
	logLine := func() { logger.Info().Msg("same caller") }
	for range 3 {
		logLine()
	}

	NewLoggerWriter(WithCondense(2*time.Second), WithLogFile(lf))
	time.Sleep(1100 * time.Millisecond)
	logLine()

	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(content, []byte("condensed 2 more entries last 2 seconds")) {
		t.Errorf("log file content = %q", content)
	}
}

// TestLogFileWriteAfterClose writes to a closed log file, as a write still in flight
// during reconfiguration does: the line must be kept without leaving the file open.
func TestLogFileWriteAfterClose(t *testing.T) {
	path := filepath.Join(t.TempDir(), "log")
	f := &LogFile{FilePath: path}
	if _, err := f.Write([]byte("before\n")); err != nil {
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := f.Write([]byte("after\n")); err != nil {
		t.Fatal(err)
	}
	if f.file != nil {
		t.Error("closed log file was reopened")
	}
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(content) != "before\nafter\n" {
		t.Errorf("log file content = %q", content)
	}
}
