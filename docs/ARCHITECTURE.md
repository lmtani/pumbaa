# Pumbaa - Architecture Documentation

## Overview

Pumbaa is a CLI tool for interacting with the Cromwell workflow engine and WDL files. The project follows Clean Architecture principles with a clear separation between domain logic, application use cases, infrastructure implementations, and user interfaces.

## Core Features

- **Workflow Operations**: Submit, query, monitor, abort, and debug Cromwell workflows
- **Interactive TUI Dashboard**: Real-time workflow monitoring with filtering and status tracking
- **Debug TUI**: Visual tree navigation of workflow execution with call details and failure analysis
- **AI Chat Agent**: LLM-powered assistant for workflow analysis and troubleshooting (Gemini, Vertex AI, Ollama)
- **WDL Bundle Creation**: Package WDL workflows with dependencies into distributable ZIP files
- **WDL Indexing**: Fast search and discovery of WDL tasks and workflows
- **Resource Monitoring**: Analyze resource usage from monitoring logs
- **Telemetry**: Optional usage tracking with Sentry integration

## Project Structure

```
├── cmd/cli/                    # Application entry point
├── internal/                   # Private application code
│   ├── application/            # Use cases (Application Layer)
│   │   ├── bundle/create/      # WDL bundle creation
│   │   └── workflow/
│   │       ├── abort/          # Workflow abortion
│   │       ├── debuginfo/      # Debug info parsing and tree building
│   │       ├── metadata/       # Metadata retrieval
│   │       ├── monitoring/     # Resource usage analysis
│   │       ├── query/          # Workflow querying
│   │       └── submit/         # Workflow submission
│   ├── config/                 # Configuration management
│   ├── container/              # Dependency injection container
│   ├── domain/                 # Domain entities and interfaces (Domain Layer)
│   │   ├── bundle/             # Bundle entities
│   │   ├── ports/              # Port interfaces (Hexagonal Architecture)
│   │   ├── wdlindex/           # WDL index entities
│   │   └── workflow/           # Workflow entities
│   │       ├── monitoring/     # Resource monitoring domain
│   │       └── preemption/     # Preemption analysis
│   ├── infrastructure/         # External services adapters (Infrastructure Layer)
│   │   ├── chat/               # LLM integrations and agent tools
│   │   │   ├── agent/tools/    # Tool registry for chat agent
│   │   │   │   ├── cromwell/   # Cromwell API tools
│   │   │   │   ├── gcs/        # Google Cloud Storage tools
│   │   │   │   └── wdl/        # WDL knowledge base tools
│   │   │   └── llm/            # LLM providers (Gemini, Vertex, Ollama)
│   │   ├── cromwell/           # Cromwell API client
│   │   ├── session/            # SQLite session management
│   │   ├── storage/            # File storage (local and GCS)
│   │   ├── telemetry/          # Telemetry service (Sentry/NoOp)
│   │   └── wdl/                # WDL indexer implementation
│   └── interfaces/             # UI adapters (Interface Layer)
│       ├── cli/handler/        # CLI command handlers
│       ├── cli/presenter/      # Output formatters
│       └── tui/                # Terminal User Interfaces
│           ├── chat/           # Chat agent TUI
│           ├── configwizard/   # Configuration setup wizard
│           ├── dashboard/      # Workflow dashboard TUI
│           └── debug/          # Debug tree navigation TUI
└── pkg/wdl/                    # Public WDL parsing library
    ├── ast/                    # Abstract Syntax Tree
    ├── parser/                 # ANTLR-generated parser
    └── visitor/                # AST visitor
```

## Architecture Layers

### 1. Domain Layer (`internal/domain/`)

Contains business entities, value objects, and port interfaces. This layer has no dependencies on external frameworks or libraries.

**Packages:**

- **`ports/`**: **Port interfaces (Hexagonal Architecture)**
  - `WorkflowRepository` - Primary port for all workflow operations (execution, metadata, health, labels)
  - `FileProvider` - Port for file storage access (local and cloud)
  - `WDLRepository` - Port for WDL indexing operations

- **`workflow/`**: Core workflow entities (`Workflow`, `Call`, `Status`, `HealthStatus`), and errors
- **`workflow/monitoring/`**: Resource monitoring entities (`MonitoringMetrics`, `EfficiencyReport`) and usage statistics
- **`workflow/preemption/`**: Preemption detection and analysis logic
- **`bundle/`**: WDL bundle entities for packaging workflows
- **`wdlindex/`**: WDL index entities (`Index`, `IndexedTask`, `IndexedWorkflow`) for knowledge base

### 2. Application Layer (`internal/application/`)

Contains use cases that orchestrate domain logic. Each use case is a single business operation with a clear input and output.

**Use Cases:**

- **`workflow/submit/`**: Submit workflows to Cromwell with validation
- **`workflow/metadata/`**: Retrieve workflow execution metadata
- **`workflow/abort/`**: Abort running workflows
- **`workflow/query/`**: Query workflows with filters (status, name, dates)
- **`workflow/debuginfo/`**: Parse metadata and build execution trees for debugging
- **`workflow/monitoring/`**: Analyze resource usage from monitoring logs (CPU, memory, disk)
- **`bundle/create/`**: Create WDL bundles with dependency resolution

### 3. Infrastructure Layer (`internal/infrastructure/`)

Contains implementations of external services and adapters for domain interfaces.

**Implementations:**

- **`cromwell/`**: Cromwell REST API client implementing `ports.WorkflowRepository`
  - HTTP client with timeout configuration
  - JSON marshaling/unmarshaling
  - Error handling and status code mapping
  - Complete workflow lifecycle management

- **`chat/llm/`**: LLM provider implementations
  - `ollama/` - Local Ollama integration
  - `gemini.go` - Google Gemini API client
  - `vertex.go` - Google Vertex AI client
  - `factory.go` - LLM provider factory pattern

- **`chat/agent/tools/`**: Tool registry for AI agent
  - **`cromwell/`**: Query, status, metadata, logs, outputs tools
  - **`gcs/`**: Google Cloud Storage file download
  - **`wdl/`**: WDL search, list, and info tools
  - **`registry.go`**: Tool registration and schema generation

- **`wdl/`**: WDL indexer implementing `ports.WDLRepository`
  - File system traversal
  - ANTLR-based parsing
  - JSON cache persistence

- **`storage/`**: File provider for local and GCS paths
  - Implements `ports.FileProvider` interface
  - Size limits and validation

- **`session/`**: SQLite-based session storage for chat history
  - Uses Google ADK session interface
  - Persistent conversation state

- **`telemetry/`**: Usage tracking and error reporting
  - Sentry integration for production
  - NoOp implementation for privacy/development

### 4. Interface Layer (`internal/interfaces/`)

Contains adapters for user interaction. This layer depends on the application layer but not on infrastructure.

**CLI Handlers:**

- `submit`, `metadata`, `abort`, `query` - Standard workflow operations
- `bundle` - WDL packaging
- `debug`, `dashboard` - Launch TUI applications
- `chat` - Launch AI chat agent TUI
- `config` - Configuration management wizard

**TUI Applications (Bubble Tea framework):**

- **`dashboard/`**: Real-time workflow monitoring
  - Auto-refresh workflow list
  - Keyboard navigation and filtering
  - Status-based color coding
  - Direct debug mode launch

- **`debug/`**: Interactive debug explorer
  - Tree navigation of workflow calls
  - Call details with inputs/outputs
  - Failure analysis and logs display
  - Timeline visualization

- **`chat/`**: AI chat interface
  - Message streaming
  - Session persistence
  - Markdown rendering
  - Copy-to-clipboard support

- **`configwizard/`**: Interactive configuration setup
  - Directory picker
  - Provider selection
  - Validation and testing

**Presenters:**

- **`cli/presenter/`**: Format data for CLI output (JSON, table, plain text)

### 5. Configuration (`internal/config/`)

Centralized configuration management with file persistence and environment variable support.

**Config Fields:**
- Cromwell host and timeout
- LLM provider settings (Gemini, Vertex, Ollama)
- WDL directory and index path
- Session database path
- Telemetry toggle
- Client ID for anonymous tracking

### 6. Dependency Injection (`internal/container/`)

The container wires all dependencies together following the dependency inversion principle.

**Container Responsibilities:**
- Initialize infrastructure clients (Cromwell, Telemetry)
- Create use cases with injected repositories
- Build handlers with use cases and presenters
- Provide singleton instances for the application lifecycle

## Dependency Flow

```
┌──────────────────────────────────────────────────────┐
│                  Interface Layer                      │
│  (CLI Handlers, TUI Applications, Presenters)        │
└──────────────────┬───────────────────────────────────┘
                   │
                   ▼
┌──────────────────────────────────────────────────────┐
│                 Application Layer                     │
│           (Use Cases - Business Logic)                │
└──────────────────┬───────────────────────────────────┘
                   │
                   ▼
┌──────────────────────────────────────────────────────┐
│                   Domain Layer                        │
│      (Entities, Interfaces, Business Rules)           │
└──────────────────▲───────────────────────────────────┘
                   │
                   │ implements interfaces
                   │
┌──────────────────┴───────────────────────────────────┐
│              Infrastructure Layer                     │
│    (Cromwell Client, LLM Providers, Storage, etc)    │
└──────────────────────────────────────────────────────┘

          Container (wires everything together)
```

**Key Principles:**
- Dependencies point inward (toward domain)
- Domain has zero external dependencies
- Infrastructure implements domain interfaces
- Interface layer depends only on application layer

## Key Design Patterns

### 1. Ports & Adapters (Hexagonal Architecture)
Domain defines port interfaces in `domain/ports/`, implemented by adapters in `infrastructure/`. This inverts dependencies and allows business logic to remain independent of technical details.

### 2. Use Case Pattern
Each business operation is encapsulated in a use case with:
- Clear input/output DTOs
- Single responsibility
- No framework dependencies

### 3. Dependency Injection
The `container.Container` manages object lifecycle and dependency wiring using constructor injection.

### 4. Factory Pattern
- `llm.NewLLM()` creates appropriate LLM provider based on configuration
- `tools.GetAllTools()` builds tool registry from available components

### 5. Strategy Pattern
Different LLM providers implement a common interface, allowing runtime provider selection.

### 6. Adapter Pattern
Infrastructure adapters translate between external APIs and domain interfaces.

## Testing Strategy

- **Domain Layer**: Pure unit tests with no mocks
- **Application Layer**: Test use cases with mock repositories
- **Infrastructure Layer**: Integration tests with test servers/fixtures
- **Interface Layer**: UI component tests with mock use cases

## Known Architecture Considerations

### Current Simplifications

1. **debuginfo package** contains both parsing and tree building logic
   - **Reason**: Tree building is tightly coupled to metadata structure and not reused elsewhere
   - **Impact**: Slightly mixed responsibilities, but isolated to one package

2. **Submit UseCase** reads files directly with `os.ReadFile()`
   - **Reason**: Simple file operations don't warrant additional abstraction for current use cases
   - **Impact**: Harder to test, but file I/O is straightforward

3. **preemption package in domain**
   - **Reason**: Preemption detection is domain logic, even if it feels like analysis
   - **Impact**: Acceptable as it represents business rules

### Design Decisions

- **Centralized Ports Package**: All port interfaces are defined in `domain/ports/` following Hexagonal Architecture. This makes it easy to see all external dependencies and maintains a clear boundary between domain and infrastructure.

- **Unified WorkflowRepository**: Single comprehensive interface for all workflow operations (execution, metadata, health, labels, costs). This eliminates interface redundancy and provides a complete contract for workflow management. Both CLI use cases and TUI handlers use the same port.

- **FileProvider abstraction**: Single interface (`ports.FileProvider`) for file access, allowing implementations for local files, GCS, S3, or any other storage backend. Application layer depends only on the port.

- **WDLRepository**: Dedicated port for WDL indexing operations, separating concerns between workflow execution (Cromwell) and workflow discovery (WDL index).

- **Chat agent tools in infrastructure**: Tools are adapters to external services (Cromwell API, GCS, WDL files), implementing domain ports where appropriate.

- **Session management**: Delegates to Google ADK interfaces for compatibility with future storage backends.

- **Telemetry service**: Interface-based design allows NoOp implementation for privacy and development.

## CI/CD Pipeline

The project uses GitHub Actions for continuous integration and releases:

### Workflows

- **`ci.yml`**: Runs on PRs and pushes to `main`
  - Go linting and formatting checks
  - Unit and integration tests with coverage
  - Build verification for multiple platforms
  - Uploads coverage and build artifacts

- **`release.yml`**: Triggered on version tags (`v*`)
  - Runs full test suite
  - Uses GoReleaser to build multi-platform binaries
  - Creates GitHub releases with signed artifacts
  - Publishes binaries for Linux, macOS, Windows (amd64, arm64)

### Release Process

1. Tag a new version: `git tag v1.2.3 && git push origin v1.2.3`
2. GitHub Actions builds and tests
3. GoReleaser creates release with artifacts
4. Users download from GitHub Releases or use `install.sh` script

## External Dependencies

### Core Libraries
- **`urfave/cli/v2`**: CLI framework and command routing
- **`charmbracelet/bubbletea`**: TUI framework (Elm architecture)
- **`charmbracelet/lipgloss`**: TUI styling and layout
- **`antlr4`**: WDL parsing (via generated parser)

### Cloud & AI
- **`google.golang.org/genai`**: Gemini API client
- **`cloud.google.com/go/vertexai`**: Vertex AI client
- **`cloud.google.com/go/storage`**: GCS file access
- **Ollama**: Local LLM via HTTP API

### Infrastructure
- **`getsentry/sentry-go`**: Error tracking and telemetry
- **`google.golang.org/adk`**: Agent Development Kit (sessions, tools)
- **`mattn/go-sqlite3`**: Session storage

## Future Improvements

1. **File Reader Interface**: Inject file reading into Submit UseCase
2. **Workflow Events**: Domain events for workflow state changes
3. **Local Workflow Cache**: Reduce API calls with intelligent caching

---

**Last Updated**: 2025-01-25
**Document Version**: 2.0
**Project Version**: See [releases](https://github.com/lmtani/pumbaa/releases)
