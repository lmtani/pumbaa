// Package container provides dependency injection container.
package container

import (
	"os"

	"github.com/lmtani/pumbaa/internal/application/bundle"
	"github.com/lmtani/pumbaa/internal/application/workflow"
	"github.com/lmtani/pumbaa/internal/config"
	"github.com/lmtani/pumbaa/internal/infrastructure/cromwell"
	"github.com/lmtani/pumbaa/internal/infrastructure/storage"
	"github.com/lmtani/pumbaa/internal/infrastructure/telemetry"
	"github.com/lmtani/pumbaa/internal/interfaces/cli/handler"
	"github.com/lmtani/pumbaa/internal/interfaces/cli/presenter"
)

// Container holds all application dependencies.
type Container struct {
	Config    *config.Config
	Presenter *presenter.Presenter

	// Infrastructure
	CromwellClient   *cromwell.Client
	TelemetryService telemetry.Service

	// Use cases
	SubmitUseCase   *workflow.SubmitUseCase
	MetadataUseCase *workflow.MetadataUseCase
	AbortUseCase    *workflow.AbortUseCase
	QueryUseCase    *workflow.QueryUseCase
	BundleUseCase   *bundle.BundleUseCase

	// Handlers
	SubmitHandler    *handler.SubmitHandler
	MetadataHandler  *handler.MetadataHandler
	AbortHandler     *handler.AbortHandler
	QueryHandler     *handler.QueryHandler
	BundleHandler    *handler.BundleHandler
	DebugHandler     *handler.DebugHandler
	DashboardHandler *handler.DashboardHandler
	ChatHandler      *handler.ChatHandler
	ConfigHandler    *handler.ConfigHandler
}

// New creates a new dependency injection container.
func New(cfg *config.Config, version string) *Container {
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

	// Initialize FileProvider for file system access
	fileProvider := storage.NewFileProvider()

	// Initialize Telemetry
	if cfg.TelemetryEnabled {
		ts, err := telemetry.NewSentryService(cfg.ClientID, version)
		if err != nil || ts == nil {
			// Fallback to NoOp if failed or DSN not configured
			c.TelemetryService = telemetry.NewNoOpService()
		} else {
			c.TelemetryService = ts
		}
	} else {
		c.TelemetryService = telemetry.NewNoOpService()
	}

	// Initialize use cases
	c.SubmitUseCase = workflow.NewSubmitUseCase(c.CromwellClient, fileProvider)
	c.MetadataUseCase = workflow.NewMetadataUseCase(c.CromwellClient)
	c.AbortUseCase = workflow.NewAbortUseCase(c.CromwellClient)
	c.QueryUseCase = workflow.NewQueryUseCase(c.CromwellClient)
	c.BundleUseCase = bundle.New()

	// Initialize handlers
	c.SubmitHandler = handler.NewSubmitHandler(c.SubmitUseCase, c.Presenter)
	c.MetadataHandler = handler.NewMetadataHandler(c.MetadataUseCase, c.Presenter)
	c.AbortHandler = handler.NewAbortHandler(c.AbortUseCase, c.Presenter)
	c.QueryHandler = handler.NewQueryHandler(c.QueryUseCase, c.Presenter)
	c.BundleHandler = handler.NewBundleHandler(c.BundleUseCase, c.Presenter)
	c.DebugHandler = handler.NewDebugHandler(c.CromwellClient, c.TelemetryService)
	c.DashboardHandler = handler.NewDashboardHandler(c.CromwellClient, c.TelemetryService)
	c.ChatHandler = handler.NewChatHandler(c.Config, c.TelemetryService)
	c.ConfigHandler = handler.NewConfigHandler()

	return c
}
