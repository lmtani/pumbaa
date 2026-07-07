package tools

import (
	"context"
	"fmt"
	"sort"

	"github.com/lmtani/pumbaa/internal/infrastructure/agents/tools/types"
)

// entry couples an action handler with its LLM-facing description.
type entry struct {
	handler     types.Handler
	description string
}

// ActionDoc describes a registered action for LLM-facing documentation.
type ActionDoc struct {
	Action      string
	Description string
}

// Registry manages the mapping between action names and their handlers.
// It provides a clean extension point for adding new actions without
// modifying existing code: the tool description shown to the LLM is
// generated from the registered actions and their descriptions.
type Registry struct {
	entries map[string]entry
}

// NewRegistry creates a new empty Registry.
func NewRegistry() *Registry {
	return &Registry{
		entries: make(map[string]entry),
	}
}

// Register adds a handler for the given action name. The description is
// surfaced to the LLM in the tool documentation, so state what the action
// does and which parameters it requires (e.g. "Required: workflow_id.").
// If a handler already exists for this action, it will be replaced.
func (r *Registry) Register(action, description string, handler types.Handler) {
	r.entries[action] = entry{handler: handler, description: description}
}

// RegisterFunc is a convenience method to register a function as a handler.
func (r *Registry) RegisterFunc(action, description string, fn types.HandlerFunc) {
	r.Register(action, description, fn)
}

// Get returns the handler for the given action.
// Returns nil and false if no handler is registered.
func (r *Registry) Get(action string) (types.Handler, bool) {
	e, ok := r.entries[action]
	if !ok {
		return nil, false
	}
	return e.handler, true
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
	actions := make([]string, 0, len(r.entries))
	for action := range r.entries {
		actions = append(actions, action)
	}
	sort.Strings(actions)
	return actions
}

// Docs returns the registered actions with their descriptions, sorted by
// action name. Used to generate the tool description shown to the LLM.
func (r *Registry) Docs() []ActionDoc {
	docs := make([]ActionDoc, 0, len(r.entries))
	for action, e := range r.entries {
		docs = append(docs, ActionDoc{Action: action, Description: e.description})
	}
	sort.Slice(docs, func(i, j int) bool { return docs[i].Action < docs[j].Action })
	return docs
}

// Count returns the number of registered handlers.
func (r *Registry) Count() int {
	return len(r.entries)
}
