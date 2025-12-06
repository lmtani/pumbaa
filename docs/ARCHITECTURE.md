# Pumbaa - Cromwell CLI & WDL Tools

A command-line tool for interacting with the Cromwell workflow engine and working with WDL (Workflow Description Language) files.

## Architecture

This project follows **Clean Architecture** and **Domain-Driven Design (DDD)** principles:

```
.
├── cmd/
│   └── cli/
│       └── main.go              # Application entry point
│
├── internal/
│   ├── domain/                  # Domain Layer (Enterprise Business Rules)
│   │   ├── workflow/            # Workflow domain
│   │   │   ├── entity.go        # Workflow, Call, Status entities
│   │   │   ├── repository.go    # Repository interface (port)
│   │   │   └── errors.go        # Domain errors
│   │   └── bundle/              # Bundle domain
│   │       ├── entity.go        # Bundle entity
│   │       └── errors.go        # Domain errors
│   │
│   ├── application/             # Application Layer (Use Cases)
│   │   ├── workflow/
│   │   │   ├── submit/          # Submit workflow use case
│   │   │   ├── metadata/        # Get metadata use case
│   │   │   ├── abort/           # Abort workflow use case
│   │   │   └── query/           # Query workflows use case
│   │   └── bundle/
│   │       └── create/          # Create bundle use case
│   │
│   ├── infrastructure/          # Infrastructure Layer (Adapters)
│   │   └── cromwell/            # Cromwell API client
│   │       ├── client.go        # HTTP client implementation
│   │       ├── types.go         # API response types
│   │       └── mapper.go        # Response to domain mappers
│   │
│   ├── interfaces/              # Interface Layer (Controllers/Presenters)
│   │   └── cli/
│   │       ├── handler/         # CLI command handlers
│   │       │   ├── submit.go
│   │       │   ├── metadata.go
│   │       │   ├── abort.go
│   │       │   ├── query.go
│   │       │   └── bundle.go
│   │       └── presenter/       # Terminal output formatting
│   │           └── presenter.go
│   │
│   ├── config/                  # Configuration
│   │   └── config.go
│   │
│   └── container/               # Dependency Injection
│       └── container.go
│
└── pkg/
    └── wdl/                     # Public WDL parsing package
        ├── parser/              # ANTLR4 generated parser
        ├── ast/                 # AST definitions
        ├── visitor/             # Parse tree visitor
        ├── wdl.go               # Public API
        ├── analyzer.go          # Dependency analysis
        └── bundle.go            # Bundle creation
```

## Installation

```bash
go install github.com/lmtani/pumbaa/cmd/cli@latest
```

Or build from source:

```bash
go build -o pumbaa ./cmd/cli
```

## Usage

### Configuration

Set the Cromwell host via environment variable or flag:

```bash
export CROMWELL_HOST=http://localhost:8000
# or
pumbaa --host http://localhost:8000 <command>
```

### Workflow Commands

#### Submit a workflow

```bash
pumbaa workflow submit \
  --workflow workflow.wdl \
  --inputs inputs.json \
  --options options.json \
  --dependencies deps.zip \
  --label env=production
```

#### Get workflow metadata

```bash
pumbaa workflow metadata <workflow-id>
pumbaa workflow metadata <workflow-id> --verbose
```

#### Abort a workflow

```bash
pumbaa workflow abort <workflow-id>
```

#### Query workflows

```bash
pumbaa workflow query --name MyWorkflow --status Running --limit 10
```

### Bundle Commands

Create a WDL bundle with all dependencies:

```bash
pumbaa bundle --workflow main.wdl --output bundle.zip
```

## Design Principles

### Clean Architecture Layers

1. **Domain Layer** (`internal/domain/`): Contains enterprise business rules - entities, value objects, and repository interfaces. Has no external dependencies.

2. **Application Layer** (`internal/application/`): Contains application-specific business rules - use cases that orchestrate domain entities.

3. **Infrastructure Layer** (`internal/infrastructure/`): Contains implementations of interfaces defined in the domain layer - database, external APIs, etc.

4. **Interface Layer** (`internal/interfaces/`): Contains adapters for external interfaces - CLI handlers, HTTP controllers, etc.

### Dependency Rule

Dependencies flow inward:
- **Interface** → **Application** → **Domain** ← **Infrastructure**

The domain layer knows nothing about the outer layers. The infrastructure layer implements interfaces defined in the domain layer.

### Dependency Injection

The `container` package provides a simple DI container that wires all dependencies together at application startup.

## Development

### Adding a New Use Case

1. Create the use case in `internal/application/<domain>/<usecase>/`
2. Define `Input` and `Output` structs
3. Implement the `Execute` method
4. Add handler in `internal/interfaces/cli/handler/`
5. Wire dependencies in `internal/container/container.go`

### Adding a New Domain

1. Create entity in `internal/domain/<domain>/entity.go`
2. Define repository interface if needed
3. Add domain errors
4. Create use cases in application layer

## License

MIT
