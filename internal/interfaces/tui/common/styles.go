package common

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

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
				Foreground(TextColor)

	// Badge styles
	BadgeStyle = lipgloss.NewStyle().
			Padding(0, 1).
			MarginLeft(1)

	DurationBadgeStyle = lipgloss.NewStyle().
				Foreground(BadgeFg).
				Background(BadgeInfoBg).
				Padding(0, 1).
				MarginLeft(1)

	CostBadgeStyle = lipgloss.NewStyle().
			Foreground(BadgeFg).
			Background(BadgeSuccessBg).
			Padding(0, 1).
			MarginLeft(1)

	// Breadcrumb styles
	BreadcrumbActiveStyle = lipgloss.NewStyle().
				Foreground(TextColor).
				Bold(true)

	BreadcrumbInactiveStyle = lipgloss.NewStyle().
				Foreground(MutedColor)

	BreadcrumbSeparatorStyle = lipgloss.NewStyle().
					Foreground(MutedColor)

	// Navigation hint style
	NavHintStyle = lipgloss.NewStyle().
			Foreground(MutedColor).
			Italic(true)
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

// StatusBadgeStyle returns a background-filled badge style for a status.
func StatusBadgeStyle(status string) lipgloss.Style {
	base := lipgloss.NewStyle().Foreground(BadgeFg).Padding(0, 1)
	switch status {
	case "Done", "Succeeded":
		return base.Background(BadgeSuccessBg)
	case "Failed":
		return base.Background(BadgeDangerBg)
	case "Running":
		return base.Background(BadgeWarnBg)
	default:
		return base.Background(BadgeNeutralBg)
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

// Screen represents a screen in the navigation hierarchy.
type Screen struct {
	Name   string
	Active bool
}

// RenderBreadcrumbs renders a breadcrumb navigation bar.
// Example: "Dashboard › Debug › Chat"
func RenderBreadcrumbs(screens []Screen) string {
	var parts []string
	separator := BreadcrumbSeparatorStyle.Render(" › ")

	for i, screen := range screens {
		var rendered string
		if screen.Active {
			rendered = BreadcrumbActiveStyle.Render(screen.Name)
		} else {
			rendered = BreadcrumbInactiveStyle.Render(screen.Name)
		}
		parts = append(parts, rendered)

		if i < len(screens)-1 {
			parts = append(parts, separator)
		}
	}

	return strings.Join(parts, "")
}

// RenderNavHints renders navigation hints for the current screen.
func RenderNavHints(canGoBack bool) string {
	if canGoBack {
		return NavHintStyle.Render("[ESC] back  [ctrl+c] quit")
	}
	return NavHintStyle.Render("[ESC] quit  [ctrl+c] quit")
}

// PlaceOverlay places a modal string centered on top of a background string.
func PlaceOverlay(_, _ int, modal, background string) string {
	return lipgloss.Place(
		lipgloss.Width(background),
		lipgloss.Height(background),
		lipgloss.Center,
		lipgloss.Center,
		modal,
		lipgloss.WithWhitespaceChars(" "),
		lipgloss.WithWhitespaceForeground(lipgloss.Color("0")),
	)
}
