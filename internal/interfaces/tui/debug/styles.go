package debug

import (
	"github.com/charmbracelet/lipgloss"

	"github.com/lmtani/pumbaa/internal/interfaces/tui/common"
)

// Styles - all styles use colors from common/ for consistency
var (
	// Header style
	headerStyle = lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(common.PrimaryColor).
			Padding(0, 1).
			MarginBottom(0)

	// Tree panel style
	treePanelStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(common.BorderColor).
			Padding(0, 1)

	// Details panel style
	detailsPanelStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(common.BorderColor).
				Padding(0, 1)

	// Header title style
	headerTitleStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("#FFFFFF"))

	durationBadgeStyle = common.DurationBadgeStyle

	costBadgeStyle = common.CostBadgeStyle

	// Title styles
	titleStyle = common.TitleStyle

	// Status styles - using common colors
	statusDoneStyle = lipgloss.NewStyle().
			Foreground(common.StatusSucceeded)

	statusFailedStyle = lipgloss.NewStyle().
				Foreground(common.StatusFailed)

	statusRunningStyle = lipgloss.NewStyle().
				Foreground(common.StatusRunning)

	statusPendingStyle = lipgloss.NewStyle().
				Foreground(common.StatusPending)

	// Label styles
	labelStyle = common.LabelStyle

	valueStyle = common.ValueStyle

	// Help bar style
	helpBarStyle = common.HelpBarStyle

	// Path style (for GCS/file paths)
	pathStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#87ceeb")).
			Italic(true)

	// Muted style for less important text
	mutedStyle = common.MutedStyle

	// Error style
	errorStyle = common.ErrorStyle

	// Modal style
	modalStyle = common.ModalStyle

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
			Background(common.PrimaryColor).
			Bold(true).
			Padding(0, 1)

	disabledButtonStyle = lipgloss.NewStyle().
				Foreground(common.MutedColor).
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
			Foreground(common.MutedColor)

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
// Delegates to common.StatusStyle for consistency across screens.
func StatusStyle(status string) lipgloss.Style {
	return common.StatusStyle(status)
}

// StatusIcon returns an icon for the status.
// Delegates to common.StatusIcon for consistency across screens.
func StatusIcon(status string) string {
	return common.StatusIcon(status)
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
