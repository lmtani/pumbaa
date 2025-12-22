# Pumbaa 🐗

A CLI tool for interacting with [Cromwell](https://cromwell.readthedocs.io/) workflow engine and WDL files.

## Installation

### Quick Install (Linux/macOS)

```bash
curl -sSL https://raw.githubusercontent.com/lmtani/pumbaa/main/install.sh | bash
```

### Manual Download

Download the latest binary from [GitHub Releases](https://github.com/lmtani/pumbaa/releases) for your platform.

## Usage

### Dashboard (Interactive TUI)

Browse and manage workflows with an interactive terminal interface:

```bash
pumbaa dashboard
```

### Chat Agent

Interactive AI assistant for querying workflows and downloading files:

```bash
# Using Ollama (default)
pumbaa chat

# Using Vertex AI
pumbaa chat --provider vertex --vertex-project <PROJECT_ID>
```

**Capabilities:**
- Query workflow status, metadata, outputs, and logs
- Download files from Google Cloud Storage
- Session persistence for conversation history

**Session Management:**
```bash
# List existing sessions
pumbaa chat --list

# Resume a session
pumbaa chat --session <SESSION_ID>
```

**Controls:**
- `Ctrl+D` - Send message
- `↑↓` - Scroll messages
- `Esc` - Exit

### Submit Workflow

```bash
pumbaa workflow submit \
  --workflow main.wdl \
  --inputs inputs.json \
  --options options.json
```

### Bundle WDL Dependencies

Package a WDL workflow with all its imports into a single ZIP file:

```bash
pumbaa bundle --workflow main.wdl --output <name>
# will create name.wdl and name.zip in the specified output path
```

## Configuration

### Cromwell Server

```bash
# Via flag
pumbaa --host http://cromwell:8000 dashboard

# Via environment variable
export CROMWELL_HOST=http://cromwell:8000
```

### Chat LLM Providers

**Ollama (default):**
```bash
export OLLAMA_HOST=http://localhost:11434
export OLLAMA_MODEL=llama3.2:3b
```

**Vertex AI:**
```bash
export VERTEX_PROJECT=<project-id>
export VERTEX_LOCATION=us-central1
export VERTEX_MODEL=gemini-2.0-flash
```

### Session Persistence

Sessions are stored in SQLite:
```bash
export PUMBAA_SESSION_DB=~/.pumbaa/sessions.db  # default
```

## Contributing

### Reporting Bugs

Found a bug? Please [open an issue](https://github.com/lmtani/pumbaa/issues/new?template=bug_report.md) with:

- A clear, descriptive title
- Steps to reproduce the issue
- Expected vs actual behavior
- Your environment (OS, Cromwell version, pumbaa version)
- Relevant logs or error messages

### Requesting Features

Have an idea for a new feature? [Open a feature request](https://github.com/lmtani/pumbaa/issues/new?template=feature_request.md) with:

- A clear description of the feature
- Why it would be useful
- Any examples or mockups if applicable

All contributions and feedback are welcome! Please ensure issues include enough details for us to investigate or implement your request.

