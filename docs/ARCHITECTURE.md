# Pumbaa - Architecture Documentation

## Overview

Pumbaa is a CLI tool for interacting with the Cromwell workflow engine and WDL files. The project follows Clean Architecture principles with a layered structure.

## Project Structure

```
├── cmd/cli/              # Application entry point
├── internal/             # Private application code
│   ├── application/      # Use cases (Application Layer)
│   ├── config/           # Configuration management
│   ├── container/        # Dependency injection container
│   ├── domain/           # Domain entities and interfaces (Domain Layer)
│   ├── infrastructure/   # External services adapters (Infrastructure Layer)
│   └── interfaces/       # UI adapters - CLI and TUI (Interface Layer)
└── pkg/wdl/              # Reusable WDL parsing library
```

## Architecture Layers

### Domain Layer (`internal/domain/`)

Contains business entities and repository interfaces (ports).

- **`workflow/`**: Workflow entities (`Workflow`, `Call`, `Status`), repository interface, and errors
- **`bundle/`**: Bundle entities for WDL packaging

### Application Layer (`internal/application/`)

Contains use cases that orchestrate domain logic.

- **`workflow/submit/`**: Workflow submission use case
- **`workflow/metadata/`**: Metadata retrieval use case
- **`workflow/abort/`**: Workflow abortion use case
- **`workflow/query/`**: Workflow query use case
- **`workflow/debuginfo/`**: Debug information parsing and tree building
- **`bundle/create/`**: WDL bundle creation use case

### Infrastructure Layer (`internal/infrastructure/`)

Contains implementations of external services.

- **`cromwell/`**: Cromwell API client implementing `workflow.Repository`

### Interface Layer (`internal/interfaces/`)

Contains adapters for user interaction.

- **`cli/handler/`**: CLI command handlers
- **`cli/presenter/`**: Output formatters
- **`tui/dashboard/`**: Interactive workflow TUI dashboard
- **`tui/debug/`**: Interactive debug TUI

### Shared Library (`pkg/wdl/`)

Reusable WDL parsing library with ANTLR-generated parser.

- **`ast/`**: Abstract Syntax Tree definitions
- **`parser/`**: ANTLR-generated lexer/parser
- **`visitor/`**: AST visitor implementation

## Dependency Flow

```
interfaces → application → domain ← infrastructure
     ↓            ↓           ↑
  container (wires everything together)
```

## Key Patterns

1. **Repository Pattern**: `workflow.Repository` interface in domain, implemented by `cromwell.Client`
2. **Use Case Pattern**: Each operation is a separate use case with `Execute()` method
3. **Dependency Injection**: `container.Container` wires all dependencies

---

## TODO: Architecture Violations

### 1. Handler depends on Infrastructure directly

**Location**: `internal/interfaces/cli/handler/debug.go`, `dashboard.go`

**Issue**: `DebugHandler` and `DashboardHandler` receive `*cromwell.Client` directly instead of an interface.

```go
// Current (violates DIP)
type DebugHandler struct {
    client *cromwell.Client
}

// Should be
type DebugHandler struct {
    repo workflow.Repository  // or a specific interface
}
```

**Impact**: Cannot test handlers in isolation, tight coupling to Cromwell implementation.

---

### 2. Missing Use Case for Debug/Dashboard

**Location**: `internal/interfaces/cli/handler/debug.go`, `dashboard.go`

**Issue**: These handlers call `cromwell.Client` methods directly, bypassing the application layer. Other handlers (submit, metadata, abort, query) correctly use use cases.

**Impact**: Business logic leaks into interface layer, inconsistent architecture.

---

### 3. debuginfo package has mixed responsibilities

**Location**: `internal/application/workflow/debuginfo/`

**Issue**: This package contains:
- Metadata parsing (should be infrastructure or domain)
- Tree building (presentation logic, should be in TUI layer)
- Types that duplicate domain entities

**Impact**: Violates Single Responsibility Principle, hard to test and maintain.

---

### 4. TUI depends on debuginfo types directly

**Location**: `internal/interfaces/tui/debug/types.go`

**Issue**: TUI uses type aliases from `debuginfo` package:
```go
type TreeNode = debuginfo.TreeNode
type CallDetails = debuginfo.CallDetails
```

**Impact**: Application layer types leak into interface layer. Should use dedicated view models.

---

### 5. Container exposes concrete types

**Location**: `internal/container/container.go`

**Issue**: Container exposes `*cromwell.Client` instead of `workflow.Repository` interface.

```go
// Current
CromwellClient *cromwell.Client

// Should be
WorkflowRepo workflow.Repository
```

**Impact**: Handlers can bypass the interface and use concrete implementation details.

---

### 6. preemption package location

**Location**: `internal/domain/workflow/preemption/`

**Issue**: Contains analysis logic that might be better suited in application layer, as it's not a core domain concept but rather a derived analysis.

**Impact**: Domain layer contains application-level logic.

---

### 7. Submit UseCase reads files directly

**Location**: `internal/application/workflow/submit/usecase.go`

**Issue**: Use case calls `os.ReadFile()` directly instead of receiving file contents.

```go
// Current
workflowSource, err := os.ReadFile(input.WorkflowFile)

// Should receive content or use a FileReader interface
```

**Impact**: Hard to test, use case has I/O responsibilities.

---

## Recommended Refactoring Priority

1. **High**: Create interfaces for DebugHandler/DashboardHandler dependencies
2. **High**: Create proper use cases for debug and dashboard features
3. **Medium**: Move tree building logic to TUI layer, keep only parsing in debuginfo
4. **Medium**: Use interfaces in Container instead of concrete types
5. **Low**: Refactor Submit UseCase to not read files directly
6. **Low**: Review preemption package location

---

## CI Migration: GitHub Actions

The project has been migrated from CircleCI to GitHub Actions. The previous CircleCI configuration (which uploaded coverage to Codecov) was removed and replaced by two GitHub Actions workflows:

- `ci.yml` — Runs tests and builds on PRs / pushes to `main`, uploads the build and coverage artifacts to Actions artifacts (no external Codecov upload).
- `release.yml` — Triggered on pushed Git tags (v*). Runs tests and then uses Goreleaser to create a release and attach signed build artifacts.

Goreleaser uses the repo's `.goreleaser.yml` to generate binaries for supported platforms. The release workflow uses the `GITHUB_TOKEN` secret (automatically provided by GitHub Actions) to create releases.

