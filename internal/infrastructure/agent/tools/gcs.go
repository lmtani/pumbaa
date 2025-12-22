// Package tools provides implementations of tools for use with Google Agents ADK.
package tools

import (
	"context"
	"fmt"
	"io"
	"log"
	"strings"

	"cloud.google.com/go/storage"
	"google.golang.org/adk/tool"
	"google.golang.org/adk/tool/functiontool"
)

const (
	// MaxGCSFileSize defines the maximum allowed file size (5 MB)
	MaxGCSFileSize int64 = 5 * 1024 * 1024
)

// GCSDownloadInput defines the input parameters for the GCS download tool
type GCSDownloadInput struct {
	// Path is the full path of the object in GCS (e.g., gs://bucket/path/to/file.txt)
	Path string `json:"path"`
}

// GCSDownloadOutput represents the response of the GCS download
type GCSDownloadOutput struct {
	// Success indicates if the download was successful
	Success bool `json:"success"`
	// Error contains the error message, if any
	Error string `json:"error,omitempty"`
	// Content is the content of the downloaded file
	Content string `json:"content,omitempty"`
	// Bucket is the name of the bucket
	Bucket string `json:"bucket,omitempty"`
	// Object is the path of the object within the bucket
	Object string `json:"object,omitempty"`
	// Size is the size of the file in bytes
	Size int64 `json:"size,omitempty"`
	// ContentType is the MIME type of the file
	ContentType string `json:"content_type,omitempty"`
}

// parseGCSPath extracts bucket and object path from a gs:// path
func parseGCSPath(gcsPath string) (bucket, object string, err error) {
	if !strings.HasPrefix(gcsPath, "gs://") {
		return "", "", fmt.Errorf("invalid path: must start with 'gs://' (received: %s)", gcsPath)
	}

	// Remove gs:// prefix
	path := strings.TrimPrefix(gcsPath, "gs://")

	// Split bucket from object
	parts := strings.SplitN(path, "/", 2)
	if len(parts) < 2 || parts[0] == "" || parts[1] == "" {
		return "", "", fmt.Errorf("invalid path: expected format 'gs://bucket/path/to/object' (received: %s)", gcsPath)
	}
	return parts[0], parts[1], nil
}

// GetGCSDownload returns a tool that downloads objects from Google Cloud Storage.
// Requires authentication via Application Default Credentials (ADC).
// Files larger than 5 MB will be rejected.
func GetGCSDownload() tool.Tool {
	t, err := functiontool.New(
		functiontool.Config{
			Name:        "gcs_download",
			Description: "Downloads and reads the content of a file stored in Google Cloud Storage. Use this tool when the user provides a gs://bucket/path/to/file path. Limited to files smaller than 5 MB.",
		},
		func(ctx tool.Context, input GCSDownloadInput) (GCSDownloadOutput, error) {
			log.Printf("[GCS] Input received: path=%q", input.Path)

			if input.Path == "" {
				log.Printf("[GCS] Error: empty path")
				return GCSDownloadOutput{Success: false, Error: "parameter 'path' is required (e.g., gs://bucket/file.txt)"}, nil
			}

			// Validate and extract bucket/object from path
			bucket, object, err := parseGCSPath(input.Path)
			if err != nil {
				log.Printf("[GCS] Parse error: %v", err)
				return GCSDownloadOutput{Success: false, Error: err.Error()}, nil
			}
			log.Printf("[GCS] Bucket=%s, Object=%s", bucket, object)

			// Create GCS client
			client, err := storage.NewClient(context.Background())
			if err != nil {
				return GCSDownloadOutput{Success: false, Error: fmt.Sprintf("failed to create GCS client: %v", err)}, nil
			}
			defer client.Close()

			// Get object reference
			obj := client.Bucket(bucket).Object(object)

			// Get object attributes to check size
			attrs, err := obj.Attrs(context.Background())
			if err != nil {
				if err == storage.ErrObjectNotExist {
					return GCSDownloadOutput{Success: false, Error: fmt.Sprintf("object not found: %s", input.Path)}, nil
				}
				return GCSDownloadOutput{Success: false, Error: fmt.Sprintf("failed to get object attributes: %v", err)}, nil
			}

			// Check file size
			if attrs.Size > MaxGCSFileSize {
				return GCSDownloadOutput{
					Success: false,
					Error:   fmt.Sprintf("file too large: %d bytes (max allowed: %d bytes / 5 MB)", attrs.Size, MaxGCSFileSize),
				}, nil
			}

			// Download content
			reader, err := obj.NewReader(context.Background())
			if err != nil {
				return GCSDownloadOutput{Success: false, Error: fmt.Sprintf("failed to open object for reading: %v", err)}, nil
			}
			defer reader.Close()

			content, err := io.ReadAll(reader)
			if err != nil {
				return GCSDownloadOutput{Success: false, Error: fmt.Sprintf("failed to read object content: %v", err)}, nil
			}

			return GCSDownloadOutput{
				Success:     true,
				Content:     string(content),
				Bucket:      bucket,
				Object:      object,
				Size:        attrs.Size,
				ContentType: attrs.ContentType,
			}, nil
		},
	)
	if err != nil {
		panic(fmt.Sprintf("failed to create tool gcs_download: %v", err))
	}
	return t
}
