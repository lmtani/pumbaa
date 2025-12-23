# Chat Agent

Interactive AI assistant for querying Cromwell workflows and reading files via natural language.

## Quick Start

```bash
# Run the configuration wizard (recommended for first time)
pumbaa config init

# Or start chatting directly
pumbaa chat
```

## Usage with Different Providers

```bash
# Gemini API (requires API key from https://aistudio.google.com/apikey)
pumbaa chat --provider gemini --gemini-api-key <API_KEY>

# Vertex AI (requires GCP project)
pumbaa chat --provider vertex --vertex-project <PROJECT_ID>

# Ollama (local, free - default)
pumbaa chat --provider ollama
```

## Capabilities

The chat agent can:

- **Query workflows** - Search by status, name, or labels
- **Get workflow status** - Check execution state
- **View metadata** - Inspect workflow details and call information
- **Get outputs** - List workflow output files
- **View logs** - Access workflow log paths
- **Read files** - Fetch files from Google Cloud Storage

## Session Management

Conversations are persisted in SQLite for context retention.

```bash
# List existing sessions
pumbaa chat --list

# Resume a session
pumbaa chat --session <SESSION_ID>
```

## Controls

| Key | Action |
|-----|--------|
| `Ctrl+D` | Send message |
| `↑↓` | Scroll messages (when typing) |
| `Tab` | Switch to message navigation mode |
| `↑↓` | Navigate messages (in navigation mode) |
| `y` | Copy selected message to clipboard |
| `Tab` | Return to typing mode |
| `Esc` | Exit |

## Configuration

### Using Config Wizard (Recommended)

```bash
pumbaa config init
```

### Using Environment Variables

**Gemini API:**
```bash
export PUMBAA_LLM_PROVIDER=gemini
export GEMINI_API_KEY=<your-api-key>
export GEMINI_MODEL=gemini-2.0-flash
```

**Ollama (default):**
```bash
export PUMBAA_LLM_PROVIDER=ollama
export OLLAMA_HOST=http://localhost:11434
export OLLAMA_MODEL=llama3.2:3b
```

**Vertex AI:**
```bash
export PUMBAA_LLM_PROVIDER=vertex
export VERTEX_PROJECT=<project-id>
export VERTEX_LOCATION=us-central1
export VERTEX_MODEL=gemini-2.0-flash
```

### Session Storage

```bash
export PUMBAA_SESSION_DB=~/.pumbaa/sessions.db
```

## Example Prompts

- "List my running workflows"
- "What is the status of workflow abc-123?"
- "Show me the outputs of workflow xyz-456"
- "Read gs://bucket/path/to/file.txt"
- "Query last 5 failed workflows"

