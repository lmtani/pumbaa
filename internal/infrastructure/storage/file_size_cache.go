package storage

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"

	"github.com/lmtani/pumbaa/internal/application/ports"
)

// FileSizeCache provides thread-safe caching of file sizes with persistent storage.
// It automatically loads from disk on first access and saves after modifications.
type FileSizeCache struct {
	mu     sync.RWMutex
	path   string
	sizes  map[string]int64
	loaded bool
	dirty  bool
}

// NewFileSizeCache creates a FileSizeCache using the default cache path.
func NewFileSizeCache() *FileSizeCache {
	return NewFileSizeCacheWithPath(defaultFileSizeCachePath())
}

// NewFileSizeCacheWithPath creates a FileSizeCache with a custom cache path.
func NewFileSizeCacheWithPath(path string) *FileSizeCache {
	return &FileSizeCache{
		path:  path,
		sizes: make(map[string]int64),
	}
}

// Load hydrates the cache from persistent storage.
func (c *FileSizeCache) Load() error {
	if c.path == "" {
		return nil
	}

	data, err := os.ReadFile(c.path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	var sizes map[string]int64
	if err := json.Unmarshal(data, &sizes); err != nil {
		return err
	}
	if sizes == nil {
		sizes = make(map[string]int64)
	}

	c.mu.Lock()
	c.sizes = sizes
	c.mu.Unlock()

	return nil
}

// Save persists the cache to storage if it has been modified.
func (c *FileSizeCache) Save() error {
	c.mu.Lock()
	if !c.dirty {
		c.mu.Unlock()
		return nil
	}
	c.dirty = false
	c.mu.Unlock()

	if c.path == "" {
		return nil
	}

	dir := filepath.Dir(c.path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	snapshot := c.snapshot()
	data, err := json.Marshal(snapshot)
	if err != nil {
		return err
	}

	return os.WriteFile(c.path, data, 0644)
}

// Get returns the cached size for a path.
// Automatically loads the cache from disk on first access.
func (c *FileSizeCache) Get(path string) (int64, bool) {
	c.ensureLoaded()
	c.mu.RLock()
	defer c.mu.RUnlock()
	size, ok := c.sizes[path]
	return size, ok
}

// Set caches the size for a path.
// Automatically loads the cache from disk on first access and marks it as dirty for saving.
func (c *FileSizeCache) Set(path string, size int64) {
	c.ensureLoaded()
	c.mu.Lock()
	if c.sizes == nil {
		c.sizes = make(map[string]int64)
	}
	c.sizes[path] = size
	c.dirty = true
	c.mu.Unlock()

	// Auto-save after modification
	_ = c.Save()
}

// ensureLoaded performs lazy loading of the cache from disk.
func (c *FileSizeCache) ensureLoaded() {
	c.mu.Lock()
	if c.loaded {
		c.mu.Unlock()
		return
	}
	c.loaded = true
	c.mu.Unlock()

	// Load outside of lock to avoid blocking other operations
	_ = c.Load()
}

func (c *FileSizeCache) snapshot() map[string]int64 {
	c.mu.RLock()
	defer c.mu.RUnlock()
	sizes := make(map[string]int64, len(c.sizes))
	for key, value := range c.sizes {
		sizes[key] = value
	}
	return sizes
}

func defaultFileSizeCachePath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".pumbaa", "input_sizes.json")
}

// Ensure FileSizeCache implements the domain interface at compile time.
var _ ports.FileSizeCache = (*FileSizeCache)(nil)
