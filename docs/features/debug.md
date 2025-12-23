# Debug View

Inspect workflow execution tree and call-level details.

!!! note "Requirement"
    Requires a working connection to your Cromwell server (set via `--host` or `CROMWELL_HOST`).

---

## :rocket: Usage

=== "From Dashboard"

    Press ++enter++ on any workflow

=== "CLI"

    ```bash
    pumbaa workflow debug <workflow-id>
    ```

---

## :keyboard: Navigation

| Key | Action |
|:---:|--------|
| ++up++ / ++down++ | Navigate tree |
| ++right++ / ++left++ | Expand / Collapse |
| ++enter++ | Toggle expand |
| ++d++ | Show call details |
| ++o++ | Show logs |
| ++q++ | Back |

---

## :zap: Quick Actions (1–5)

The details panel exposes quick actions for the selected call:

| Key | Action | Description |
|:---:|--------|-------------|
| ++1++ | **Inputs** | Open modal with call inputs |
| ++2++ | **Outputs** | Open modal with call outputs |
| ++3++ | **Command** | View executed command |
| ++4++ | **Logs** | Load logs (from GCS or local) |
| ++5++ | **Efficiency** | Resource usage metrics |

!!! tip "Copy to Clipboard"
    In modals, press ++y++ to copy content to clipboard.

---

## :stopwatch: Timing

Press ++t++ to open the timeline for the selected workflow or subworkflow.

- Navigate to a subworkflow in the tree and press ++t++ to see its timeline
- Timing updates as you expand subworkflows

---

## :chart_with_upwards_trend: Efficiency

The Efficiency view (action ++5++) displays resource usage metrics:

!!! warning "Requirement"
    This feature requires the **monitoring script**. See [Resource Monitoring](resource-monitoring.md).

**Displayed metrics:**

| Metric | Description |
|--------|-------------|
| Peak CPU | Usage vs. allocated cores |
| Peak Memory | Usage vs. allocated memory |
| Efficiency % | Resource utilization percentage |

!!! tip "Cost Optimization"
    Identify over-provisioned tasks where resources can be reduced.

---

## :books: See Also

- [:material-view-dashboard: Dashboard](dashboard.md)
- [:material-file-document: Metadata](metadata.md)
- [:material-monitor: Resource Monitoring](resource-monitoring.md)
