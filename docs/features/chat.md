# Chat Agent

Interactive AI assistant for querying Cromwell workflows and reading files via natural language.

## Quick Start

=== "Using Config Wizard"

    ```bash
    pumbaa config init
    ```
    
    The wizard will guide you through setting up your preferred LLM provider.

=== "Direct Start"

    ```bash
    pumbaa chat
    ```

## :robot: LLM Providers

Choose your preferred AI backend:

=== ":material-google: Gemini API"

    ```bash
    pumbaa chat --provider gemini --gemini-api-key <API_KEY>
    ```
    
    !!! tip "Get an API Key"
        Free tier available at [Google AI Studio](https://aistudio.google.com/apikey)

=== ":material-cloud: Vertex AI"

    ```bash
    pumbaa chat --provider vertex --vertex-project <PROJECT_ID>
    ```
    
    !!! note "GCP Project Required"
        Requires an active Google Cloud project with Vertex AI API enabled.

=== ":material-server: Ollama (Local)"

    ```bash
    pumbaa chat --provider ollama
    ```
    
    !!! info "Default Option"
        Runs locally, no API key needed. Install from [ollama.ai](https://ollama.ai)

---

## :sparkles: Capabilities

<div class="grid cards" markdown>

-   :material-magnify: **Query Workflows**
    
    Search by status, name, or labels

-   :material-check-circle: **Get Status**
    
    Check execution state in real-time

-   :material-file-document: **View Metadata**
    
    Inspect workflow details and call info

-   :material-download: **Get Outputs**
    
    List workflow output files

-   :material-text-long: **View Logs**
    
    Access workflow log paths for debugging

-   :material-cloud-download: **Read GCS Files**
    
    Fetch files from Google Cloud Storage

</div>

---

## :keyboard: Controls

| Key | Action |
|:---:|--------|
| ++ctrl+d++ | Send message |
| ++up++ / ++down++ | Scroll messages (input mode) |
| ++tab++ | Toggle navigation mode |
| ++y++ | Copy selected message |
| ++esc++ | Exit chat |

---

## :floppy_disk: Session Management

Conversations are persisted in SQLite for context retention.

```bash
# List existing sessions
pumbaa chat --list

# Resume a session
pumbaa chat --session <SESSION_ID>
```

---

## :gear: Configuration

### Environment Variables

=== ":material-google: Gemini"

    ```bash
    export PUMBAA_LLM_PROVIDER=gemini
    export GEMINI_API_KEY=<your-api-key>
    export GEMINI_MODEL=gemini-2.0-flash  # optional
    ```

=== ":material-server: Ollama"

    ```bash
    export PUMBAA_LLM_PROVIDER=ollama
    export OLLAMA_HOST=http://localhost:11434
    export OLLAMA_MODEL=llama3.2:3b
    ```

=== ":material-cloud: Vertex AI"

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

---

## :speech_balloon: Example Prompts

!!! example "Try these prompts"

    - "List my running workflows"
    - "What is the status of workflow abc-123?"
    - "Show me the outputs of workflow xyz-456"
    - "Read gs://bucket/path/to/file.txt"
    - "Query last 5 failed workflows"
