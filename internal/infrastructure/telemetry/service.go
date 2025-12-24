package telemetry

import "time"

// Event represents a tracked user action
type Event struct {
	Command   string
	Duration  int64 // milliseconds
	Success   bool
	Error     string
	Version   string
	OS        string
	Arch      string
	Timestamp int64
}

// CommandContext holds context for command execution tracking
type CommandContext struct {
	AppName   string
	Args      []string
	StartTime time.Time
}

// Service defines the interface for telemetry collection
type Service interface {
	// Track captures an event. It should be non-blocking.
	Track(event Event)

	// TrackCommand tracks a command execution with automatic event creation.
	// It handles duration calculation, command name extraction, and error details.
	TrackCommand(ctx CommandContext, err error)

	// CaptureError captures an error with operation context.
	// Use this for errors that occur outside the normal command flow,
	// such as errors in TUI interactions or background operations.
	CaptureError(operation string, err error)

	// AddBreadcrumb logs an event that will appear as context when an error occurs.
	// Breadcrumbs create a trail of events leading up to an error.
	// Categories: "app", "navigation", "user", "http", "query"
	AddBreadcrumb(category, message string)

	// Close flushes any pending events.
	Close()
}
