// Package container provides dependency injection container.
package container

import (
	"os"

	"github.com/lmtani/pumbaa/internal/application/bundle/create"
	"github.com/lmtani/pumbaa/internal/application/workflow/abort"
	"github.com/lmtani/pumbaa/internal/application/workflow/metadata"
	"github.com/lmtani/pumbaa/internal/application/workflow/query"
	"github.com/lmtani/pumbaa/internal/application/workflow/submit"
	"github.com/lmtani/pumbaa/internal/config"
	"github.com/lmtani/pumbaa/internal/infrastructure/cromwell"
	"github.com/lmtani/pumbaa/internal/interfaces/cli/handler"
	"github.com/lmtani/pumbaa/internal/interfaces/cli/presenter"
)

// Container holds all application dependencies.
type Container struct {
	Config    *config.Config
	Presenter *presenter.Presenter

	// Infrastructure
	CromwellClient *cromwell.Client

	// Use cases
	SubmitUseCase   *submit.UseCase
	MetadataUseCase *metadata.UseCase
	AbortUseCase    *abort.UseCase
	QueryUseCase    *query.UseCase
	BundleUseCase   *create.UseCase

	// Handlers
	SubmitHandler    *handler.SubmitHandler
	MetadataHandler  *handler.MetadataHandler
	AbortHandler     *handler.AbortHandler
	QueryHandler     *handler.QueryHandler
	BundleHandler    *handler.BundleHandler
	DebugHandler     *handler.DebugHandler
	DashboardHandler *handler.DashboardHandler
	ChatHandler      *handler.ChatHandler
	AgentTestHandler *handler.AgentTestHandler
}

// New creates a new dependency injection container.
func New(cfg *config.Config) *Container {
	c := &Container{
		Config: cfg,
	}

	// Initialize presenter
	c.Presenter = presenter.New(os.Stdout)

	// Initialize infrastructure
	c.CromwellClient = cromwell.NewClient(cromwell.Config{
		Host:    cfg.CromwellHost,
		Timeout: cfg.CromwellTimeout,
	})

	// Initialize use cases
	c.SubmitUseCase = submit.New(c.CromwellClient)
	c.MetadataUseCase = metadata.New(c.CromwellClient)
	c.AbortUseCase = abort.New(c.CromwellClient)
	c.QueryUseCase = query.New(c.CromwellClient)
	c.BundleUseCase = create.New()

	// Initialize handlers
	c.SubmitHandler = handler.NewSubmitHandler(c.SubmitUseCase, c.Presenter)
	c.MetadataHandler = handler.NewMetadataHandler(c.MetadataUseCase, c.Presenter)
	c.AbortHandler = handler.NewAbortHandler(c.AbortUseCase, c.Presenter)
	c.QueryHandler = handler.NewQueryHandler(c.QueryUseCase, c.Presenter)
	c.BundleHandler = handler.NewBundleHandler(c.BundleUseCase, c.Presenter)
	c.DebugHandler = handler.NewDebugHandler(c.CromwellClient)
	c.DashboardHandler = handler.NewDashboardHandler(c.CromwellClient)
	c.ChatHandler = handler.NewChatHandler(c.Config)
	c.AgentTestHandler = handler.NewAgentTestHandler(c.Config)

	return c
}
