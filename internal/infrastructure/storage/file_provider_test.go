package storage

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestFileProvider_Read_Local(t *testing.T) {
	fp := NewFileProvider()
	ctx := context.Background()

	t.Run("read valid local file", func(t *testing.T) {
		content := "hello world"
		tmpDir := t.TempDir()
		tmpFile := filepath.Join(tmpDir, "test.txt")
		err := os.WriteFile(tmpFile, []byte(content), 0644)
		if err != nil {
			t.Fatalf("failed to create temp file: %v", err)
		}

		got, err := fp.Read(ctx, tmpFile)
		if err != nil {
			t.Errorf("Read() unexpected error: %v", err)
		}
		if got != content {
			t.Errorf("Read() = %q, want %q", got, content)
		}
	})

	t.Run("read non-existent file", func(t *testing.T) {
		_, err := fp.Read(ctx, "non-existent-file.txt")
		if err == nil {
			t.Error("Read() expected error for non-existent file, got nil")
		}
	})

	t.Run("file too large", func(t *testing.T) {
		tmpDir := t.TempDir()
		tmpFile := filepath.Join(tmpDir, "large.txt")
		
		// Create a file larger than 1MB
		largeContent := make([]byte, maxFileSize+1)
		err := os.WriteFile(tmpFile, largeContent, 0644)
		if err != nil {
			t.Fatalf("failed to create large temp file: %v", err)
		}

		_, err = fp.Read(ctx, tmpFile)
		if err == nil {
			t.Error("Read() expected error for large file, got nil")
		}
		if !strings.Contains(err.Error(), "file too large") {
			t.Errorf("expected 'file too large' error, got: %v", err)
		}
	})
}

func TestFileProvider_Read_GCS_Validation(t *testing.T) {
	fp := NewFileProvider()
	ctx := context.Background()

	t.Run("invalid GCS path", func(t *testing.T) {
		_, err := fp.Read(ctx, "gs://invalid-path")
		if err == nil {
			t.Error("Read() expected error for invalid GCS path, got nil")
		}
		if !strings.Contains(err.Error(), "invalid GCS path") {
			t.Errorf("expected 'invalid GCS path' error, got: %v", err)
		}
	})

	t.Run("failed to create GCS client", func(t *testing.T) {
		// This should fail in most CI/test environments because of missing credentials
		// unless GOOGLE_APPLICATION_CREDENTIALS is set.
		// We want to verify it handles the error.
		_, err := fp.Read(ctx, "gs://bucket/object")
		if err == nil {
			// If it doesn't fail, maybe the environment has credentials, but usually it should.
			return
		}
		// We just check that it returns an error, specifically mention client creation or attributes
		if !strings.Contains(err.Error(), "failed to create GCS client") && 
		   !strings.Contains(err.Error(), "failed to get object attributes") {
			t.Errorf("unexpected error message: %v", err)
		}
	})
}
