package common

import "github.com/charmbracelet/lipgloss"

// Single source of truth for every color in the TUI. View code must reference
// these tokens instead of hardcoding hex values, so the palette stays
// consistent and theme changes happen in one place.
//
// Colors are adaptive: lipgloss picks Light or Dark based on the terminal
// background. Exceptions are the badge/button colors, which paint their own
// background and therefore render the same everywhere.

// Status colors
var (
	StatusSucceeded = lipgloss.AdaptiveColor{Light: "#1A7F37", Dark: "#4EC9B0"}
	StatusFailed    = lipgloss.AdaptiveColor{Light: "#C0392B", Dark: "#F47067"}
	StatusRunning   = lipgloss.AdaptiveColor{Light: "#9A6700", Dark: "#E5C07B"}
	StatusPending   = lipgloss.AdaptiveColor{Light: "#6E6E6E", Dark: "#888888"}
)

// UI colors
var (
	PrimaryColor   = lipgloss.AdaptiveColor{Light: "#6B46C1", Dark: "#7D56F4"}
	SecondaryColor = lipgloss.AdaptiveColor{Light: "#4C3FA0", Dark: "#5A4FCF"}
	BorderColor    = lipgloss.AdaptiveColor{Light: "#C9C9C9", Dark: "#444444"}
	FocusBorder    = lipgloss.AdaptiveColor{Light: "#6B46C1", Dark: "#7D56F4"}
	TextColor      = lipgloss.AdaptiveColor{Light: "#1F1F1F", Dark: "#FAFAFA"}
	MutedColor     = lipgloss.AdaptiveColor{Light: "#767676", Dark: "#888888"}
	HighlightColor = lipgloss.AdaptiveColor{Light: "#E4DAFB", Dark: "#5B4B8A"} // selection background
	SubtleColor    = lipgloss.AdaptiveColor{Light: "#E8E8E8", Dark: "#333333"} // empty gauges, disabled backgrounds
	InfoColor      = lipgloss.AdaptiveColor{Light: "#0969DA", Dark: "#79B8E3"} // informational notes, file/GCS paths
	WarningColor   = lipgloss.AdaptiveColor{Light: "#BC4C00", Dark: "#F0883E"} // degraded, preempted, attention
	ErrorSoftColor = lipgloss.AdaptiveColor{Light: "#A94442", Dark: "#E8A19B"} // error message bodies (softer than StatusFailed)
)

// Badge and button colors — fixed on purpose: they sit on their own
// background, so they are readable regardless of the terminal theme.
var (
	BadgeFg        = lipgloss.Color("#1B1B1B")
	BadgeInfoBg    = lipgloss.Color("#8AB6D6") // counts, duration, provider
	BadgeSuccessBg = lipgloss.Color("#96C7A8") // cost, tokens
	BadgeWarnBg    = lipgloss.Color("#D9BC6A") // active filters, context, running status
	BadgeDangerBg  = lipgloss.Color("#D98C84") // update available, failed status
	BadgeSearchBg  = lipgloss.Color("#D9A05C") // search mode
	BadgeNeutralBg = lipgloss.Color("#A5A5A5") // unknown/pending status
	OnPrimaryColor = lipgloss.Color("#FFFFFF") // text on PrimaryColor backgrounds
)
