# Chat Agent

Interactive AI assistant for querying Cromwell workflows and reading files via natural language.

<div class="grid cards" markdown>

-   :material-chat-processing: **Natural Language**

    Ask questions about workflows in plain English

-   :material-tools: **Built-in Tools**

    Query workflows, read GCS files, check status

-   :material-database: **Session Persistence**

    Resume conversations with context retention

</div>

## :material-rocket-launch: Quick Start

=== "Using Config Wizard"

    ```bash
    pumbaa config init
    ```
    
    The wizard will guide you through setting up your preferred LLM provider.

=== "Direct Start"

    ```bash
    pumbaa chat
    ```

## :material-robot: LLM Providers

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

## :material-star: Capabilities

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

-   :material-file-edit: **Write Files**
    
    Save scripts or notes to the working directory

-   :material-book-search: **WDL Knowledge Base**
    
    List, search, and inspect your indexed WDL workflows (`PUMBAA_WDL_DIR`)

-   :material-airplane-check: **Prepare a Submission**
    
    Scaffold an inputs template from a WDL and preflight it before running

-   :material-alert-circle: **Diagnose Failures**
    
    Summarize root causes and read the failing task's log

</div>

!!! info "Streaming"
    Responses stream in real time as the model generates them. Press ++esc++ during generation to cancel it.

## :material-keyboard: Controls

| Key | Action |
|:---:|--------|
| ++enter++ | Send message |
| ++ctrl+j++ | Insert line break |
| ++up++ / ++down++ | Scroll messages (input mode) |
| ++tab++ | Toggle navigation mode |
| ++y++ | Copy selected message (navigation mode) |
| ++g++ / ++shift+g++ | First / last message (navigation mode) |
| ++ctrl+s++ | Browse and switch sessions |
| ++ctrl+r++ | Resume previous session for the same task |
| ++esc++ | Cancel generation · leave input · exit chat |

## :material-floppy: Session Management

Conversations are persisted in SQLite (`~/.pumbaa/sessions.db`) for context retention.

```bash
# List existing sessions
pumbaa chat --list

# Resume a session
pumbaa chat --session <SESSION_ID>
```

Inside the TUI:

- ++ctrl+s++ opens a session browser to switch conversations
- ++ctrl+r++ resumes the most recent session for the current task
- Opening the chat from the debug view for the same task automatically resumes that task's conversation

## :material-cog: Configuration

### Environment Variables

=== ":material-google: Gemini"

    ```bash
    export PUMBAA_LLM_PROVIDER=gemini
    export GEMINI_API_KEY=<your-api-key>
    export GEMINI_MODEL=gemini-2.5-flash  # optional
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
    export VERTEX_MODEL=gemini-2.5-flash
    ```

### Session Storage

```bash
export PUMBAA_SESSION_DB=~/.pumbaa/sessions.db
```

## :material-message-text: Example Prompts

!!! example "Try these prompts"

    - "List my running workflows"
    - "What is the status of workflow abc-123?"
    - "Show me the outputs of workflow xyz-456"
    - "Read gs://bucket/path/to/file.txt"
    - "Query last 5 failed workflows"
    - "What inputs does main.wdl need?"
    - "Check my inputs.json against main.wdl before I submit"
    - "Why did workflow abc-123 fail?"
