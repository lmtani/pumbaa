package storage

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/lmtani/pumbaa/internal/domain/ports"
)

func TestFileProvider_DelegatesToCorrectBackend(t *testing.T) {
	ctx := context.Background()

	// Create a mock backend for testing
	mockBackend := &mockStorageBackend{
		canHandleFunc: func(path string) bool {
			return strings.HasPrefix(path, "mock://")
		},
		readFunc: func(_ context.Context, path string) (string, error) {
			return "mock content for " + path, nil
		},
	}

	fp := NewFileProviderWithBackends(mockBackend, NewLocalBackend())

	t.Run("delegates to mock backend", func(t *testing.T) {
		got, err := fp.Read(ctx, "mock://test/file.txt")
		if err != nil {
			t.Errorf("Read() unexpected error: %v", err)
		}
		if got != "mock content for mock://test/file.txt" {
			t.Errorf("Read() = %q, want mock content", got)
		}
	})

	t.Run("falls back to local backend", func(t *testing.T) {
		tmpDir := t.TempDir()
		tmpFile := filepath.Join(tmpDir, "test.txt")
		err := os.WriteFile(tmpFile, []byte("local content"), 0644)
		if err != nil {
			t.Fatalf("failed to create temp file: %v", err)
		}

		got, err := fp.Read(ctx, tmpFile)
		if err != nil {
			t.Errorf("Read() unexpected error: %v", err)
		}
		if got != "local content" {
			t.Errorf("Read() = %q, want %q", got, "local content")
		}
	})
}

func TestFileProvider_ReadBytes_DelegatesToCorrectBackend(t *testing.T) {
	ctx := context.Background()

	mockBackend := &mockStorageBackend{
		canHandleFunc: func(path string) bool {
			return strings.HasPrefix(path, "mock://")
		},
		readBytesFunc: func(_ context.Context, path string) ([]byte, error) {
			return []byte("mock bytes"), nil
		},
	}

	fp := NewFileProviderWithBackends(mockBackend, NewLocalBackend())

	t.Run("delegates to mock backend", func(t *testing.T) {
		got, err := fp.ReadBytes(ctx, "mock://test/file.bin")
		if err != nil {
			t.Errorf("ReadBytes() unexpected error: %v", err)
		}
		if string(got) != "mock bytes" {
			t.Errorf("ReadBytes() = %q, want mock bytes", got)
		}
	})
}

func TestFileProvider_NoBackendFound(t *testing.T) {
	ctx := context.Background()

	// Create a FileProvider with a backend that handles nothing
	neverHandles := &mockStorageBackend{
		canHandleFunc: func(_ string) bool { return false },
	}
	fp := NewFileProviderWithBackends(neverHandles)

	t.Run("Read returns error", func(t *testing.T) {
		_, err := fp.Read(ctx, "any://path")
		if err == nil {
			t.Error("Read() expected error when no backend found, got nil")
		}
		if !strings.Contains(err.Error(), "no storage backend found") {
			t.Errorf("expected 'no storage backend found' error, got: %v", err)
		}
	})

	t.Run("ReadBytes returns error", func(t *testing.T) {
		_, err := fp.ReadBytes(ctx, "any://path")
		if err == nil {
			t.Error("ReadBytes() expected error when no backend found, got nil")
		}
		if !strings.Contains(err.Error(), "no storage backend found") {
			t.Errorf("expected 'no storage backend found' error, got: %v", err)
		}
	})
}

func TestNewFileProvider_DefaultBackends(t *testing.T) {
	fp := NewFileProvider()

	if len(fp.backends) != 2 {
		t.Errorf("NewFileProvider() should have 2 backends, got %d", len(fp.backends))
	}
}

// mockStorageBackend is a configurable mock for testing.
type mockStorageBackend struct {
	canHandleFunc func(path string) bool
	readFunc      func(ctx context.Context, path string) (string, error)
	readBytesFunc func(ctx context.Context, path string) ([]byte, error)
}

func (m *mockStorageBackend) CanHandle(path string) bool {
	if m.canHandleFunc != nil {
		return m.canHandleFunc(path)
	}
	return false
}

func (m *mockStorageBackend) Read(ctx context.Context, path string) (string, error) {
	if m.readFunc != nil {
		return m.readFunc(ctx, path)
	}
	return "", nil
}

func (m *mockStorageBackend) ReadBytes(ctx context.Context, path string) ([]byte, error) {
	if m.readBytesFunc != nil {
		return m.readBytesFunc(ctx, path)
	}
	return nil, nil
}

var _ ports.StorageBackend = (*mockStorageBackend)(nil)
