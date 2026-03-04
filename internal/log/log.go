package log

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

const maxFileSize = 10 * 1024 * 1024 // 10MB

// Logger writes structured log lines to a file.
type Logger struct {
	mu   sync.Mutex
	file *os.File
}

// New creates a Logger that appends to the file at path.
// It creates parent directories if needed and truncates the file if it exceeds 10MB.
func New(path string) (*Logger, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, fmt.Errorf("create log dir: %w", err)
	}

	// Truncate if file is too large.
	if info, err := os.Stat(path); err == nil && info.Size() > maxFileSize {
		if err := os.Truncate(path, 0); err != nil {
			return nil, fmt.Errorf("truncate log file: %w", err)
		}
	}

	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return nil, fmt.Errorf("open log file: %w", err)
	}

	l := &Logger{file: f}
	l.Info("startup")
	return l, nil
}

// Debug logs a message at DEBUG level.
func (l *Logger) Debug(msg string, kvs ...any) {
	l.log("DEBUG", msg, kvs...)
}

// Info logs a message at INFO level.
func (l *Logger) Info(msg string, kvs ...any) {
	l.log("INFO ", msg, kvs...)
}

// Warn logs a message at WARN level.
func (l *Logger) Warn(msg string, kvs ...any) {
	l.log("WARN ", msg, kvs...)
}

// Error logs a message at ERROR level.
func (l *Logger) Error(msg string, kvs ...any) {
	l.log("ERROR", msg, kvs...)
}

// Close closes the underlying log file.
func (l *Logger) Close() error {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.file.Close()
}

func (l *Logger) log(level, msg string, kvs ...any) {
	l.mu.Lock()
	defer l.mu.Unlock()

	ts := time.Now().UTC().Format(time.RFC3339Nano)

	var line string
	if len(kvs) > 0 {
		pairs := formatKVs(kvs)
		line = fmt.Sprintf("%s %s %s %s\n", ts, level, msg, pairs)
	} else {
		line = fmt.Sprintf("%s %s %s\n", ts, level, msg)
	}

	_, _ = l.file.WriteString(line)
}

func formatKVs(kvs []any) string {
	var b []byte
	for i := 0; i+1 < len(kvs); i += 2 {
		if len(b) > 0 {
			b = append(b, ' ')
		}
		b = append(b, fmt.Sprintf("%v=%v", kvs[i], kvs[i+1])...)
	}
	return string(b)
}
