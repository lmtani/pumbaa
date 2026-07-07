package localfs

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/lmtani/pumbaa/internal/infrastructure/agents/tools/types"
)

// chdir switches to a temp dir for the test, since the handler writes
// relative to the process working directory.
func chdir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	old, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(old) })
	// Resolve symlinks (macOS /tmp) so comparisons match os.Getwd
	resolved, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	return resolved
}

func TestWriteFileCreatesScript(t *testing.T) {
	dir := chdir(t)
	h := NewWriteHandler()

	out, err := h.Handle(context.Background(), types.Input{
		Action:     "write_file",
		Path:       "debug/reproduce.sh",
		Content:    "#!/bin/bash\ngsutil cp gs://bucket/input .\n",
		Executable: true,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !out.Success {
		t.Fatalf("expected success, got: %s", out.Error)
	}

	full := filepath.Join(dir, "debug", "reproduce.sh")
	info, err := os.Stat(full)
	if err != nil {
		t.Fatalf("file not written: %v", err)
	}
	if info.Mode()&0o111 == 0 {
		t.Errorf("expected executable bit, got mode %v", info.Mode())
	}
	data, _ := os.ReadFile(full)
	if !strings.Contains(string(data), "gsutil cp") {
		t.Errorf("unexpected content: %s", data)
	}
}

func TestWriteFileRefusesOverwriteWithoutFlag(t *testing.T) {
	chdir(t)
	h := NewWriteHandler()

	first, _ := h.Handle(context.Background(), types.Input{Path: "a.sh", Content: "one"})
	if !first.Success {
		t.Fatalf("first write failed: %s", first.Error)
	}

	second, _ := h.Handle(context.Background(), types.Input{Path: "a.sh", Content: "two"})
	if second.Success {
		t.Fatalf("expected refusal to overwrite without flag")
	}

	third, _ := h.Handle(context.Background(), types.Input{Path: "a.sh", Content: "two", Overwrite: true})
	if !third.Success {
		t.Fatalf("overwrite with flag failed: %s", third.Error)
	}
	data, _ := os.ReadFile("a.sh")
	if string(data) != "two" {
		t.Errorf("content not replaced: %s", data)
	}
}

func TestWriteFileRejectsUnsafePaths(t *testing.T) {
	chdir(t)
	h := NewWriteHandler()

	for _, path := range []string{"", "/etc/passwd", "../outside.sh", "sub/../../outside.sh", "."} {
		out, err := h.Handle(context.Background(), types.Input{Path: path, Content: "x"})
		if err != nil {
			t.Fatalf("unexpected transport error for %q: %v", path, err)
		}
		if out.Success {
			t.Errorf("expected rejection for path %q", path)
		}
	}
}

func TestWriteFileRequiresContent(t *testing.T) {
	chdir(t)
	h := NewWriteHandler()

	out, _ := h.Handle(context.Background(), types.Input{Path: "empty.sh", Content: "  "})
	if out.Success {
		t.Fatalf("expected rejection of empty content")
	}
}
