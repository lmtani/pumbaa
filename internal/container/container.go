// Package container provides dependency injection container.
package container

import (
	"fmt"
	"os"

	"google.golang.org/adk/tool"

	"github.com/lmtani/pumbaa/internal/application/bundle"
	"github.com/lmtani/pumbaa/internal/application/ports"
	"github.com/lmtani/pumbaa/internal/application/workflow"
	"github.com/lmtani/pumbaa/internal/config"
	"github.com/lmtani/pumbaa/internal/infrastructure/agents/llm"
	"github.com/lmtani/pumbaa/internal/infrastructure/agents/tools"
	wdltools "github.com/lmtani/pumbaa/internal/infrastructure/agents/tools/wdl"
	"github.com/lmtani/pumbaa/internal/infrastructure/cloudlogging"
	"github.com/lmtani/pumbaa/internal/infrastructure/cromwell"
	"github.com/lmtani/pumbaa/internal/infrastructure/metrics"
	"github.com/lmtani/pumbaa/internal/infrastructure/recommendation"
	"github.com/lmtani/pumbaa/internal/infrastructure/session"
	"github.com/lmtani/pumbaa/internal/infrastructure/storage"
	"github.com/lmtani/pumbaa/internal/infrastructure/telemetry"
	"github.com/lmtani/pumbaa/internal/infrastructure/templates"
	"github.com/lmtani/pumbaa/internal/infrastructure/version"
	"github.com/lmtani/pumbaa/internal/infrastructure/wdlindexer"
	"github.com/lmtani/pumbaa/internal/interfaces/cli/handler"
	"github.com/lmtani/pumbaa/internal/interfaces/cli/presenter"
	"github.com/lmtani/pumbaa/internal/interfaces/tui"
)

// githubRepo is the GitHub repository used for release update checks.
const githubRepo = "lmtani/pumbaa"

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
	PreflightUseCase             *workflow.PreflightUseCase
	ScaffoldInputsUseCase        *workflow.ScaffoldInputsUseCase
	MetadataUseCase              *workflow.MetadataUseCase
	CompareUseCase               *workflow.CompareUseCase
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
	PreflightHandler      *handler.PreflightHandler
	ScaffoldHandler       *handler.ScaffoldHandler
	MetadataHandler       *handler.MetadataHandler
	DiffHandler           *handler.DiffHandler
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
func New(cfg *config.Config, appVersion string) *Container {
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
	metricsWriter := metrics.NewTSVWriter()
	fileSizeCache := storage.NewFileSizeCache()

	// Initialize Telemetry
	if cfg.TelemetryEnabled {
		ts := telemetry.NewCloudflareService(cfg.ClientID, appVersion)
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
	c.PreflightUseCase = workflow.NewPreflightUseCase(fileProvider, c.CromwellClient)
	c.ScaffoldInputsUseCase = workflow.NewScaffoldInputsUseCase(fileProvider)
	c.SubmitUseCase = workflow.NewSubmitUseCase(c.CromwellClient, fileProvider, c.PreflightUseCase)
	c.MetadataUseCase = workflow.NewMetadataUseCase(c.CromwellClient)
	c.CompareUseCase = workflow.NewCompareUseCase(c.CromwellClient)
	c.AbortUseCase = workflow.NewAbortUseCase(c.CromwellClient)
	c.QueryUseCase = workflow.NewQueryUseCase(c.CromwellClient)
	c.OutputsUseCase = workflow.NewOutputsUseCase(c.CromwellClient)
	c.InputsUseCase = workflow.NewInputsUseCase(c.CromwellClient)
	c.MonitoringUseCase = workflow.NewMonitoringUseCase(fileProvider)
	c.ResourceReportUseCase = workflow.NewResourceReportUseCase(c.CromwellClient, fileProvider, metricsWriter, fileSizeCache)
	c.BatchLogsUseCase = workflow.NewGetBatchLogsUseCase(c.CloudLoggingRepo)
	c.BundleUseCase = bundle.New()

	// Initialize metrics reader for TSV files
	metricsReader := metrics.NewTSVReader()

	// Initialize LLM-based recommendation generator if LLM is configured
	// The container creates the tools and passes them to the generator
	var wdlTools = tools.GetWDLOnlyTools(nil)
	if cfg.WDLDirectory != "" {
		// Try to initialize WDL indexer for better recommendations
		indexer, err := wdlindexer.NewIndexer(cfg.WDLDirectory, cfg.WDLIndexPath, false)
		if err == nil {
			wdlTools = tools.GetWDLOnlyTools(indexer)
		}
	}
	recommendationGenerator := recommendation.NewLLMGenerator(cfg, wdlTools)
	llmDebugWriterFactory := func(path string) (ports.LLMDebugWriter, error) {
		return recommendation.NewFileDebugWriter(path)
	}
	c.ResourceVisualizationUseCase = workflow.NewResourceVisualizationUseCase(metricsReader, recommendationGenerator, templates.NewHTMLRenderer(), llmDebugWriterFactory)

	// Initialize handlers
	c.SubmitHandler = handler.NewSubmitHandler(c.SubmitUseCase, c.Presenter)
	c.PreflightHandler = handler.NewPreflightHandler(c.PreflightUseCase, c.Presenter)
	c.ScaffoldHandler = handler.NewScaffoldHandler(c.ScaffoldInputsUseCase, c.Presenter)
	c.MetadataHandler = handler.NewMetadataHandler(c.MetadataUseCase, c.Presenter)
	c.DiffHandler = handler.NewDiffHandler(c.CompareUseCase, c.Presenter)
	c.AbortHandler = handler.NewAbortHandler(c.AbortUseCase, c.Presenter)
	c.QueryHandler = handler.NewQueryHandler(c.QueryUseCase, c.Presenter)
	c.OutputsHandler = handler.NewOutputsHandler(c.OutputsUseCase, c.Presenter)
	c.InputsHandler = handler.NewInputsHandler(c.InputsUseCase, c.Presenter)
	c.ResourceReportHandler = handler.NewResourceReportHandler(c.ResourceReportUseCase, c.Presenter)
	c.BundleHandler = handler.NewBundleHandler(c.BundleUseCase, c.Presenter)
	c.DebugHandler = handler.NewDebugHandler(c.CromwellClient, c.TelemetryService, c.MonitoringUseCase, fileProvider, c.BatchLogsUseCase, c.ChatDependencies)
	c.DashboardHandler = handler.NewDashboardHandler(c.CromwellClient, c.TelemetryService, c.MonitoringUseCase, fileProvider, c.BatchLogsUseCase, c.CompareUseCase, version.NewGitHubChecker(githubRepo), appVersion, c.ChatDependencies)
	c.ChatHandler = handler.NewChatHandler(c.Config, c.TelemetryService, c.ChatDependencies, c.SessionStore)
	c.ConfigHandler = handler.NewConfigHandler()
	c.AnalyzeHandler = handler.NewAnalyzeHandler(c.ResourceVisualizationUseCase, c.Presenter)

	return c
}

// SessionStore opens the SQLite chat session store. It does not require an
// LLM, so session listing works even when chat is not configured. Callers
// own Close.
func (c *Container) SessionStore() (ports.ChatSessionStore, error) {
	svc, err := session.NewSQLiteService(c.Config.SessionDBPath)
	if err != nil {
		return nil, err
	}
	return svc, nil
}

// initWDLRepository builds the WDL index repository from config. Returns nil
// (WDL actions disabled) when no directory is configured or indexing fails.
// The index is cached at cfg.WDLIndexPath, so subsequent startups are instant.
func (c *Container) initWDLRepository(forceRebuild bool) wdltools.Repository {
	if c.Config.WDLDirectory == "" {
		return nil
	}
	indexer, err := wdlindexer.NewIndexer(c.Config.WDLDirectory, c.Config.WDLIndexPath, forceRebuild)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: WDL tools disabled - indexing failed: %v\n", err)
		return nil
	}
	if idx, err := indexer.List(); err == nil {
		fmt.Printf("WDL index: %d tasks, %d workflows\n", len(idx.Tasks), len(idx.Workflows))
	}
	return indexer
}

// ChatDependencies builds the chat dependency bundle (LLM, agent tools,
// session service) shared by the standalone chat command and the TUI screens.
// It returns (nil, nil) when no LLM provider is configured — chat features
// are simply disabled.
//
// extraTools is the extension point for adding standalone ADK tools to the
// chat agent beyond the built-in pumbaa tool; see the tools package docs.
func (c *Container) ChatDependencies(rebuildWDLIndex bool, extraTools ...tool.Tool) (*tui.ChatDependencies, error) {
	if c.Config.LLMProvider == "" {
		return nil, nil
	}
	llmModel, err := llm.NewLLM(c.Config)
	if err != nil {
		return nil, fmt.Errorf("LLM initialization failed: %w", err)
	}
	svc, err := session.NewSQLiteService(c.Config.SessionDBPath)
	if err != nil {
		return nil, fmt.Errorf("session service initialization failed: %w", err)
	}
	agentTools := tools.GetAllTools(tools.Deps{
		Repo:         c.CromwellClient,
		Fetcher:      c.CromwellClient,
		WDLRepo:      c.initWDLRepository(rebuildWDLIndex),
		FileProvider: storage.NewFileProvider(),
	}, extraTools...)
	return &tui.ChatDependencies{LLM: llmModel, Tools: agentTools, SessionSvc: svc}, nil
}
