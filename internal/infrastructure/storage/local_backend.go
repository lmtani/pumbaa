package storage

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/lmtani/pumbaa/internal/domain/ports"
)

// LocalBackend implements StorageBackend for local filesystem access.
type LocalBackend struct{}

// NewLocalBackend creates a new LocalBackend.
func NewLocalBackend() *LocalBackend {
	return &LocalBackend{}
}

// CanHandle returns true for paths that are not cloud storage paths.
// LocalBackend acts as the fallback handler for non-prefixed paths.
func (l *LocalBackend) CanHandle(path string) bool {
	return !strings.HasPrefix(path, "gs://") &&
		!strings.HasPrefix(path, "s3://") &&
		!strings.HasPrefix(path, "az://")
}

// Read reads the content of a local file as a string.
// Enforces maxFileSize limit to prevent memory issues.
func (l *LocalBackend) Read(_ context.Context, path string) (string, error) {
	info, err := os.Stat(path)
	if err != nil {
		return "", fmt.Errorf("failed to stat file: %w", err)
	}

	if info.Size() > maxFileSize {
		return "", fmt.Errorf("file too large (%.2f MB > 1 MB limit)", float64(info.Size())/(1024*1024))
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("failed to read file: %w", err)
	}

	return string(data), nil
}

// ReadBytes reads the content of a local file as raw bytes.
// No size limit is enforced, suitable for binary files like ZIP dependencies.
func (l *LocalBackend) ReadBytes(_ context.Context, path string) ([]byte, error) {
	return os.ReadFile(path)
}

// Ensure LocalBackend implements StorageBackend at compile time.
var _ ports.StorageBackend = (*LocalBackend)(nil)
