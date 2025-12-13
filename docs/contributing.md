# Contributing

## Development Setup

### Prerequisites

- Go 1.21+
- Make

### Build

```bash
git clone https://github.com/lmtani/pumbaa
cd pumbaa
make build
```

Binary: `./build/pumbaa`

### Run

```bash
./build/pumbaa --help
```

## Testing

```bash
make test
```

## Project Structure

```
pumbaa/
├── cmd/cli/           # CLI entry point
├── internal/
│   ├── application/   # Use cases
│   ├── domain/        # Domain entities
│   ├── infrastructure/# External integrations
│   └── interfaces/    # CLI & TUI handlers
└── pkg/wdl/          # WDL parser
```

## Pull Requests

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Run tests: `make test`
5. Submit PR

## Reporting Issues

Use GitHub Issues:
- Bug reports: Use bug report template
- Feature requests: Use feature request template
