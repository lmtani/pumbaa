package storage

import (
	"context"
	"crypto/md5"
	"encoding/base64"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"hash/crc32"
	"io"
	"io/fs"
	"os"
	"strings"

	"github.com/lmtani/pumbaa/internal/application/ports"
)

// LocalBackend implements StorageBackend for local filesystem access.
type LocalBackend struct{}

// NewLocalBackend creates a new LocalBackend.
func NewLocalBackend() *LocalBackend {
	return &LocalBackend{}
}

// CanHandle returns true for paths that are not cloud storage paths.
// LocalBackend acts as the fallback handler for non-prefixed paths.
func (l *LocalBackend) CanHandle(path string) bool {
	return !strings.HasPrefix(path, "gs://") &&
		!strings.HasPrefix(path, "s3://") &&
		!strings.HasPrefix(path, "az://")
}

// Read reads the content of a local file as a string.
// Enforces maxFileSize limit to prevent memory issues.
func (l *LocalBackend) Read(_ context.Context, path string) (string, error) {
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

// ReadBytes reads the content of a local file as raw bytes.
// No size limit is enforced, suitable for binary files like ZIP dependencies.
func (l *LocalBackend) ReadBytes(_ context.Context, path string) ([]byte, error) {
	return os.ReadFile(path)
}

// GetSize returns the size of a local file without reading its content.
func (l *LocalBackend) GetSize(_ context.Context, path string) (int64, error) {
	info, err := os.Stat(path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return 0, fmt.Errorf("%w: %s", ports.ErrFileNotFound, path)
		}
		return 0, fmt.Errorf("failed to stat file: %w", err)
	}
	return info.Size(), nil
}

// GetContentDigests returns a local file's checksums, streaming the file once
// so large inputs never have to fit in memory and both digests come from a
// single read.
func (l *LocalBackend) GetContentDigests(_ context.Context, path string) (ports.FileDigests, error) {
	f, err := os.Open(path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return ports.FileDigests{}, fmt.Errorf("%w: %s", ports.ErrFileNotFound, path)
		}
		return ports.FileDigests{}, fmt.Errorf("failed to open file: %w", err)
	}
	defer func() { _ = f.Close() }()

	sum := md5.New() //nolint:gosec // reproducing Cromwell's hash, not a security primitive
	crc := crc32.New(crc32.MakeTable(crc32.Castagnoli))
	if _, err := io.Copy(io.MultiWriter(sum, crc), f); err != nil {
		return ports.FileDigests{}, fmt.Errorf("failed to hash file: %w", err)
	}
	return ports.FileDigests{
		MD5:    hex.EncodeToString(sum.Sum(nil)),
		CRC32C: encodeCRC32C(crc.Sum32()),
	}, nil
}

// encodeCRC32C renders a Castagnoli checksum the way GCS reports it and
// Cromwell stores it: base64 of the four bytes, big-endian.
func encodeCRC32C(sum uint32) string {
	var b [4]byte
	binary.BigEndian.PutUint32(b[:], sum)
	return base64.StdEncoding.EncodeToString(b[:])
}

// Ensure LocalBackend implements StorageBackend at compile time.
var _ ports.StorageBackend = (*LocalBackend)(nil)
