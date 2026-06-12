package debug

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/lmtani/pumbaa/internal/application/workflow"
	"github.com/lmtani/pumbaa/internal/interfaces/tui/common"
)

// countPreemptions counts the number of preempted tasks in a node and its children
func countPreemptions(node *TreeNode) int {
	count := 0
	// Check if this node itself is preempted
	if node.Status == "Preempted" || node.Status == "RetryableFailure" {
		count++
	}
	// Also check CallData for preemption status
	if node.CallData != nil && (string(node.CallData.Status) == "Preempted" || string(node.CallData.Status) == "RetryableFailure") {
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
	return common.FormatDuration(d)
}

// truncate truncates a string to maxLen characters.
func truncate(s string, maxLen int) string {
	return common.Truncate(s, maxLen)
}

// truncatePath truncates a GCS or local path intelligently, keeping bucket and basename visible.
// Example: gs://bucket-name/workspace/workflow/uuid/call-Name/file.log -> gs://bucket.../uuid.../file.log
func truncatePath(path string, maxLen int) string {
	if len(path) <= maxLen {
		return path
	}

	// Handle GCS paths
	if strings.HasPrefix(path, "gs://") {
		path = strings.TrimPrefix(path, "gs://")
		parts := strings.Split(path, "/")
		if len(parts) < 2 {
			return "gs://" + truncate(path, maxLen-5)
		}

		bucket := parts[0]
		// Truncate bucket if too long
		if len(bucket) > 12 {
			bucket = bucket[:9] + "..."
		}

		// Find UUID-like segment (36 chars with dashes)
		var uuid string
		for _, p := range parts {
			if len(p) == 36 && strings.Count(p, "-") == 4 {
				uuid = p[:8] + "..."
				break
			}
		}

		// Keep full basename (filename) - never truncate
		basename := parts[len(parts)-1]

		if uuid != "" {
			return fmt.Sprintf("gs://%s/%s/%s", bucket, uuid, basename)
		}
		return fmt.Sprintf("gs://%s/.../%s", bucket, basename)
	}

	// For local paths, keep the basename and truncate the directory part
	lastSlash := strings.LastIndex(path, "/")
	if lastSlash > 0 {
		basename := path[lastSlash+1:]
		dirPart := []rune(path[:lastSlash])
		availableLen := maxLen - len(basename) - 4 // ".../"
		if availableLen > 10 && availableLen <= len(dirPart) {
			return string(dirPart[:availableLen]) + ".../" + basename
		}
	}

	// Fallback: truncate from middle (rune-safe)
	runes := []rune(path)
	half := (maxLen - 3) / 2
	if half < 1 || len(runes) <= maxLen {
		return path
	}
	return string(runes[:half]) + "..." + string(runes[len(runes)-half:])
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

// wrapText wraps text to fit within maxWidth characters.
func wrapText(text string, maxWidth int) string {
	return common.WrapText(text, maxWidth)
}

// clipboardCopiedMsg is an alias for common.ClipboardCopiedMsg
type clipboardCopiedMsg = common.ClipboardCopiedMsg

// fetchTotalCost fetches the total cost asynchronously.
func (m Model) fetchTotalCost() tea.Cmd {
	if m.fetcher == nil {
		return nil
	}

	workflowID := m.metadata.ID
	return func() tea.Msg {
		cost, _, err := m.fetcher.GetWorkflowCost(context.Background(), workflowID)
		if err != nil {
			// Silently fail - just don't show cost
			return costLoadedMsg{totalCost: 0}
		}
		return costLoadedMsg{totalCost: cost}
	}
}

func (m Model) loadResourceAnalysis(path string) tea.Cmd {
	return func() tea.Msg {
		if m.monitoringUC == nil {
			return resourceAnalysisErrorMsg{err: fmt.Errorf("monitoring use case not initialized")}
		}

		// Use the injected monitoring use case to analyze resource usage
		// We use context.Background() here as we don't have a context in the Model yet
		result, err := m.monitoringUC.Execute(context.Background(), workflow.MonitoringInput{LogPath: path})
		if err != nil {
			return resourceAnalysisErrorMsg{err: err}
		}

		return resourceAnalysisLoadedMsg{report: result.Report}
	}
}

// copyToClipboard creates a tea.Cmd that copies text to the system clipboard.
func copyToClipboard(text, ctx string) tea.Cmd {
	return common.CopyToClipboard(text, ctx)
}

// getRawInputsJSON returns the workflow inputs as raw JSON string
func (m Model) getRawInputsJSON() string {
	if m.metadata.SubmittedInputs == "" {
		return "{}"
	}
	return m.metadata.SubmittedInputs
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
