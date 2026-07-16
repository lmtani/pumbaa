// Package ports defines the interfaces for external dependencies (repositories, services).
// This file defines the pumbaa-specific extension of the ADK session service
// used by the chat interfaces.
package ports

import (
	"context"
	"time"
)

// DefaultChatAppName and DefaultChatUserID scope all Pumbaa chat sessions.
// Every session is stored and looked up under this single app/user pair.
const (
	DefaultChatAppName = "pumbaa"
	DefaultChatUserID  = "default"
)

// ChatSessionInfo summarizes a stored chat session.
type ChatSessionInfo struct {
	ID           string
	Summary      string
	ContextLabel string // Which workflow ▸ task the chat was opened for
	CreatedAt    time.Time
	UpdatedAt    time.Time
	InputTokens  int
	OutputTokens int
	EventCount   int
}

// ChatSessionStore extends the ADK session service with the pumbaa-specific
// queries used by the chat interfaces: session listing, resume-by-task,
// summaries and token accounting. Implementations also satisfy the ADK
// session.Service interface; callers holding a session.Service obtain this
// view via type assertion and degrade gracefully when it is absent.
type ChatSessionStore interface {
	ListWithSummaries(ctx context.Context, appName, userID string) ([]ChatSessionInfo, error)
	FindLatestByContextLabel(ctx context.Context, appName, userID, label string) (*ChatSessionInfo, error)
	SetContextLabel(ctx context.Context, sessionID, label string) error
	UpdateSummary(ctx context.Context, sessionID, summary string) error
	UpdateTokenUsage(ctx context.Context, sessionID string, inputTokens, outputTokens int) error
	Close() error
}
