package storage

import (
	"context"
	"fmt"
	"io"
	"strings"

	"cloud.google.com/go/storage"

	"github.com/lmtani/pumbaa/internal/domain/ports"
)

// GCSBackend implements StorageBackend for Google Cloud Storage access.
type GCSBackend struct{}

// NewGCSBackend creates a new GCSBackend.
func NewGCSBackend() *GCSBackend {
	return &GCSBackend{}
}

// CanHandle returns true for paths starting with "gs://".
func (g *GCSBackend) CanHandle(path string) bool {
	return strings.HasPrefix(path, "gs://")
}

// Read reads the content of a GCS object as a string.
// Enforces maxFileSize limit to prevent memory issues.
func (g *GCSBackend) Read(ctx context.Context, path string) (string, error) {
	bucket, object, err := g.parsePath(path)
	if err != nil {
		return "", err
	}

	client, err := storage.NewClient(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to create GCS client: %w", err)
	}
	defer client.Close()

	// Check object size before reading
	attrs, err := client.Bucket(bucket).Object(object).Attrs(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to get object attributes: %w", err)
	}

	if attrs.Size > maxFileSize {
		return "", fmt.Errorf("file too large (%.2f MB > 1 MB limit)", float64(attrs.Size)/(1024*1024))
	}

	rc, err := client.Bucket(bucket).Object(object).NewReader(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to open GCS object: %w", err)
	}
	defer rc.Close()

	data, err := io.ReadAll(rc)
	if err != nil {
		return "", fmt.Errorf("failed to read GCS object: %w", err)
	}

	return string(data), nil
}

// ReadBytes reads the content of a GCS object as raw bytes.
// No size limit is enforced, suitable for binary files like ZIP dependencies.
func (g *GCSBackend) ReadBytes(ctx context.Context, path string) ([]byte, error) {
	bucket, object, err := g.parsePath(path)
	if err != nil {
		return nil, err
	}

	client, err := storage.NewClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCS client: %w", err)
	}
	defer client.Close()

	rc, err := client.Bucket(bucket).Object(object).NewReader(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to open GCS object: %w", err)
	}
	defer rc.Close()

	return io.ReadAll(rc)
}

// parsePath extracts bucket and object from a gs:// path.
func (g *GCSBackend) parsePath(path string) (bucket, object string, err error) {
	cleanPath := strings.TrimPrefix(path, "gs://")
	parts := strings.SplitN(cleanPath, "/", 2)
	if len(parts) != 2 {
		return "", "", fmt.Errorf("invalid GCS path: %s", path)
	}
	return parts[0], parts[1], nil
}

// Ensure GCSBackend implements StorageBackend at compile time.
var _ ports.StorageBackend = (*GCSBackend)(nil)
