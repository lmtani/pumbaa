# Pumbaa

A CLI tool for interacting with [Cromwell](https://cromwell.readthedocs.io/) workflow engine and WDL files.


📖 **[Pumbaa Documentation](https://lmtani.github.io/pumbaa/)**

## Installation

```bash
curl -sSL https://raw.githubusercontent.com/lmtani/pumbaa/main/install.sh | bash
```

Or download from [GitHub Releases](https://github.com/lmtani/pumbaa/releases).

## Quick Start

```bash
pumbaa config init   # Interactive setup wizard
pumbaa dashboard     # Launch interactive TUI
```

## Development

### Prerequisites

- Go 1.25+

### Build

```bash
make build           # outputs dist/pumbaa
# or: go build -o pumbaa ./cmd/cli

# Run tests
make test
```

### Project Structure

```
cmd/cli/              # CLI entrypoint
internal/
  ├── domain/         # Business entities
  ├── application/    # Use cases and ports (hexagonal architecture)
  ├── infrastructure/ # External services (Cromwell, GCS, LLM)
  └── interfaces/     # CLI commands and TUI
pkg/wdl/              # WDL parser (ANTLR)
docs/                 # MkDocs documentation
```

See [ARCHITECTURE.md](docs/ARCHITECTURE.md) for details.

## Contributing

- 🐛 [Report bugs](https://github.com/lmtani/pumbaa/issues/new?template=bug_report.md)
- 💡 [Request features](https://github.com/lmtani/pumbaa/issues/new?template=feature_request.md)
- 📖 [Documentation](https://lmtani.github.io/pumbaa/contributing/)
