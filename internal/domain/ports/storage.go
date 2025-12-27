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
}
