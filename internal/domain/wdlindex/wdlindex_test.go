package wdlindex

import (
	"testing"
)

func TestNewIndex(t *testing.T) {
	dir := "/tmp/test"
	idx := NewIndex(dir)

	if idx.Directory != dir {
		t.Errorf("got %s, want %s", idx.Directory, dir)
	}

	if idx.Version != 1 {
		t.Errorf("got %d, want 1", idx.Version)
	}

	if idx.Tasks == nil {
		t.Error("Tasks map should be initialized")
	}

	if idx.Workflows == nil {
		t.Error("Workflows map should be initialized")
	}

	if idx.IndexedAt.IsZero() {
		t.Error("IndexedAt should be set")
	}
}
