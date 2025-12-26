package storage

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"

	"cloud.google.com/go/storage"

	"github.com/lmtani/pumbaa/internal/domain/ports"
)

// maxFileSize is the maximum file size we'll read (1 MB)
const maxFileSize = 1 * 1024 * 1024

// FileProvider implements ports.FileProvider to read from local or GCS storage.
type FileProvider struct{}

// NewFileProvider creates a new FileProvider.
func NewFileProvider() *FileProvider {
	return &FileProvider{}
}

// Read reads the content of a file from a local path or a GCS path.
func (f *FileProvider) Read(ctx context.Context, path string) (string, error) {
	if strings.HasPrefix(path, "gs://") {
		return f.readGCSFile(ctx, path)
	}
	return f.readLocalFile(path)
}

func (f *FileProvider) readGCSFile(ctx context.Context, path string) (string, error) {
	// Parse gs://bucket/object path
	cleanPath := strings.TrimPrefix(path, "gs://")
	parts := strings.SplitN(cleanPath, "/", 2)
	if len(parts) != 2 {
		return "", fmt.Errorf("invalid GCS path: %s", path)
	}
	bucket := parts[0]
	object := parts[1]

	client, err := storage.NewClient(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to create GCS client: %w", err)
	}
	defer client.Close()

	// Get object attributes to check size
	attrs, err := client.Bucket(bucket).Object(object).Attrs(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to get object attributes: %w", err)
	}

	if attrs.Size > maxFileSize {
		return "", fmt.Errorf("file too large (%.2f MB > 1 MB limit)", float64(attrs.Size)/(1024*1024))
	}

	// Read the object
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

func (f *FileProvider) readLocalFile(path string) (string, error) {
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

// ReadBytes reads the content of a file from a local path or a GCS path as raw bytes.
// This method does not enforce size limits, suitable for workflow dependencies.
func (f *FileProvider) ReadBytes(ctx context.Context, path string) ([]byte, error) {
	if strings.HasPrefix(path, "gs://") {
		return f.readGCSFileBytes(ctx, path)
	}
	return os.ReadFile(path)
}

func (f *FileProvider) readGCSFileBytes(ctx context.Context, path string) ([]byte, error) {
	// Parse gs://bucket/object path
	cleanPath := strings.TrimPrefix(path, "gs://")
	parts := strings.SplitN(cleanPath, "/", 2)
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid GCS path: %s", path)
	}
	bucket := parts[0]
	object := parts[1]

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

// Ensure FileProvider implements the domain interface
var _ ports.FileProvider = (*FileProvider)(nil)
