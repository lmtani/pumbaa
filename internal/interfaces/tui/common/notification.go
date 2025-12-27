package common

import (
	"time"

	"github.com/charmbracelet/lipgloss"
)

// NotificationType defines the type of notification
type NotificationType int

const (
	// NotifyInfo is for informational messages
	NotifyInfo NotificationType = iota
	// NotifySuccess is for success messages
	NotifySuccess
	// NotifyWarning is for warning messages
	NotifyWarning
	// NotifyError is for error messages
	NotifyError
)

// Notification styles
var (
	infoStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#87CEEB"))

	successNotifyStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#98FB98"))

	warningNotifyStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#FFD700"))

	errorNotifyStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#FF6B6B"))

	temporaryStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFFFFF")).
			Bold(true)
)

// RenderNotification renders a styled notification message.
func RenderNotification(msg string, notifyType NotificationType) string {
	switch notifyType {
	case NotifySuccess:
		return successNotifyStyle.Render("✓ " + msg)
	case NotifyWarning:
		return warningNotifyStyle.Render("⚠ " + msg)
	case NotifyError:
		return errorNotifyStyle.Render("✗ " + msg)
	default:
		return infoStyle.Render("ℹ " + msg)
	}
}

// RenderStatusMessage renders a temporary status with optional progress bar.
// Shows progress bar if expiresAt is in the future.
func RenderStatusMessage(msg string, expiresAt time.Time, totalDuration time.Duration) string {
	remaining := time.Until(expiresAt)

	if remaining <= 0 {
		return temporaryStyle.Render(msg)
	}

	progressBar := " " + RenderTimeoutProgress(remaining, totalDuration)
	return temporaryStyle.Render(msg + progressBar)
}

// RenderLoadingMessage renders a loading message with progress bar.
func RenderLoadingMessage(msg string, startTime time.Time, expectedDuration time.Duration) string {
	elapsed := time.Since(startTime)
	progressBar := " " + RenderLoadingProgress(elapsed, expectedDuration)
	return temporaryStyle.Render("⏳ " + msg + progressBar)
}
