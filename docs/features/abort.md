# Abort Workflows

Stop running workflows.

<div class="grid cards" markdown>

-   :material-stop-circle: **Immediate Stop**

    Terminate running tasks immediately

-   :material-keyboard: **TUI Support**

    Abort from Dashboard with confirmation

</div>

!!! warning "Irreversible"
    Aborting a workflow cannot be undone.

## :material-rocket-launch: Quick Start

```bash
pumbaa workflow abort <workflow-id>
```

## :material-lightbulb: Example

```bash
pumbaa workflow abort abc12345-6789-0def-ghij-klmnopqrstuv
```

## :material-alert: Behavior

| Action | Description |
|--------|-------------|
| :material-stop: Stop tasks | Running tasks are terminated |
| :material-sync: Status change | `Running` → `Aborting` → `Aborted` |
| :material-folder: Intermediate files | **Not cleaned up** |

## :material-keyboard: From Dashboard

1. Navigate to workflow with ++up++ / ++down++
2. Press ++a++
3. Confirm action

## :material-book-open-variant: See Also

- [:material-view-dashboard: Dashboard](dashboard.md)
- [:material-magnify: Query](query.md)
