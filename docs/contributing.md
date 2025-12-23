# Contributing

Thank you for your interest in contributing to Pumbaa! :heart:

---

## :wrench: Development Setup

### Prerequisites

- Go 1.21+
- Make

### Build

```bash
git clone https://github.com/lmtani/pumbaa
cd pumbaa
make build
```

!!! success "Binary Location"
    `./dist/pumbaa`

### Run

```bash
./dist/pumbaa --help
```

---

## :test_tube: Testing

```bash
make test
```

---

## :file_folder: Project Structure

```
pumbaa/
├── cmd/cli/            # CLI entry point
├── internal/
│   ├── application/    # Use cases
│   ├── container/      # Dependency injection
│   ├── domain/         # Domain entities
│   ├── infrastructure/ # External integrations (Cromwell, LLM, etc.)
│   └── interfaces/     # CLI & TUI handlers
└── pkg/wdl/            # WDL parser (ANTLR)
```

---

## :rocket: Pull Requests

1. :fork_and_knife: Fork the repository
2. :seedling: Create a feature branch
3. :pencil: Make your changes
4. :white_check_mark: Run tests: `make test`
5. :inbox_tray: Submit PR

---

## :bug: Reporting Bugs

Found a bug? [:material-github: Open an issue](https://github.com/lmtani/pumbaa/issues/new?template=bug_report.md){ .md-button }

Please include:

- A clear, descriptive title
- Steps to reproduce the issue
- Expected vs actual behavior
- Your environment (OS, Cromwell version, pumbaa version)
- Relevant logs or error messages

---

## :bulb: Requesting Features

Have an idea? [:material-github: Open a feature request](https://github.com/lmtani/pumbaa/issues/new?template=feature_request.md){ .md-button }

Please include:

- A clear description of the feature
- Why it would be useful
- Any examples or mockups if applicable

---

!!! info "All contributions welcome!"
    All contributions and feedback are welcome! Please ensure issues include enough details for us to investigate or implement your request.
