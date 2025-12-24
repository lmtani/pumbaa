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

	// Close flushes any pending events.
	Close()
}
