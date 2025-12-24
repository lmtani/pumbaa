// Package gcs provides action handlers for Google Cloud Storage operations.
package gcs

import (
	"context"
	"fmt"
	"io"
	"strings"

	"cloud.google.com/go/storage"
	"github.com/lmtani/pumbaa/internal/infrastructure/agent/tools/types"
)

// MaxFileSize is the maximum file size that can be downloaded (5 MB).
const MaxFileSize int64 = 5 * 1024 * 1024

// DownloadHandler handles the "gcs_download" action to read files from GCS.
type DownloadHandler struct{}

// NewDownloadHandler creates a new DownloadHandler.
func NewDownloadHandler() *DownloadHandler {
	return &DownloadHandler{}
}

// Handle implements types.Handler.
func (h *DownloadHandler) Handle(ctx context.Context, input types.Input) (types.Output, error) {
	const action = "gcs_download"

	if input.Path == "" {
		return types.NewErrorOutput(action, "path is required (e.g., gs://bucket/file)"), nil
	}

	if !strings.HasPrefix(input.Path, "gs://") {
		return types.NewErrorOutput(action, "path must start with gs://"), nil
	}

	bucket, object, err := parsePath(input.Path)
	if err != nil {
		return types.NewErrorOutput(action, err.Error()), nil
	}

	client, err := storage.NewClient(ctx)
	if err != nil {
		return types.NewErrorOutput(action, fmt.Sprintf("failed to create GCS client: %v", err)), nil
	}
	defer client.Close()

	content, attrs, err := downloadObject(ctx, client, bucket, object)
	if err != nil {
		return types.NewErrorOutput(action, err.Error()), nil
	}

	return types.NewSuccessOutput(action, map[string]interface{}{
		"bucket":       bucket,
		"object":       object,
		"size":         attrs.Size,
		"content_type": attrs.ContentType,
		"content":      content,
	}), nil
}

// parsePath extracts bucket and object from a GCS path.
func parsePath(gcsPath string) (bucket, object string, err error) {
	path := strings.TrimPrefix(gcsPath, "gs://")
	parts := strings.SplitN(path, "/", 2)
	if len(parts) < 2 || parts[0] == "" || parts[1] == "" {
		return "", "", fmt.Errorf("invalid path format, expected gs://bucket/object")
	}
	return parts[0], parts[1], nil
}

// downloadObject downloads an object from GCS and returns its content.
func downloadObject(ctx context.Context, client *storage.Client, bucket, object string) (string, *storage.ObjectAttrs, error) {
	obj := client.Bucket(bucket).Object(object)

	attrs, err := obj.Attrs(ctx)
	if err != nil {
		if err == storage.ErrObjectNotExist {
			return "", nil, fmt.Errorf("object not found: gs://%s/%s", bucket, object)
		}
		return "", nil, fmt.Errorf("failed to get attrs: %v", err)
	}

	if attrs.Size > MaxFileSize {
		return "", nil, fmt.Errorf("file too large: %d bytes (max 5MB)", attrs.Size)
	}

	reader, err := obj.NewReader(ctx)
	if err != nil {
		return "", nil, fmt.Errorf("failed to read: %v", err)
	}
	defer reader.Close()

	content, err := io.ReadAll(reader)
	if err != nil {
		return "", nil, fmt.Errorf("failed to read content: %v", err)
	}

	return string(content), attrs, nil
}
