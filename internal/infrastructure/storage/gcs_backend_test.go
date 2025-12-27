package storage

import (
	"context"
	"strings"
	"testing"
)

func TestGCSBackend_CanHandle(t *testing.T) {
	backend := NewGCSBackend()

	tests := []struct {
		name     string
		path     string
		expected bool
	}{
		{"GCS path", "gs://bucket/file.txt", true},
		{"GCS path with nested object", "gs://bucket/folder/file.txt", true},
		{"local path", "/home/user/file.txt", false},
		{"S3 path", "s3://bucket/file.txt", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := backend.CanHandle(tt.path); got != tt.expected {
				t.Errorf("CanHandle(%q) = %v, want %v", tt.path, got, tt.expected)
			}
		})
	}
}

func TestGCSBackend_ParsePath(t *testing.T) {
	backend := NewGCSBackend()

	t.Run("valid path", func(t *testing.T) {
		bucket, object, err := backend.parsePath("gs://my-bucket/path/to/file.txt")
		if err != nil {
			t.Errorf("parsePath() unexpected error: %v", err)
		}
		if bucket != "my-bucket" {
			t.Errorf("bucket = %q, want %q", bucket, "my-bucket")
		}
		if object != "path/to/file.txt" {
			t.Errorf("object = %q, want %q", object, "path/to/file.txt")
		}
	})

	t.Run("invalid path - no object", func(t *testing.T) {
		_, _, err := backend.parsePath("gs://bucket-only")
		if err == nil {
			t.Error("parsePath() expected error for invalid path, got nil")
		}
		if !strings.Contains(err.Error(), "invalid GCS path") {
			t.Errorf("expected 'invalid GCS path' error, got: %v", err)
		}
	})
}

func TestGCSBackend_Read_ValidationErrors(t *testing.T) {
	backend := NewGCSBackend()
	ctx := context.Background()

	t.Run("invalid GCS path", func(t *testing.T) {
		_, err := backend.Read(ctx, "gs://invalid-path")
		if err == nil {
			t.Error("Read() expected error for invalid GCS path, got nil")
		}
		if !strings.Contains(err.Error(), "invalid GCS path") {
			t.Errorf("expected 'invalid GCS path' error, got: %v", err)
		}
	})
}

func TestGCSBackend_ReadBytes_ValidationErrors(t *testing.T) {
	backend := NewGCSBackend()
	ctx := context.Background()

	t.Run("invalid GCS path", func(t *testing.T) {
		_, err := backend.ReadBytes(ctx, "gs://invalid-path")
		if err == nil {
			t.Error("ReadBytes() expected error for invalid GCS path, got nil")
		}
		if !strings.Contains(err.Error(), "invalid GCS path") {
			t.Errorf("expected 'invalid GCS path' error, got: %v", err)
		}
	})
}
