package log

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLogLevels(t *testing.T) {
	tests := []struct {
		name      string
		logFunc   func(*Logger, string, ...any)
		wantLevel string
	}{
		{"debug writes DEBUG", (*Logger).Debug, "DEBUG"},
		{"info writes INFO", (*Logger).Info, "INFO "},
		{"warn writes WARN", (*Logger).Warn, "WARN "},
		{"error writes ERROR", (*Logger).Error, "ERROR"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			path := filepath.Join(dir, "test.log")

			lg, err := New(path)
			require.NoError(t, err)

			tt.logFunc(lg, "hello", "key", "val")
			lg.Close()

			data, err := os.ReadFile(path)
			require.NoError(t, err)

			lines := strings.Split(strings.TrimSpace(string(data)), "\n")
			// First line is the startup line, second is our log line
			require.GreaterOrEqual(t, len(lines), 2)

			line := lines[len(lines)-1]
			assert.Contains(t, line, tt.wantLevel)
			assert.Contains(t, line, "hello")
			assert.Contains(t, line, "key=val")
		})
	}
}

func TestStartupLine(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.log")

	lg, err := New(path)
	require.NoError(t, err)
	lg.Close()

	data, err := os.ReadFile(path)
	require.NoError(t, err)

	line := strings.TrimSpace(string(data))
	assert.Contains(t, line, "INFO ")
	assert.Contains(t, line, "startup")
}

func TestCreatesParentDirectories(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "a", "b", "c", "test.log")

	lg, err := New(path)
	require.NoError(t, err)
	lg.Close()

	_, err = os.Stat(path)
	assert.NoError(t, err)
}

func TestLargeFileTruncation(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.log")

	// Create a file larger than 10MB
	f, err := os.Create(path)
	require.NoError(t, err)
	chunk := strings.Repeat("x", 1024)
	for i := 0; i < 11*1024; i++ { // ~11MB
		_, err := f.WriteString(chunk)
		require.NoError(t, err)
	}
	f.Close()

	info, err := os.Stat(path)
	require.NoError(t, err)
	require.Greater(t, info.Size(), int64(10*1024*1024))

	lg, err := New(path)
	require.NoError(t, err)
	lg.Close()

	info, err = os.Stat(path)
	require.NoError(t, err)
	assert.Less(t, info.Size(), int64(10*1024*1024))
}

func TestKeyValuePairs(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.log")

	lg, err := New(path)
	require.NoError(t, err)

	lg.Info("fetched", "service", "ec2", "count", 5)
	lg.Close()

	data, err := os.ReadFile(path)
	require.NoError(t, err)

	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	last := lines[len(lines)-1]
	assert.Contains(t, last, "service=ec2")
	assert.Contains(t, last, "count=5")
}
