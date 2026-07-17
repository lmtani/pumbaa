package chat

import (
	"github.com/charmbracelet/lipgloss"

	"github.com/lmtani/pumbaa/internal/interfaces/tui/common"
)

// Lipgloss styles shared by the chat views.
// Styles for chat
var (
	userStyle = lipgloss.NewStyle().
			Foreground(common.InfoColor).
			Bold(true)

	contextBadgeStyle = lipgloss.NewStyle().
				Foreground(common.BadgeFg).
				Background(common.BadgeWarnBg).
				Padding(0, 1)

	llmBadgeStyle = lipgloss.NewStyle().
			Foreground(common.BadgeFg).
			Background(common.BadgeInfoBg).
			Padding(0, 1)

	tokenBadgeStyle = lipgloss.NewStyle().
			Foreground(common.BadgeFg).
			Background(common.BadgeSuccessBg).
			Padding(0, 1)

	sessionSummaryStyle = lipgloss.NewStyle().
				Foreground(common.MutedColor).
				Italic(true)

	agentStyle = lipgloss.NewStyle().
			Foreground(common.PrimaryColor).
			Bold(true)

	errorStyle = lipgloss.NewStyle().
			Foreground(common.StatusFailed).
			Bold(true)

	messageStyle = lipgloss.NewStyle().
			PaddingLeft(2).
			MarginBottom(1)

	infoStyle = common.MutedStyle.
			Bold(true)

	infoMessageStyle = messageStyle.
				Foreground(common.MutedColor)

	inputStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(common.BorderColor).
			Padding(0, 1)

	focusedInputStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(common.FocusBorder).
				Padding(0, 1)
)
