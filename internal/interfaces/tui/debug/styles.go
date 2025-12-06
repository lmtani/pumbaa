package debug

import "github.com/charmbracelet/lipgloss"

// Colors
var (
	// Status colors
	statusSucceeded = lipgloss.Color("#00ff00")
	statusFailed    = lipgloss.Color("#ff0000")
	statusRunning   = lipgloss.Color("#ffff00")
	statusPending   = lipgloss.Color("#888888")

	// UI colors
	primaryColor   = lipgloss.Color("#7D56F4")
	secondaryColor = lipgloss.Color("#5A4FCF")
	borderColor    = lipgloss.Color("#444444")
	focusBorder    = lipgloss.Color("#7D56F4")
	textColor      = lipgloss.Color("#FAFAFA")
	mutedColor     = lipgloss.Color("#888888")
	highlightColor = lipgloss.Color("#874BFD")
)

// Styles
var (
	// Base styles
	baseStyle = lipgloss.NewStyle().
			Foreground(textColor)

	// Header style
	headerStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(primaryColor).
			BorderStyle(lipgloss.NormalBorder()).
			BorderBottom(true).
			BorderForeground(borderColor).
			Padding(0, 1)

	// Panel styles
	panelStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(borderColor).
			Padding(0, 1)

	focusedPanelStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(focusBorder).
				Padding(0, 1)

	// Title styles
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(primaryColor).
			MarginBottom(1)

	// Tree node styles
	treeNodeStyle = lipgloss.NewStyle().
			Foreground(textColor)

	selectedNodeStyle = lipgloss.NewStyle().
				Foreground(textColor).
				Background(highlightColor)

	// Status styles
	statusDoneStyle = lipgloss.NewStyle().
			Foreground(statusSucceeded)

	statusFailedStyle = lipgloss.NewStyle().
				Foreground(statusFailed)

	statusRunningStyle = lipgloss.NewStyle().
				Foreground(statusRunning)

	statusPendingStyle = lipgloss.NewStyle().
				Foreground(statusPending)

	// Label styles
	labelStyle = lipgloss.NewStyle().
			Foreground(mutedColor)

	valueStyle = lipgloss.NewStyle().
			Foreground(textColor)

	// Help bar style
	helpBarStyle = lipgloss.NewStyle().
			Foreground(mutedColor).
			BorderStyle(lipgloss.NormalBorder()).
			BorderTop(true).
			BorderForeground(borderColor).
			Padding(0, 1)

	// Command style
	commandStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#00ffff")).
			Background(lipgloss.Color("#1a1a1a")).
			Padding(1)

	// Path style (for GCS/file paths)
	pathStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#87ceeb")).
			Italic(true)

	// Key style for key bindings
	keyStyle = lipgloss.NewStyle().
			Foreground(primaryColor).
			Bold(true)

	// Description style for help
	descStyle = lipgloss.NewStyle().
			Foreground(mutedColor)

	// Muted style for less important text
	mutedStyle = lipgloss.NewStyle().
			Foreground(mutedColor)
)

// StatusStyle returns the appropriate style for a status.
func StatusStyle(status string) lipgloss.Style {
	switch status {
	case "Done", "Succeeded":
		return statusDoneStyle
	case "Failed":
		return statusFailedStyle
	case "Running":
		return statusRunningStyle
	default:
		return statusPendingStyle
	}
}

// StatusIcon returns an icon for the status.
func StatusIcon(status string) string {
	switch status {
	case "Done", "Succeeded":
		return "✓"
	case "Failed":
		return "✗"
	case "Running":
		return "●"
	default:
		return "○"
	}
}

// NodeTypeIcon returns an icon for the node type.
func NodeTypeIcon(t NodeType) string {
	switch t {
	case NodeTypeWorkflow:
		return "📋"
	case NodeTypeSubWorkflow:
		return "📂"
	case NodeTypeCall:
		return "⚙"
	case NodeTypeShard:
		return "  "
	default:
		return " "
	}
}

// TreePrefix returns the tree drawing prefix.
func TreePrefix(depth int, isLast bool, parentExpanded bool) string {
	if depth == 0 {
		return ""
	}

	prefix := ""
	for i := 0; i < depth-1; i++ {
		prefix += "│  "
	}

	if isLast {
		prefix += "└──"
	} else {
		prefix += "├──"
	}

	return prefix
}

// ExpandIcon returns the expand/collapse icon.
func ExpandIcon(expanded bool, hasChildren bool) string {
	if !hasChildren {
		return " "
	}
	if expanded {
		return "▼"
	}
	return "▶"
}
