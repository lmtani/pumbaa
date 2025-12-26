# Pumbaa

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
# Quick setup (interactive wizard)
pumbaa config init

# Privacy Note: Pumbaa collects anonymous usage statistics. 
# Opt-out: pumbaa config set telemetry_enabled false


# Or run directly with a provider
pumbaa chat --provider gemini --gemini-api-key <API_KEY>
pumbaa chat --provider vertex --vertex-project <PROJECT_ID>
pumbaa chat --provider ollama  # default, local
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
- `Tab` - Navigate between messages
- `y` - Copy selected message
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

### Quick Setup (Recommended)

Run the interactive configuration wizard:

```bash
pumbaa config init
```

This saves your settings to `~/.pumbaa/config.yaml`.

### Manual Configuration

**Cromwell Server:**
```bash
pumbaa config set cromwell_host http://cromwell:8000
# Or via environment variable
export CROMWELL_HOST=http://cromwell:8000
```

**Chat LLM Providers:**

```bash
# Gemini API (recommended for most users)
pumbaa config set llm_provider gemini
pumbaa config set gemini_api_key <your-api-key>

# Ollama (local, free)
pumbaa config set llm_provider ollama
pumbaa config set ollama_host http://localhost:11434

# Vertex AI (for GCP users)
pumbaa config set llm_provider vertex
pumbaa config set vertex_project <project-id>
```

**View current configuration:**
```bash
pumbaa config list
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

