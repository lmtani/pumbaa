package tools

import (
	"google.golang.org/adk/tool"
	"google.golang.org/adk/tool/toolconfirmation"
)

// noopToolContext is a minimal tool.Context for invoking function tools
// outside an ADK runner — the TUI chat drives its own agent loop instead of
// using the ADK Runner, which normally supplies the context.
//
// Only the methods touched by functiontool.Run's happy path are implemented:
// ToolConfirmation returning nil selects the no-confirmation branch (ADK
// v1.0.0 panics on a nil Context here). Any other method call hits the nil
// embedded interface and panics — functiontool recovers it into a tool error,
// so a future ADK version that needs more context fails loudly, not silently.
type noopToolContext struct{ tool.Context }

func (noopToolContext) ToolConfirmation() *toolconfirmation.ToolConfirmation { return nil }

// NoopToolContext returns the context to use when executing tools directly
// (outside an ADK runner).
func NoopToolContext() tool.Context { return noopToolContext{} }
