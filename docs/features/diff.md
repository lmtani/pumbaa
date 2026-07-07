# Diff Two Runs

Compare two workflow runs: inputs, options, WDL source, and per-task changes.

<div class="grid cards" markdown>

-   :material-file-compare: **Full Comparison**

    Inputs, options, source, and task-level differences

-   :material-cached: **Cache-aware**

    Recovers real metrics of call-cached tasks from their source runs

-   :material-code-json: **JSON Output**

    Machine-readable diff for scripting

</div>

## :material-rocket-launch: Quick Start

```bash
pumbaa workflow diff <workflow-id-a> <workflow-id-b>
```

Alias: `pumbaa wf compare`

## :material-flag: Flags

| Flag | Description |
|------|-------------|
| `--json` | Output the diff as JSON |
| `--no-cache-resolve` | Do not fetch cache sources to recover real metrics of cache-hit tasks |

## :material-file-document: Output

The diff is organized in four sections:

| Section | What it shows |
|---------|---------------|
| **Inputs** | Added (`+`), removed (`-`), and changed (`~`) input keys |
| **Options** | Same, for workflow options |
| **Source** | Whether the submitted WDL source changed (with line counts) |
| **Tasks** | Tasks only in one run, plus status, docker image, shard count, attempts, and duration changes |

Duration changes come with a verdict (e.g. `2.3× slower`), making regressions easy to spot.

!!! info "Call caching"
    When a task was a **call-cache hit**, its wall-clock time says nothing about real performance. By default, Pumbaa follows the cache back to the source run and compares the **original** metrics. Use `--no-cache-resolve` to skip this (faster, fewer API calls).

## :material-lightbulb: Examples

=== "Human-readable"

    ```bash
    pumbaa workflow diff abc-123 def-456
    ```

=== "JSON for scripting"

    ```bash
    pumbaa workflow diff abc-123 def-456 --json | jq '.Tasks'
    ```

!!! tip "From the dashboard"
    The same comparison is available interactively: press ++c++ on two workflows in the [Dashboard](dashboard.md).

## :material-wrench: Use Cases

<div class="grid cards" markdown>

-   :material-speedometer: **Performance Regressions**

    Why did this run take twice as long?

-   :material-bug: **Failure Triage**

    What changed between the run that worked and the one that failed?

-   :material-docker: **Image Updates**

    Confirm which tasks picked up a new docker image

</div>

## :material-book-open-variant: See Also

- [:material-view-dashboard: Dashboard](dashboard.md) — Interactive diff with ++c++
- [:material-file-document: Metadata](metadata.md) — Single-run details
