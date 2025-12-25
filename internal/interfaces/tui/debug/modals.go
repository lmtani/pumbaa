package debug

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/lmtani/pumbaa/internal/interfaces/tui/common"
)

// modalFooter generates the footer for modals, including copy feedback if present
func (m Model) modalFooter() string {
	baseFooter := "↑↓/PgUp/PgDn scroll • y copy • esc close"
	if m.statusMessage != "" {
		return mutedStyle.Render(baseFooter) + "  " + temporaryStatusStyle.Render(m.statusMessage)
	}
	return mutedStyle.Render(baseFooter)
}

// formatValueForModal formats a value for display in modals with appropriate colors.
func formatValueForModal(v interface{}, maxWidth int) string {
	return formatValueWithStyles(v, maxWidth, modalValueStyle, modalPathStyle, mutedStyle)
}

// formatValueWithStyles formats a value using the provided styles.
func formatValueWithStyles(v interface{}, maxWidth int, valStyle, pthStyle, mutStyle lipgloss.Style) string {
	if maxWidth < 20 {
		maxWidth = 80
	}

	switch val := v.(type) {
	case nil:
		return mutStyle.Render("  null")
	case bool:
		return valStyle.Render(fmt.Sprintf("  %v", val))
	case float64:
		// Check if it's an integer
		if val == float64(int64(val)) {
			return valStyle.Render(fmt.Sprintf("  %d", int64(val)))
		}
		return valStyle.Render(fmt.Sprintf("  %g", val))
	case string:
		wrappedVal := val
		if len(val) > maxWidth-4 {
			wrappedVal = wrapText(val, maxWidth-4)
		}
		// Handle GCS paths with special styling
		if strings.HasPrefix(val, "gs://") {
			return pthStyle.Render("  " + wrappedVal)
		}
		// Handle local paths
		if strings.HasPrefix(val, "/") {
			return pthStyle.Render("  " + wrappedVal)
		}
		return valStyle.Render("  " + wrappedVal)
	case []interface{}:
		if len(val) == 0 {
			return mutStyle.Render("  []")
		}
		var sb strings.Builder
		for i, item := range val {
			prefix := "  - "
			itemStr := formatValueWithStyles(item, maxWidth-4, valStyle, pthStyle, mutStyle)
			// Remove leading spaces from nested formatValue
			itemStr = strings.TrimPrefix(itemStr, "  ")
			sb.WriteString(prefix + itemStr)
			if i < len(val)-1 {
				sb.WriteString("\n")
			}
		}
		return sb.String()
	case map[string]interface{}:
		// Pretty print maps with indentation
		jsonBytes, err := json.MarshalIndent(val, "  ", "  ")
		if err != nil {
			return mutStyle.Render("  [complex object]")
		}
		highlighted := common.Highlight(string(jsonBytes), common.ProfileJSON, maxWidth-4)
		return strings.ReplaceAll(highlighted, "\n", "\n  ")
	default:
		// Fallback to JSON for unknown types
		jsonBytes, err := json.MarshalIndent(val, "  ", "  ")
		if err != nil {
			return valStyle.Render(fmt.Sprintf("  %v", val))
		}
		highlighted := common.Highlight(string(jsonBytes), common.ProfileJSON, maxWidth-4)
		return strings.ReplaceAll(highlighted, "\n", "\n  ")
	}
}
