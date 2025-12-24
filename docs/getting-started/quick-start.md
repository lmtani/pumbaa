# Quick Start

Get up and running with Pumbaa in 5 minutes.

---

## :white_check_mark: Prerequisites

- Pumbaa installed ([Installation Guide](installation.md))
- Access to a Cromwell server
- (Optional) A WDL workflow file

---

## :one: Configure Cromwell Server

=== "Config Wizard (Recommended)"

    ```bash
    pumbaa config init
    ```

=== "Environment Variable"

    ```bash
    export CROMWELL_HOST=http://localhost:8000
    ```

=== "Persisted Config"

    ```bash
    pumbaa config set cromwell_host http://localhost:8000
    ```

---

## :two: Verify Connection

Test the connection to your Cromwell server:

```bash
pumbaa workflow query --limit 5
```

!!! success "Expected Result"
    You should see a list of recent workflows (or an empty list if none exist).

---

## :three: Explore the Dashboard

Launch the interactive dashboard:

```bash
pumbaa dashboard
```

### :keyboard: Dashboard Navigation

| Key | Action |
|:---:|--------|
| ++up++ / ++k++ | Move up |
| ++down++ / ++j++ | Move down |
| ++enter++ | Open debug view |
| ++s++ | Cycle status filter |
| ++slash++ | Filter by name |
| ++l++ | Filter by label |
| ++ctrl+x++ | Clear all filters |
| ++a++ | Abort workflow |
| ++shift+l++ | Manage labels |
| ++r++ | Refresh |
| ++q++ | Quit |

---

## :four: Submit a Workflow

```bash
pumbaa workflow submit \
  --workflow myworkflow.wdl \
  --inputs inputs.json
```

??? example "Example inputs.json"
    ```json
    {
      "myworkflow.input_file": "gs://bucket/data.txt",
      "myworkflow.sample_name": "sample-001"
    }
    ```

---

## :five: Monitor Workflow

=== "Query by Status"

    ```bash
    pumbaa workflow query --status Running
    ```

=== "View Metadata"

    ```bash
    pumbaa workflow metadata <workflow-id>
    ```

=== "Debug View"

    ```bash
    pumbaa workflow debug --id <workflow-id>
    ```

---

## :wrench: Common Tasks

<div class="grid cards" markdown>

-   :material-cancel: **Abort Workflow**
    
    ```bash
    pumbaa workflow abort <id>
    ```

-   :material-package: **Bundle WDL**
    
    ```bash
    pumbaa bundle --workflow main.wdl
    ```

</div>

---

## :bug: Troubleshooting

??? warning "Connection refused"
    Verify Cromwell is running:
    ```bash
    curl $CROMWELL_HOST/api/workflows/v1/query
    ```

??? warning "No workflows found"
    Submit a test workflow first.

??? warning "Dashboard not updating"
    Press ++r++ to manually refresh.

---

## :books: Next Steps

- [:material-view-dashboard: Dashboard Guide](../features/dashboard.md)
- [:material-bug: Debug View](../features/debug.md)
- [:material-upload: Submit Workflows](../features/submit.md)
- [:material-package: Bundle WDL](../features/bundle.md)

---

## :question: Getting Help

```bash
pumbaa --help
pumbaa <command> --help
```

[:material-github: GitHub Issues](https://github.com/lmtani/pumbaa/issues){ .md-button }
