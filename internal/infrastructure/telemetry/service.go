package telemetry

import "github.com/lmtani/pumbaa/internal/application/ports"

// The telemetry contract lives in ports so interface layers depend on it
// without importing infrastructure; the aliases below keep this package's
// API unchanged for the implementations and the composition root.

// Event represents a tracked user action.
type Event = ports.TelemetryEvent

// CommandContext holds context for command execution tracking.
type CommandContext = ports.TelemetryCommandContext

// Service defines the interface for telemetry collection.
type Service = ports.Telemetry
