# Configuration

Pumbaa can be configured through the interactive wizard, command-line flags, environment variables, or a configuration file.

## Quick Setup (Recommended)

The easiest way to configure Pumbaa is using the interactive wizard:

```bash
pumbaa config init
```

This will guide you through:

1. Choosing your LLM provider (Ollama, Gemini API, or Vertex AI)
2. Provider-specific settings (API key, model, etc.)
3. Cromwell server URL
4. Optional WDL directory for chat context

Configuration is saved to `~/.pumbaa/config.yaml`.

## Configuration Commands

```bash
# Interactive setup wizard
pumbaa config init

# Set individual values
pumbaa config set <key> <value>

# Get a value
pumbaa config get <key>

# List all configuration
pumbaa config list

# Show config file path
pumbaa config path
```

### Available Keys

| Key | Description | Example |
|-----|-------------|---------|
| `llm_provider` | LLM backend | `ollama`, `gemini`, `vertex` |
| `cromwell_host` | Cromwell server URL | `http://localhost:8000` |
| `ollama_host` | Ollama server URL | `http://localhost:11434` |
| `ollama_model` | Ollama model name | `llama3.2:3b` |
| `gemini_api_key` | Gemini API key | `AIza...` |
| `gemini_model` | Gemini model | `gemini-2.0-flash` |
| `vertex_project` | GCP project ID | `my-project` |
| `vertex_location` | Vertex AI region | `us-central1` |
| `vertex_model` | Vertex AI model | `gemini-2.0-flash` |
| `wdl_directory` | WDL files for context | `/path/to/workflows` |

## Configuration Priority

Settings are applied in this order (later overrides earlier):

1. Default values
2. **Config file** (`~/.pumbaa/config.yaml`)
3. Environment variables
4. Command-line flags

## Cromwell Server

### Using Config Command
```bash
pumbaa config set cromwell_host http://cromwell.example.com:8000
```

### Using Environment Variable
```bash
export CROMWELL_HOST=http://cromwell.example.com:8000
```

### Using Command-Line Flag
```bash
pumbaa --host http://cromwell.example.com:8000 dashboard
```

!!! note "Default value"
    The default host is `http://localhost:8000`.

## Chat LLM Providers

### Gemini API (Recommended for Quick Start)

Get your API key at [Google AI Studio](https://aistudio.google.com/apikey).

```bash
pumbaa config set llm_provider gemini
pumbaa config set gemini_api_key <your-api-key>
```

### Ollama (Local, Free)

```bash
pumbaa config set llm_provider ollama
pumbaa config set ollama_host http://localhost:11434
pumbaa config set ollama_model llama3.2:3b
```

### Vertex AI (for GCP Users)

```bash
pumbaa config set llm_provider vertex
pumbaa config set vertex_project <project-id>
pumbaa config set vertex_location us-central1
```

## Authentication

Pumbaa assumes a direct connection to a reachable Cromwell server; it does not perform authentication itself.

If your Cromwell instance runs inside Kubernetes, expose it locally:

```bash
kubectl -n <namespace> port-forward svc/cromwell 8000:8000
pumbaa config set cromwell_host http://localhost:8000
```

## Next Steps

- [Quick Start](quick-start.md) - Run your first commands
- [Dashboard](../features/dashboard.md) - Learn about the interactive dashboard
- [Chat Agent](../features/chat.md) - Use AI to query workflows

