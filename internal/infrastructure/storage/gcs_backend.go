package storage

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"strings"
	"sync"

	"cloud.google.com/go/storage"

	"github.com/lmtani/pumbaa/internal/application/ports"
)

// GCSBackend implements StorageBackend for Google Cloud Storage access.
//
// The client is built once and shared. Constructing one per request costs a
// credential lookup each time, which dominates any workload that touches many
// objects — a metadata sweep over a few hundred inputs spent almost all of its
// wall clock there.
type GCSBackend struct {
	once   sync.Once
	client *storage.Client
	err    error
}

// NewGCSBackend creates a new GCSBackend.
func NewGCSBackend() *GCSBackend {
	return &GCSBackend{}
}

// clientFor returns the shared client, building it on first use so that a
// process which never touches cloud storage never pays for credentials.
func (g *GCSBackend) clientFor(ctx context.Context) (*storage.Client, error) {
	g.once.Do(func() {
		g.client, g.err = storage.NewClient(ctx)
	})
	if g.err != nil {
		return nil, fmt.Errorf("failed to create GCS client: %w", g.err)
	}
	return g.client, nil
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

	client, err := g.clientFor(ctx)
	if err != nil {
		return "", err
	}

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
	defer func() { _ = rc.Close() }()

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

	client, err := g.clientFor(ctx)
	if err != nil {
		return nil, err
	}

	rc, err := client.Bucket(bucket).Object(object).NewReader(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to open GCS object: %w", err)
	}
	defer func() { _ = rc.Close() }()

	return io.ReadAll(rc)
}

// GetSize returns the size of a GCS object without reading its content.
// Uses the Attrs API to fetch metadata only, avoiding data transfer costs.
func (g *GCSBackend) GetSize(ctx context.Context, path string) (int64, error) {
	bucket, object, err := g.parsePath(path)
	if err != nil {
		return 0, err
	}

	client, err := g.clientFor(ctx)
	if err != nil {
		return 0, err
	}

	attrs, err := client.Bucket(bucket).Object(object).Attrs(ctx)
	if err != nil {
		if errors.Is(err, storage.ErrObjectNotExist) {
			return 0, fmt.Errorf("%w: %s", ports.ErrFileNotFound, path)
		}
		return 0, fmt.Errorf("failed to get object attributes: %w", err)
	}

	return attrs.Size, nil
}

// GetContentDigests returns a GCS object's checksums, read from object
// metadata so no data is transferred.
//
// Cromwell records the crc32c for GCS inputs — verified against a real
// GoogleBatch run — so that is the field that matters there; the MD5 is
// returned too when present. Composite objects, produced by parallel or
// resumable uploads, carry no MD5 at all, which is exactly why the crc32c is
// the one to rely on.
func (g *GCSBackend) GetContentDigests(ctx context.Context, path string) (ports.FileDigests, error) {
	bucket, object, err := g.parsePath(path)
	if err != nil {
		return ports.FileDigests{}, err
	}

	client, err := g.clientFor(ctx)
	if err != nil {
		return ports.FileDigests{}, err
	}

	attrs, err := client.Bucket(bucket).Object(object).Attrs(ctx)
	if err != nil {
		if errors.Is(err, storage.ErrObjectNotExist) {
			return ports.FileDigests{}, fmt.Errorf("%w: %s", ports.ErrFileNotFound, path)
		}
		return ports.FileDigests{}, fmt.Errorf("failed to get object attributes: %w", err)
	}

	digests := ports.FileDigests{CRC32C: encodeCRC32C(attrs.CRC32C)}
	if len(attrs.MD5) > 0 {
		digests.MD5 = hex.EncodeToString(attrs.MD5)
	}
	return digests, nil
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
