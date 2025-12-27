package storage

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLocalBackend_CanHandle(t *testing.T) {
	backend := NewLocalBackend()

	tests := []struct {
		name     string
		path     string
		expected bool
	}{
		{"local relative path", "file.txt", true},
		{"local absolute path", "/home/user/file.txt", true},
		{"GCS path", "gs://bucket/file.txt", false},
		{"S3 path", "s3://bucket/file.txt", false},
		{"Azure path", "az://container/file.txt", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := backend.CanHandle(tt.path); got != tt.expected {
				t.Errorf("CanHandle(%q) = %v, want %v", tt.path, got, tt.expected)
			}
		})
	}
}

func TestLocalBackend_Read(t *testing.T) {
	backend := NewLocalBackend()
	ctx := context.Background()

	t.Run("read valid local file", func(t *testing.T) {
		content := "hello world"
		tmpDir := t.TempDir()
		tmpFile := filepath.Join(tmpDir, "test.txt")
		err := os.WriteFile(tmpFile, []byte(content), 0644)
		if err != nil {
			t.Fatalf("failed to create temp file: %v", err)
		}

		got, err := backend.Read(ctx, tmpFile)
		if err != nil {
			t.Errorf("Read() unexpected error: %v", err)
		}
		if got != content {
			t.Errorf("Read() = %q, want %q", got, content)
		}
	})

	t.Run("read non-existent file", func(t *testing.T) {
		_, err := backend.Read(ctx, "non-existent-file.txt")
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

		_, err = backend.Read(ctx, tmpFile)
		if err == nil {
			t.Error("Read() expected error for large file, got nil")
		}
		if !strings.Contains(err.Error(), "file too large") {
			t.Errorf("expected 'file too large' error, got: %v", err)
		}
	})
}

func TestLocalBackend_ReadBytes(t *testing.T) {
	backend := NewLocalBackend()
	ctx := context.Background()

	t.Run("read bytes from valid file", func(t *testing.T) {
		content := []byte{0x00, 0x01, 0x02, 0x03}
		tmpDir := t.TempDir()
		tmpFile := filepath.Join(tmpDir, "binary.bin")
		err := os.WriteFile(tmpFile, content, 0644)
		if err != nil {
			t.Fatalf("failed to create temp file: %v", err)
		}

		got, err := backend.ReadBytes(ctx, tmpFile)
		if err != nil {
			t.Errorf("ReadBytes() unexpected error: %v", err)
		}
		if string(got) != string(content) {
			t.Errorf("ReadBytes() content mismatch")
		}
	})

	t.Run("read bytes from non-existent file", func(t *testing.T) {
		_, err := backend.ReadBytes(ctx, "non-existent-file.bin")
		if err == nil {
			t.Error("ReadBytes() expected error for non-existent file, got nil")
		}
	})
}
