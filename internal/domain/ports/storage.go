package ports

import "context"

// FileProvider defines the interface for reading file contents.
// Implementations may support local files, cloud storage (GCS, S3), or other sources.
type FileProvider interface {
	// Read reads the content of a file from the specified path as a string.
	// Supports local paths and cloud storage paths (e.g., gs://bucket/file).
	Read(ctx context.Context, path string) (string, error)

	// ReadBytes reads the content of a file from the specified path as raw bytes.
	// Useful for binary files like ZIP dependencies.
	ReadBytes(ctx context.Context, path string) ([]byte, error)

	// GetSize returns the size in bytes of a file without reading its content.
	// This is useful for calculating input sizes efficiently.
	GetSize(ctx context.Context, path string) (int64, error)
}

// FileSizeCache defines the interface for caching file sizes.
// Implementations may persist the cache to disk or other storage.
type FileSizeCache interface {
	// Load hydrates the cache from its persistent storage.
	Load() error

	// Save persists the cache to its storage.
	Save() error

	// Get returns the cached size for a path and whether it exists.
	Get(path string) (int64, bool)

	// Set caches the size for a path.
	Set(path string, size int64)
}

// StorageBackend defines the interface for individual storage backends.
// Each implementation handles a specific storage type (local, GCS, S3, etc.)
// This follows the Strategy Pattern, allowing new backends to be added
// without modifying existing code (Open/Closed Principle).
type StorageBackend interface {
	// CanHandle returns true if this backend can handle the given path.
	// Used by FileProvider to select the appropriate backend.
	CanHandle(path string) bool

	// Read reads the content of a file as a string with size limit enforcement.
	Read(ctx context.Context, path string) (string, error)

	// ReadBytes reads the content of a file as raw bytes without size limits.
	ReadBytes(ctx context.Context, path string) ([]byte, error)

	// GetSize returns the size in bytes of a file without reading its content.
	// Uses metadata API for cloud storage to avoid data transfer costs.
	GetSize(ctx context.Context, path string) (int64, error)
}
