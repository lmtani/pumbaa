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
	primaryColor = lipgloss.Color("#7D56F4")
	borderColor  = lipgloss.Color("#444444")
	textColor    = lipgloss.Color("#FAFAFA")
	mutedColor   = lipgloss.Color("#888888")
)

// Styles
var (
	// Header style
	headerStyle = lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(primaryColor).
			Padding(0, 1).
			MarginBottom(0)

	// Tree panel style
	treePanelStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#444444")).
			Padding(0, 1)

	// Details panel style
	detailsPanelStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color("#444444")).
				Padding(0, 1)

	// Header title style
	headerTitleStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("#FFFFFF"))

	durationBadgeStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#000000")).
				Background(lipgloss.Color("#87CEEB")).
				Padding(0, 1).
				MarginLeft(1)

	costBadgeStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#000000")).
			Background(lipgloss.Color("#98FB98")).
			Padding(0, 1).
			MarginLeft(1)

	// Title styles
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(primaryColor).
			MarginBottom(1)

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

	// Path style (for GCS/file paths)
	pathStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#87ceeb")).
			Italic(true)

	// Muted style for less important text
	mutedStyle = lipgloss.NewStyle().
			Foreground(mutedColor)

	// Error style
	errorStyle = lipgloss.NewStyle().
			Foreground(statusFailed).
			Bold(true)

	// Modal style
	modalStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(primaryColor).
			Padding(1, 2)

	// Modal label style (brighter for dark background)
	modalLabelStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#AAAAAA")).
			Bold(true)

	// Modal value style (bright for dark background)
	modalValueStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFFFFF"))

	// Modal path style (for GCS/file paths in modals)
	modalPathStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#87ceeb"))

	// Button styles for quick actions
	buttonStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#000000")).
			Background(primaryColor).
			Bold(true).
			Padding(0, 1)

	disabledButtonStyle = lipgloss.NewStyle().
				Foreground(mutedColor).
				Background(lipgloss.Color("#333333")).
				Padding(0, 1)

	// Temporary status message style
	temporaryStatusStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#FFFF00")).
				Bold(true)

	// Docker tag style (highlighted for visibility)
	tagStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#00FFAA")).
			Bold(true)

	// Breadcrumb styles - inline, no margins
	breadcrumbStyle = lipgloss.NewStyle().
			Foreground(mutedColor)

	breadcrumbSeparatorStyle = lipgloss.NewStyle().
					Foreground(lipgloss.Color("#555555"))

	breadcrumbActiveStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#FFFFFF")).
				Bold(true)

	// Section separator style
	sectionSeparatorStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#444444"))

	// Selected tree node style
	selectedStyle = lipgloss.NewStyle().
			Background(lipgloss.Color("#333333")).
			Foreground(lipgloss.Color("#FFFFFF")).
			Bold(true)
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
