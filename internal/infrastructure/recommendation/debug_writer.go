package recommendation

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/lmtani/pumbaa/internal/application/ports"
)

// FileDebugWriter implements LLMDebugWriter by writing to a file.
type FileDebugWriter struct {
	file *os.File
}

// NewFileDebugWriter creates a new debug writer that writes to the specified file.
// If the file exists, it will be overwritten.
func NewFileDebugWriter(filePath string) (*FileDebugWriter, error) {
	// Ensure directory exists
	dir := filepath.Dir(filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create debug directory: %w", err)
	}

	file, err := os.Create(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to create debug file: %w", err)
	}

	// Write header
	header := fmt.Sprintf("# LLM Debug Log\n# Generated: %s\n\n", time.Now().Format(time.RFC3339))
	if _, err := file.WriteString(header); err != nil {
		file.Close()
		return nil, fmt.Errorf("failed to write header: %w", err)
	}

	return &FileDebugWriter{file: file}, nil
}

// WriteInteraction logs a complete LLM interaction.
func (w *FileDebugWriter) WriteInteraction(callType, systemInstruction, prompt, response string) error {
	if w.file == nil {
		return nil
	}

	content := fmt.Sprintf("=== %s ===\nTimestamp: %s\n\n", callType, time.Now().Format(time.RFC3339))
	content += fmt.Sprintf("--- SYSTEM INSTRUCTION ---\n%s\n\n", systemInstruction)
	content += fmt.Sprintf("--- USER PROMPT ---\n%s\n\n", prompt)
	content += fmt.Sprintf("--- LLM RESPONSE ---\n%s\n\n", response)
	content += "========================================\n\n"

	_, err := w.file.WriteString(content)
	return err
}

// Close closes the file.
func (w *FileDebugWriter) Close() error {
	if w.file == nil {
		return nil
	}
	return w.file.Close()
}

// Ensure FileDebugWriter implements LLMDebugWriter
var _ ports.LLMDebugWriter = (*FileDebugWriter)(nil)

// NullDebugWriter is a no-op implementation of LLMDebugWriter.
type NullDebugWriter struct{}

// WriteInteraction does nothing.
func (w *NullDebugWriter) WriteInteraction(_, _, _, _ string) error {
	return nil
}

// Close does nothing.
func (w *NullDebugWriter) Close() error {
	return nil
}

// Ensure NullDebugWriter implements LLMDebugWriter
var _ ports.LLMDebugWriter = (*NullDebugWriter)(nil)
