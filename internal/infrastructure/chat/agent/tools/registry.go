package tools

import (
	"context"
	"fmt"
	"sort"

	"github.com/lmtani/pumbaa/internal/infrastructure/chat/agent/tools/types"
)

// Registry manages the mapping between action names and their handlers.
// It provides a clean extension point for adding new actions without modifying existing code.
type Registry struct {
	handlers map[string]types.Handler
}

// NewRegistry creates a new empty Registry.
func NewRegistry() *Registry {
	return &Registry{
		handlers: make(map[string]types.Handler),
	}
}

// Register adds a handler for the given action name.
// If a handler already exists for this action, it will be replaced.
func (r *Registry) Register(action string, handler types.Handler) {
	r.handlers[action] = handler
}

// RegisterFunc is a convenience method to register a function as a handler.
func (r *Registry) RegisterFunc(action string, fn types.HandlerFunc) {
	r.handlers[action] = fn
}

// Get returns the handler for the given action.
// Returns nil and false if no handler is registered.
func (r *Registry) Get(action string) (types.Handler, bool) {
	h, ok := r.handlers[action]
	return h, ok
}

// Handle dispatches the input to the appropriate handler based on the action.
// Returns an error output if no handler is found for the action.
func (r *Registry) Handle(ctx context.Context, input types.Input) (types.Output, error) {
	handler, ok := r.Get(input.Action)
	if !ok {
		return types.NewErrorOutput(input.Action, fmt.Sprintf(
			"unknown action: %s. Valid actions: %v",
			input.Action,
			r.Actions(),
		)), nil
	}
	return handler.Handle(ctx, input)
}

// Actions returns a sorted list of all registered action names.
func (r *Registry) Actions() []string {
	actions := make([]string, 0, len(r.handlers))
	for action := range r.handlers {
		actions = append(actions, action)
	}
	sort.Strings(actions)
	return actions
}

// Count returns the number of registered handlers.
func (r *Registry) Count() int {
	return len(r.handlers)
}
