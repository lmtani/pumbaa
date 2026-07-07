# Debug View

Inspect workflow execution tree and call-level details.

<div class="grid cards" markdown>

-   :material-file-tree: **Execution Tree**

    Navigate workflow structure with expandable nodes

-   :material-clipboard-text: **Call Details**

    View inputs, outputs, commands, and logs

-   :material-chart-timeline: **Timeline Analysis**

    Visualize task durations and identify bottlenecks

-   :material-chart-box: **Resource Efficiency**

    Analyze CPU, memory, and disk utilization

</div>

!!! note "Requirement"
    Requires a working connection to your Cromwell server (set via `--host` or `CROMWELL_HOST`).

## :material-rocket-launch: Quick Start

=== "From Dashboard"

    Press ++enter++ on any workflow

=== "CLI"

    ```bash
    pumbaa workflow debug --id <workflow-id>
    ```

    Other options: `--file metadata.json` to debug from a saved metadata file (offline), and `--expand-subworkflows` to fetch subworkflow metadata upfront.

## :material-keyboard: Navigation

| Key | Action |
|:---:|--------|
| ++up++ / ++down++ | Navigate tree |
| ++right++ / ++left++ | Expand / Collapse |
| ++enter++ | Toggle expand |
| ++tab++ | Switch panel focus |
| ++shift+e++ | Expand all nodes |
| ++shift+c++ | Collapse all nodes |
| ++slash++ | Search tree · ++n++ / ++shift+n++ next/prev match |
| ++f++ | Expand failed/preempted paths |
| ++"]"++ / ++"["++ | Jump to next / previous failure |
| ++shift+f++ | Failure summary (grouped by error) |
| ++e++ | Error details for selected node |
| ++d++ | Node details panel |
| ++"$"++ | Cost breakdown by task |
| ++w++ | Watch mode (auto-refresh) |
| ++y++ | Copy menu (context-sensitive) |
| ++"<"++ / ++">"++ | Resize tree/details split |
| ++a++ | Chat with AI |
| ++esc++ | Close modal / back · ++ctrl+c++ quits |

## :material-lightning-bolt: Quick Actions

Quick actions are context-sensitive and depend on the selected node type. Press number keys ++1++ to ++6++ to trigger actions.

### Workflow / SubWorkflow Nodes

| Key | Action | Description |
|:---:|--------|-------------|
| ++1++ | **Inputs** | Open modal with workflow inputs |
| ++2++ | **Outputs** | Open modal with workflow outputs |
| ++3++ | **Options** | View submitted workflow options |
| ++4++ | **Timeline** | Open timeline modal (tasks sorted by duration) |
| ++5++ | **Workflow Log** | Load and display workflow log |

### Task / Shard Nodes

| Key | Action | Description |
|:---:|--------|-------------|
| ++1++ | **Inputs** | Open modal with task inputs |
| ++2++ | **Outputs** | Open modal with task outputs |
| ++3++ | **Command** | View executed command |
| ++4++ | **Logs** | Switch to logs view (stdout/stderr/monitoring) |
| ++5++ | **Efficiency** | Analyze resource usage (requires monitoring script) |
| ++6++ | **AI Chat** | Open chat with selected context data |

!!! tip "Copy to Clipboard"
    In modals, press ++y++ to copy content to clipboard.

## :material-robot: AI Chat Integration

Press ++6++ or ++a++ on a **Task** or **Shard** node to open the AI chat with task context.

### Context Selection

Before starting the chat, you'll be prompted to select which data to include in the AI context:

| Option | Description |
|--------|-------------|
| **Metadata** | Task info, status, timing, command |
| **Stderr** | Standard error output (last 200 lines) |
| **Stdout** | Standard output (last 200 lines) |
| **Monitoring** | Resource efficiency analysis |
| **Batch Logs** | Google Batch logs (requires Google Batch backend) |

!!! tip "Debugging Failed Tasks"
    When debugging a failed task, select **Metadata** + **Stderr** for the most relevant context. The AI can help interpret error messages and suggest fixes.

### Selection Controls

| Key | Action |
|:---:|--------|
| ++up++ / ++down++ | Navigate options |
| ++space++ | Toggle selection |
| ++enter++ | Confirm and start chat |
| ++esc++ | Cancel |

## :material-timer-outline: Timeline Analysis

Press ++4++ on a **Workflow** or **SubWorkflow** node to open the timeline modal.

The timeline shows:

- **Tasks sorted by duration** (longest first)
- **Visual timeline bars** showing when each task ran relative to workflow start
- **Time range** for each task (start → end)
- **Status icons** (✓ Done, ● Running, ✗ Failed, ↺ Preempted)

!!! tip "Subworkflow Timelines"
    Navigate to a subworkflow node and press ++4++ to see its specific timeline, separate from the parent workflow.

## :material-currency-usd: Cost Breakdown

Press ++"$"++ to open the per-task cost breakdown modal.

- Tasks are listed with their **estimated cost**, sorted by the most expensive first
- Costs are billed on **VM lifetime** (provisioning to teardown), not just task wall-clock time
- **Subworkflow costs are included by default** — the modal fetches expanded metadata as needed

!!! tip "Where is my money going?"
    Combine the cost modal with the timeline (++4++) and resource efficiency (++5++) views: the most expensive tasks are usually the best candidates for right-sizing.

## :material-alert-circle: Investigating Failures

For workflows with failed tasks:

1. Press ++f++ to expand only the failed/preempted paths of the tree
2. Jump between failures with ++"]"++ / ++"["++
3. Press ++shift+f++ for a **failure summary** modal that groups failures by error message
4. Press ++e++ on a node to see its full error details

For workflows still running, press ++w++ to enable **watch mode** — the tree refreshes automatically while preserving your expand/collapse state.

## :material-chart-areaspline: Resource Efficiency

Press ++5++ on a **Task** or **Shard** node to analyze resource utilization.

!!! warning "Requirement"
    This feature requires the **monitoring script** to be configured in Cromwell. See [Resource Monitoring](resource-monitoring.md).

**Displayed metrics:**

| Metric | Description |
|--------|-------------|
| **CPU** | Peak and average usage vs. allocated cores |
| **Memory** | Peak usage vs. allocated memory |
| **Disk** | Peak usage vs. allocated disk space |
| **Efficiency %** | Visual gauge showing utilization percentage |

!!! tip "Cost Optimization"
    Low efficiency indicates over-provisioned resources. Consider reducing CPU/memory/disk allocations for tasks with < 50% efficiency.

## :material-sync: Preemption Summary

For **Workflow** and **SubWorkflow** nodes running with preemptible instances, the details panel shows:

- **Cost Efficiency** — Overall efficiency considering preemption overhead
- **Preemptible/Total Tasks** — How many tasks used preemptible instances
- **Total Attempts** — Including retries after preemptions
- **Total Preemptions** — Number of times tasks were preempted

### Problematic Tasks

Tasks with high preemption impact are highlighted:

- Tasks with **< 70% cost efficiency**
- Tasks where preemption caused **> 10% cost overhead**

!!! info "Subworkflows"
    Preemption stats are calculated per-level. Navigate into subworkflows to see their individual preemption analysis.

## :material-chart-bar: Scatter/Shard Summary

For **Call** nodes with multiple shards (scatter operations), the panel shows:

- **Total Shards** count
- **Status Breakdown** — Done, Running, Failed, Preempted counts
- **Timing Statistics** — Wall clock, min/max/avg per-shard duration

Expand the node to navigate individual shards.

## :material-book-open-variant: See Also

- [:material-view-dashboard: Dashboard](dashboard.md)
- [:material-file-document: Metadata](metadata.md)
- [:material-monitor: Resource Monitoring](resource-monitoring.md)
