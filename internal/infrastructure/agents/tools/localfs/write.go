// Package localfs provides action handlers for writing files in the user's
// working directory — e.g. debug scripts the agent generates on request
// (fetch task inputs with gsutil, reproduce an analysis locally with docker).
package localfs

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/lmtani/pumbaa/internal/infrastructure/agents/tools/types"
)

// WriteHandler handles the "write_file" action.
type WriteHandler struct{}

// NewWriteHandler creates a new WriteHandler.
func NewWriteHandler() *WriteHandler {
	return &WriteHandler{}
}

// Handle implements types.Handler.
func (h *WriteHandler) Handle(_ context.Context, input types.Input) (types.Output, error) {
	const action = "write_file"

	if strings.TrimSpace(input.Content) == "" {
		return types.NewErrorOutput(action, "content is required"), nil
	}

	fullPath, err := ResolveWorkingDirPath(input.Path)
	if err != nil {
		return types.NewErrorOutput(action, err.Error()), nil
	}

	if _, err := os.Stat(fullPath); err == nil && !input.Overwrite {
		return types.NewErrorOutput(action, fmt.Sprintf(
			"file already exists: %s. Pass overwrite=true to replace it, or choose another name.",
			input.Path,
		)), nil
	}

	if err := os.MkdirAll(filepath.Dir(fullPath), 0o755); err != nil {
		return types.NewErrorOutput(action, fmt.Sprintf("failed to create directory: %v", err)), nil
	}

	perm := os.FileMode(0o644)
	if input.Executable {
		perm = 0o755
	}
	if err := os.WriteFile(fullPath, []byte(input.Content), perm); err != nil {
		return types.NewErrorOutput(action, fmt.Sprintf("failed to write file: %v", err)), nil
	}
	if input.Executable {
		// WriteFile's perm only applies on creation; ensure the bit on overwrite
		if err := os.Chmod(fullPath, 0o755); err != nil {
			return types.NewErrorOutput(action, fmt.Sprintf("failed to mark file executable: %v", err)), nil
		}
	}

	return types.NewSuccessOutput(action, map[string]any{
		"path":       fullPath,
		"bytes":      len(input.Content),
		"executable": input.Executable,
	}), nil
}

// ResolveWorkingDirPath validates that the requested path stays inside the
// user's current working directory and returns its absolute form. The agent
// must never write outside the directory pumbaa was launched from.
func ResolveWorkingDirPath(path string) (string, error) {
	if strings.TrimSpace(path) == "" {
		return "", fmt.Errorf("path is required (relative to the current working directory)")
	}
	if filepath.IsAbs(path) {
		return "", fmt.Errorf("path must be relative to the current working directory, got absolute path %q", path)
	}

	cwd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("failed to resolve working directory: %v", err)
	}

	fullPath := filepath.Clean(filepath.Join(cwd, path))
	if fullPath == cwd {
		return "", fmt.Errorf("path %q resolves to the working directory itself, not a file", path)
	}
	if !strings.HasPrefix(fullPath, cwd+string(os.PathSeparator)) {
		return "", fmt.Errorf("path %q escapes the current working directory", path)
	}

	return fullPath, nil
}
