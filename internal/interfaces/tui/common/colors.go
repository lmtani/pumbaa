package common

import "github.com/charmbracelet/lipgloss"

// Status colors
var (
	StatusSucceeded = lipgloss.Color("#00ff00")
	StatusFailed    = lipgloss.Color("#ff0000")
	StatusRunning   = lipgloss.Color("#ffff00")
	StatusPending   = lipgloss.Color("#888888")
)

// UI colors
var (
	PrimaryColor   = lipgloss.Color("#7D56F4")
	SecondaryColor = lipgloss.Color("#5A4FCF")
	BorderColor    = lipgloss.Color("#444444")
	FocusBorder    = lipgloss.Color("#7D56F4")
	TextColor      = lipgloss.Color("#FAFAFA")
	MutedColor     = lipgloss.Color("#888888")
	HighlightColor = lipgloss.Color("#874BFD")
)
