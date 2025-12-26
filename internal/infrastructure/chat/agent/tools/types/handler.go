package types

import "context"

// Handler defines the interface for handlers that process specific actions.
// Each handler is responsible for a single action, following the Single Responsibility Principle.
type Handler interface {
	// Handle processes the input and returns the output.
	// The context should be used for cancellation and timeouts.
	Handle(ctx context.Context, input Input) (Output, error)
}

// HandlerFunc is an adapter that allows using ordinary functions as Handler.
type HandlerFunc func(ctx context.Context, input Input) (Output, error)

// Handle implements Handler interface.
func (f HandlerFunc) Handle(ctx context.Context, input Input) (Output, error) {
	return f(ctx, input)
}
