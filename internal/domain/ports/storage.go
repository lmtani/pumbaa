package ports

import "context"

// FileProvider defines the interface for reading file contents.
// Implementations may support local files, cloud storage (GCS, S3), or other sources.
type FileProvider interface {
	// Read reads the content of a file from the specified path.
	// Supports local paths and cloud storage paths (e.g., gs://bucket/file).
	Read(ctx context.Context, path string) (string, error)
}
