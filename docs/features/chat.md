# Chat Agent

Interactive AI assistant for querying Cromwell workflows and reading files via natural language.

## Usage

```bash
# Using Ollama (default)
pumbaa chat

# Using Vertex AI
pumbaa chat --provider vertex --vertex-project <PROJECT_ID>
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
| `↑↓` | Scroll messages |
| `Esc` | Exit |

## Configuration

### Ollama (default)

```bash
export OLLAMA_HOST=http://localhost:11434
export OLLAMA_MODEL=llama3.2:3b
```

### Vertex AI

```bash
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
