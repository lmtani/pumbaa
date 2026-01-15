// Package tui provides the terminal user interface for the application.
package tui

import (
	adkmodel "google.golang.org/adk/model"
	adksession "google.golang.org/adk/session"
	"google.golang.org/adk/tool"

	workflowapp "github.com/lmtani/pumbaa/internal/application/workflow"
	"github.com/lmtani/pumbaa/internal/domain/ports"
)

// Dependencies holds all shared dependencies for TUI screens.
// This centralizes dependency injection and eliminates duplication across handlers.
type Dependencies struct {
	// Core infrastructure
	Repository     ports.WorkflowRepository
	FileProvider   ports.FileProvider
	MetadataParser ports.MetadataParser

	// Use cases
	MonitoringUC *workflowapp.MonitoringUseCase
	BatchLogsUC  *workflowapp.GetBatchLogsUseCase

	// Chat dependencies (optional - nil if LLM not configured)
	ChatDeps *ChatDependencies
}

// ChatDependencies holds optional dependencies for chat functionality.
type ChatDependencies struct {
	LLM        adkmodel.LLM
	Tools      []tool.Tool
	SessionSvc adksession.Service
}

// HasChat returns true if chat dependencies are configured.
func (d *Dependencies) HasChat() bool {
	return d.ChatDeps != nil && d.ChatDeps.LLM != nil
}
