package debug

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"cloud.google.com/go/storage"
	tea "github.com/charmbracelet/bubbletea"
	"golang.org/x/net/context"
)

// maxLogSize is the maximum log file size we'll read (1 MB)
const maxLogSize = 1 * 1024 * 1024

// countPreemptions counts the number of preempted tasks in a node and its children
func countPreemptions(node *TreeNode) int {
	count := 0
	// Check if this node itself is preempted
	if node.Status == "Preempted" || node.Status == "RetryableFailure" {
		count++
	}
	// Also check CallData for preemption status
	if node.CallData != nil && (node.CallData.ExecutionStatus == "Preempted" || node.CallData.ExecutionStatus == "RetryableFailure") {
		if node.Status != "Preempted" && node.Status != "RetryableFailure" {
			count++ // Only count if not already counted
		}
	}
	// Recursively count children
	for _, child := range node.Children {
		count += countPreemptions(child)
	}
	return count
}

// formatDuration formats a duration for display.
func formatDuration(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%.1fs", d.Seconds())
	}
	if d < time.Hour {
		return fmt.Sprintf("%.1fm", d.Minutes())
	}
	h := int(d.Hours())
	m := int(d.Minutes()) % 60
	return fmt.Sprintf("%dh%dm", h, m)
}

// truncate truncates a string to maxLen characters.
func truncate(s string, maxLen int) string {
	if maxLen <= 3 {
		return s
	}
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

// formatDockerImage formats a Docker image name for display.
// It breaks long image names into readable parts: registry, repository, and tag.
func formatDockerImage(image string) string {
	var sb strings.Builder

	// Parse the image into components
	// Format: [registry/][namespace/]name[:tag][@digest]

	// Handle digest (sha256:...)
	digest := ""
	if idx := strings.Index(image, "@"); idx != -1 {
		digest = image[idx+1:]
		image = image[:idx]
	}

	// Handle tag
	tag := "latest"
	if idx := strings.LastIndex(image, ":"); idx != -1 {
		// Make sure it's not a port number (registry:port/image)
		afterColon := image[idx+1:]
		if !strings.Contains(afterColon, "/") {
			tag = afterColon
			image = image[:idx]
		}
	}

	// Split by slashes to get registry and path
	parts := strings.Split(image, "/")

	var registry, path string
	if len(parts) == 1 {
		// Simple image like "python" or "ubuntu"
		path = parts[0]
	} else if len(parts) == 2 {
		// Could be "namespace/image" (Docker Hub) or "registry/image"
		if strings.Contains(parts[0], ".") || strings.Contains(parts[0], ":") {
			// It's a registry
			registry = parts[0]
			path = parts[1]
		} else {
			// It's a Docker Hub namespace
			path = image
		}
	} else {
		// Full path like "registry.io/namespace/image"
		registry = parts[0]
		path = strings.Join(parts[1:], "/")
	}

	// Format output
	if registry != "" {
		sb.WriteString("  " + mutedStyle.Render(registry+"/") + "\n")
		sb.WriteString("  " + pathStyle.Render(path))
	} else {
		sb.WriteString("  " + pathStyle.Render(path))
	}

	// Add tag with highlighting
	sb.WriteString(valueStyle.Render(":") + tagStyle.Render(tag))

	// Add digest if present
	if digest != "" {
		sb.WriteString("\n  " + mutedStyle.Render("@"+truncate(digest, 20)))
	}

	sb.WriteString("\n")
	return sb.String()
}

// nodeTypeName returns a human-readable name for a NodeType.
func nodeTypeName(t NodeType) string {
	switch t {
	case NodeTypeWorkflow:
		return "Workflow"
	case NodeTypeCall:
		return "Call"
	case NodeTypeSubWorkflow:
		return "SubWorkflow"
	case NodeTypeShard:
		return "Shard"
	default:
		return "Unknown"
	}
}

// wrapText wraps text to fit within maxWidth characters.
// It respects existing line breaks and wraps long lines.
func wrapText(text string, maxWidth int) string {
	if maxWidth <= 0 {
		maxWidth = 80
	}

	var result strings.Builder
	lines := strings.Split(text, "\n")

	for i, line := range lines {
		if i > 0 {
			result.WriteString("\n")
		}

		if len(line) <= maxWidth {
			result.WriteString(line)
			continue
		}

		// Wrap long lines
		for len(line) > maxWidth {
			// Try to find a good break point (space)
			breakPoint := maxWidth
			for j := maxWidth; j > maxWidth/2; j-- {
				if line[j] == ' ' {
					breakPoint = j
					break
				}
			}

			result.WriteString(line[:breakPoint])
			result.WriteString("\n")
			line = strings.TrimLeft(line[breakPoint:], " ")
		}
		if len(line) > 0 {
			result.WriteString(line)
		}
	}

	return result.String()
}

// readGCSFile reads a file from Google Cloud Storage
func readGCSFile(path string) (string, error) {
	// Parse gs://bucket/object path
	path = strings.TrimPrefix(path, "gs://")
	parts := strings.SplitN(path, "/", 2)
	if len(parts) != 2 {
		return "", fmt.Errorf("invalid GCS path: gs://%s", path)
	}
	bucket := parts[0]
	object := parts[1]

	ctx := context.Background()
	client, err := storage.NewClient(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to create GCS client: %w", err)
	}
	defer client.Close()

	// Get object attributes to check size
	attrs, err := client.Bucket(bucket).Object(object).Attrs(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to get object attributes: %w", err)
	}

	if attrs.Size > maxLogSize {
		return "", fmt.Errorf("log file too large (%.2f MB > 1 MB limit)", float64(attrs.Size)/(1024*1024))
	}

	// Read the object
	rc, err := client.Bucket(bucket).Object(object).NewReader(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to open GCS object: %w", err)
	}
	defer rc.Close()

	data, err := io.ReadAll(rc)
	if err != nil {
		return "", fmt.Errorf("failed to read GCS object: %w", err)
	}

	return string(data), nil
}

// readLocalFile reads a local file
func readLocalFile(path string) (string, error) {
	info, err := os.Stat(path)
	if err != nil {
		return "", fmt.Errorf("failed to stat file: %w", err)
	}

	if info.Size() > maxLogSize {
		return "", fmt.Errorf("log file too large (%.2f MB > 1 MB limit)", float64(info.Size())/(1024*1024))
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("failed to read file: %w", err)
	}

	return string(data), nil
}

// clipboardCopiedMsg is sent when clipboard copy is complete
type clipboardCopiedMsg struct {
	success bool
	err     error
}

// copyToClipboard creates a tea.Cmd that copies text to the system clipboard
func copyToClipboard(text string) tea.Cmd {
	return func() tea.Msg {
		var cmd *exec.Cmd

		switch runtime.GOOS {
		case "darwin":
			cmd = exec.Command("pbcopy")
		case "linux":
			// Try xclip first, then xsel, then wl-copy (for Wayland)
			if _, err := exec.LookPath("xclip"); err == nil {
				cmd = exec.Command("xclip", "-selection", "clipboard")
			} else if _, err := exec.LookPath("xsel"); err == nil {
				cmd = exec.Command("xsel", "--clipboard", "--input")
			} else if _, err := exec.LookPath("wl-copy"); err == nil {
				cmd = exec.Command("wl-copy")
			} else {
				return clipboardCopiedMsg{success: false, err: fmt.Errorf("no clipboard tool found (install xclip, xsel, or wl-copy)")}
			}
		case "windows":
			cmd = exec.Command("clip")
		default:
			return clipboardCopiedMsg{success: false, err: fmt.Errorf("unsupported OS: %s", runtime.GOOS)}
		}

		stdin, err := cmd.StdinPipe()
		if err != nil {
			return clipboardCopiedMsg{success: false, err: err}
		}

		if err := cmd.Start(); err != nil {
			return clipboardCopiedMsg{success: false, err: err}
		}

		_, err = stdin.Write([]byte(text))
		if err != nil {
			return clipboardCopiedMsg{success: false, err: err}
		}
		stdin.Close()

		if err := cmd.Wait(); err != nil {
			return clipboardCopiedMsg{success: false, err: err}
		}

		return clipboardCopiedMsg{success: true}
	}
}

// getRawInputsJSON returns the workflow inputs as raw JSON string
func (m Model) getRawInputsJSON() string {
	if len(m.metadata.Inputs) == 0 {
		return "{}"
	}
	data, err := json.MarshalIndent(m.metadata.Inputs, "", "  ")
	if err != nil {
		return "{}"
	}
	return string(data)
}

// getRawOutputsJSON returns the workflow outputs as raw JSON string
func (m Model) getRawOutputsJSON() string {
	if len(m.metadata.Outputs) == 0 {
		return "{}"
	}
	data, err := json.MarshalIndent(m.metadata.Outputs, "", "  ")
	if err != nil {
		return "{}"
	}
	return string(data)
}

// getRawOptionsJSON returns the workflow options as raw JSON string
func (m Model) getRawOptionsJSON() string {
	if m.metadata.SubmittedOptions == "" {
		return "{}"
	}
	return m.metadata.SubmittedOptions
}

// getRawCallInputsJSON returns the call inputs as raw JSON string
func (m Model) getRawCallInputsJSON(node *TreeNode) string {
	if node.CallData == nil || len(node.CallData.Inputs) == 0 {
		return "{}"
	}
	data, err := json.MarshalIndent(node.CallData.Inputs, "", "  ")
	if err != nil {
		return "{}"
	}
	return string(data)
}

// getRawCallOutputsJSON returns the call outputs as raw JSON string
func (m Model) getRawCallOutputsJSON(node *TreeNode) string {
	if node.CallData == nil || len(node.CallData.Outputs) == 0 {
		return "{}"
	}
	data, err := json.MarshalIndent(node.CallData.Outputs, "", "  ")
	if err != nil {
		return "{}"
	}
	return string(data)
}
