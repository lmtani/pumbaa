package chat

import (
	"github.com/charmbracelet/lipgloss"

	"github.com/lmtani/pumbaa/internal/interfaces/tui/common"
)

// Styles for chat
var (
	userStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#00BFFF")).
			Bold(true)

	agentStyle = lipgloss.NewStyle().
			Foreground(common.PrimaryColor).
			Bold(true)

	errorStyle = lipgloss.NewStyle().
			Foreground(common.StatusFailed).
			Bold(true)

	messageStyle = lipgloss.NewStyle().
			PaddingLeft(2).
			MarginBottom(1)

	inputStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(common.BorderColor).
			Padding(0, 1)

	focusedInputStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(common.FocusBorder).
				Padding(0, 1)

	// Style for selected message
	selectedMsgStyle = lipgloss.NewStyle().
				Background(lipgloss.Color("#3a3a3a")).
				Padding(0, 1)
)
