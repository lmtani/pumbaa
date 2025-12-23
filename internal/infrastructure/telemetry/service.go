package telemetry

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

// Service defines the interface for telemetry collection
type Service interface {
	// Track captures an event. It should be non-blocking.
	Track(event Event)
	// Close flushing any pending events.
	Close()
}
