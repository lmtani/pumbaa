package storage

import (
	"context"
	"fmt"

	"github.com/lmtani/pumbaa/internal/domain/ports"
)

// FileProvider implements ports.FileProvider using a registry of storage backends.
// It delegates read operations to the appropriate backend based on the path prefix.
type FileProvider struct {
	backends []ports.StorageBackend
}

// NewFileProvider creates a FileProvider with default backends (GCS and Local).
// The order matters: backends are checked in order, with Local as the fallback.
func NewFileProvider() *FileProvider {
	return &FileProvider{
		backends: []ports.StorageBackend{
			NewGCSBackend(),
			NewLocalBackend(), // Local is the fallback (last)
		},
	}
}

// NewFileProviderWithBackends creates a FileProvider with custom backends.
// Use this to inject mock backends for testing or to add new storage types.
// Backends are checked in order; place more specific backends first.
func NewFileProviderWithBackends(backends ...ports.StorageBackend) *FileProvider {
	return &FileProvider{backends: backends}
}

// Read reads the content of a file by delegating to the appropriate backend.
func (f *FileProvider) Read(ctx context.Context, path string) (string, error) {
	for _, backend := range f.backends {
		if backend.CanHandle(path) {
			return backend.Read(ctx, path)
		}
	}
	return "", fmt.Errorf("no storage backend found for path: %s", path)
}

// ReadBytes reads the content of a file as bytes by delegating to the appropriate backend.
func (f *FileProvider) ReadBytes(ctx context.Context, path string) ([]byte, error) {
	for _, backend := range f.backends {
		if backend.CanHandle(path) {
			return backend.ReadBytes(ctx, path)
		}
	}
	return nil, fmt.Errorf("no storage backend found for path: %s", path)
}

// Ensure FileProvider implements the domain interface at compile time.
var _ ports.FileProvider = (*FileProvider)(nil)
