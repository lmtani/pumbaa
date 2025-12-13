package common

import "github.com/charmbracelet/lipgloss"

// Shared styles used across multiple screens
var (
	// Base styles
	BaseStyle = lipgloss.NewStyle().
			Foreground(TextColor)

	// Panel styles
	PanelStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(BorderColor).
			Padding(0, 1)

	FocusedPanelStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(FocusBorder).
				Padding(0, 1)

	// Title styles
	TitleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(PrimaryColor).
			MarginBottom(1)

	// Label styles
	LabelStyle = lipgloss.NewStyle().
			Foreground(MutedColor)

	ValueStyle = lipgloss.NewStyle().
			Foreground(TextColor)

	// Muted style for less important text
	MutedStyle = lipgloss.NewStyle().
			Foreground(MutedColor)

	// Error style
	ErrorStyle = lipgloss.NewStyle().
			Foreground(StatusFailed).
			Bold(true)

	// Warning style
	WarningStyle = lipgloss.NewStyle().
			Foreground(StatusRunning).
			Bold(true)

	// Success style
	SuccessStyle = lipgloss.NewStyle().
			Foreground(StatusSucceeded)

	// Help bar style
	HelpBarStyle = lipgloss.NewStyle().
			Foreground(MutedColor).
			BorderStyle(lipgloss.NormalBorder()).
			BorderTop(true).
			BorderForeground(BorderColor).
			Padding(0, 1)

	// Key style for key bindings
	KeyStyle = lipgloss.NewStyle().
			Foreground(PrimaryColor).
			Bold(true)

	// Description style for help
	DescStyle = lipgloss.NewStyle().
			Foreground(MutedColor)

	// Modal style
	ModalStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(PrimaryColor).
			Padding(1, 2)

	// Header style
	HeaderStyle = lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(PrimaryColor).
			Padding(0, 1).
			MarginBottom(0)

	// Header title style
	HeaderTitleStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("#FFFFFF"))

	// Badge styles
	BadgeStyle = lipgloss.NewStyle().
			Padding(0, 1).
			MarginLeft(1)

	DurationBadgeStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#000000")).
				Background(lipgloss.Color("#87CEEB")).
				Padding(0, 1).
				MarginLeft(1)

	CostBadgeStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#000000")).
			Background(lipgloss.Color("#98FB98")).
			Padding(0, 1).
			MarginLeft(1)
)

// StatusStyle returns the appropriate style for a status.
func StatusStyle(status string) lipgloss.Style {
	switch status {
	case "Done", "Succeeded":
		return SuccessStyle
	case "Failed":
		return ErrorStyle
	case "Running":
		return WarningStyle
	default:
		return MutedStyle
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
