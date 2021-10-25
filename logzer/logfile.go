package logzer

import (
	"fmt"
	"os"
	"sync"
)

// LogFile provides file rotation
type LogFile struct {
	mu       sync.Mutex
	file     *os.File
	fileSize int64

	FilePath string
	MaxSize  int64
	Rotate   int
}

// Close implements io.Closer interface
func (f *LogFile) Close() error {
	return f.file.Close()
}

// Write implements io.Writer interface
func (f *LogFile) Write(p []byte) (int, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	if f.file == nil {
		f.open()
	}
	if f.file != nil &&
		(f.MaxSize > 0 && f.MaxSize < f.fileSize+int64(len(p))) {
		f.rotate()
	}

	n, err := f.file.Write(p)
	if err != nil {
		f.open()
		n, err = f.file.Write(p)
	}
	if err == nil {
		f.fileSize += int64(n)
	}
	return n, err
}

func (f *LogFile) open() {
	if f.file != nil {
		_ = f.file.Close()
	}
	if file, err := os.OpenFile(f.FilePath,
		os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644); err == nil {
		f.file = file
		if fileInfo, err := file.Stat(); err == nil {
			f.fileSize = fileInfo.Size()
		}
	}
}

func (f *LogFile) rotate() {
	filename := f.file.Name()
	_ = f.file.Close()
	if f.Rotate == 0 {
		_ = os.Remove(filename)
	} else {
		for i := f.Rotate; i > 0; i-- {
			_ = os.Rename(fmt.Sprintf("%s.%d", filename, i-1), fmt.Sprintf("%s.%d", filename, i))
		}
		_ = os.Rename(filename, fmt.Sprintf("%s.%d", filename, 1))
	}
	f.open()
}
