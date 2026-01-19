// Package container provides dependency injection container.
package container

import (
	"os"

	"github.com/lmtani/pumbaa/internal/application/bundle"
	"github.com/lmtani/pumbaa/internal/application/workflow"
	"github.com/lmtani/pumbaa/internal/config"
	"github.com/lmtani/pumbaa/internal/infrastructure/cloudlogging"
	"github.com/lmtani/pumbaa/internal/infrastructure/cromwell"
	"github.com/lmtani/pumbaa/internal/infrastructure/metrics"
	"github.com/lmtani/pumbaa/internal/infrastructure/recommendation"
	"github.com/lmtani/pumbaa/internal/infrastructure/storage"
	"github.com/lmtani/pumbaa/internal/infrastructure/telemetry"
	wdlindexer "github.com/lmtani/pumbaa/internal/infrastructure/wdl"
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
	CloudLoggingRepo *cloudlogging.CloudLoggingRepository

	// Use cases
	SubmitUseCase                *workflow.SubmitUseCase
	MetadataUseCase              *workflow.MetadataUseCase
	AbortUseCase                 *workflow.AbortUseCase
	QueryUseCase                 *workflow.QueryUseCase
	OutputsUseCase               *workflow.OutputsUseCase
	InputsUseCase                *workflow.InputsUseCase
	MonitoringUseCase            *workflow.MonitoringUseCase
	ResourceReportUseCase        *workflow.ResourceReportUseCase
	BatchLogsUseCase             *workflow.GetBatchLogsUseCase
	BundleUseCase                *bundle.BundleUseCase
	ResourceVisualizationUseCase *workflow.ResourceVisualizationUseCase

	// Handlers
	SubmitHandler         *handler.SubmitHandler
	MetadataHandler       *handler.MetadataHandler
	AbortHandler          *handler.AbortHandler
	QueryHandler          *handler.QueryHandler
	OutputsHandler        *handler.OutputsHandler
	InputsHandler         *handler.InputsHandler
	ResourceReportHandler *handler.ResourceReportHandler
	BundleHandler         *handler.BundleHandler
	DebugHandler          *handler.DebugHandler
	DashboardHandler      *handler.DashboardHandler
	ChatHandler           *handler.ChatHandler
	ConfigHandler         *handler.ConfigHandler
	AnalyzeHandler        *handler.AnalyzeHandler
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
		ts := telemetry.NewCloudflareService(cfg.ClientID, version)
		if ts == nil {
			// Fallback to NoOp if failed or endpoint not configured
			c.TelemetryService = telemetry.NewNoOpService()
		} else {
			c.TelemetryService = ts
		}
	} else {
		c.TelemetryService = telemetry.NewNoOpService()
	}

	// Initialize infrastructure adapters
	c.CloudLoggingRepo = cloudlogging.NewCloudLoggingRepository()

	// Initialize use cases
	c.SubmitUseCase = workflow.NewSubmitUseCase(c.CromwellClient, fileProvider)
	c.MetadataUseCase = workflow.NewMetadataUseCase(c.CromwellClient)
	c.AbortUseCase = workflow.NewAbortUseCase(c.CromwellClient)
	c.QueryUseCase = workflow.NewQueryUseCase(c.CromwellClient)
	c.OutputsUseCase = workflow.NewOutputsUseCase(c.CromwellClient)
	c.InputsUseCase = workflow.NewInputsUseCase(c.CromwellClient)
	c.MonitoringUseCase = workflow.NewMonitoringUseCase(fileProvider)
	c.ResourceReportUseCase = workflow.NewResourceReportUseCase(c.CromwellClient, fileProvider)
	c.BatchLogsUseCase = workflow.NewGetBatchLogsUseCase(c.CloudLoggingRepo)
	c.BundleUseCase = bundle.New()

	// Initialize metrics reader for TSV files
	metricsReader := metrics.NewTSVReader()

	// Initialize LLM-based recommendation generator if LLM is configured
	var recommendationGenerator = recommendation.NewLLMGenerator(cfg, nil)
	if cfg.WDLDirectory != "" {
		// Try to initialize WDL indexer for better recommendations
		indexer, err := wdlindexer.NewIndexer(cfg.WDLDirectory, cfg.WDLIndexPath, false)
		if err == nil {
			recommendationGenerator = recommendation.NewLLMGenerator(cfg, indexer)
		}
	}
	c.ResourceVisualizationUseCase = workflow.NewResourceVisualizationUseCase(metricsReader, recommendationGenerator)

	// Initialize handlers
	c.SubmitHandler = handler.NewSubmitHandler(c.SubmitUseCase, c.Presenter)
	c.MetadataHandler = handler.NewMetadataHandler(c.MetadataUseCase, c.Presenter)
	c.AbortHandler = handler.NewAbortHandler(c.AbortUseCase, c.Presenter)
	c.QueryHandler = handler.NewQueryHandler(c.QueryUseCase, c.Presenter)
	c.OutputsHandler = handler.NewOutputsHandler(c.OutputsUseCase, c.Presenter)
	c.InputsHandler = handler.NewInputsHandler(c.InputsUseCase, c.Presenter)
	c.ResourceReportHandler = handler.NewResourceReportHandler(c.ResourceReportUseCase, c.Presenter)
	c.BundleHandler = handler.NewBundleHandler(c.BundleUseCase, c.Presenter)
	c.DebugHandler = handler.NewDebugHandler(c.CromwellClient, c.TelemetryService, c.MonitoringUseCase, fileProvider, c.CromwellClient, c.BatchLogsUseCase, c.Config)
	c.DashboardHandler = handler.NewDashboardHandler(c.CromwellClient, c.TelemetryService, c.MonitoringUseCase, fileProvider, c.CromwellClient, c.BatchLogsUseCase, c.Config, version)
	c.ChatHandler = handler.NewChatHandler(c.Config, c.TelemetryService)
	c.ConfigHandler = handler.NewConfigHandler()
	c.AnalyzeHandler = handler.NewAnalyzeHandler(c.ResourceVisualizationUseCase, c.Presenter)

	return c
}
