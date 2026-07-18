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

	// GetContentDigests returns the file's content fingerprints. Cloud backends
	// must read them from object metadata rather than downloading.
	//
	// Which digest matters depends on the Cromwell backend: local runs record
	// an MD5, while GCS records a crc32c. Both are returned in one call because
	// a single object-metadata read (or a single pass over a local file)
	// produces them together.
	GetContentDigests(ctx context.Context, path string) (FileDigests, error)
}

// FileDigests holds the content fingerprints of a file, in the exact encodings
// Cromwell records in call-caching hashes. A field is empty when the backend
// cannot supply it — a GCS composite object carries no MD5, for instance — and
// callers must then degrade to "cannot determine" rather than treating the
// absence as a difference.
type FileDigests struct {
	// MD5 is lowercase hex.
	MD5 string
	// CRC32C is the base64 of the 4-byte big-endian Castagnoli checksum, the
	// form GCS reports and Cromwell stores verbatim.
	CRC32C string
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

	// GetContentDigests returns the file's content fingerprints, or
	// ErrHashUnavailable when this backend cannot supply any.
	GetContentDigests(ctx context.Context, path string) (FileDigests, error)
}
