package common

import (
	"strings"
	"time"
)

// ProgressBarWidth is the default width for progress bars
const ProgressBarWidth = 20

// RenderProgressBar renders a progress bar with the given percentage (0-100).
// Width specifies the number of characters for the bar.
func RenderProgressBar(percentage int, width int) string {
	if percentage < 0 {
		percentage = 0
	}
	if percentage > 100 {
		percentage = 100
	}
	if width <= 0 {
		width = ProgressBarWidth
	}

	filled := (percentage * width) / 100
	empty := width - filled

	return "[" + strings.Repeat("━", filled) + strings.Repeat("╌", empty) + "]"
}

// RenderLoadingProgress renders loading progress based on elapsed time.
// Returns a progress bar that fills up as time elapses.
func RenderLoadingProgress(elapsed time.Duration, expectedDuration time.Duration) string {
	if expectedDuration <= 0 {
		expectedDuration = 5 * time.Second
	}

	percentage := int(elapsed.Seconds() * 100 / expectedDuration.Seconds())
	if percentage > 100 {
		percentage = 100
	}

	return RenderProgressBar(percentage, ProgressBarWidth)
}

// RenderTimeoutProgress renders decreasing progress for timeout/expiration.
// Returns a progress bar that empties as time remaining decreases.
func RenderTimeoutProgress(remaining time.Duration, totalDuration time.Duration) string {
	if totalDuration <= 0 {
		totalDuration = 3 * time.Second
	}

	if remaining <= 0 {
		return RenderProgressBar(0, ProgressBarWidth)
	}

	percentage := int(remaining.Seconds() * 100 / totalDuration.Seconds())
	if percentage > 100 {
		percentage = 100
	}

	return RenderProgressBar(percentage, ProgressBarWidth)
}
