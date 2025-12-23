# Abort Workflows

Stop running workflows.

---

## :rocket: Usage

```bash
pumbaa workflow abort <workflow-id>
```

---

## :bulb: Example

```bash
pumbaa workflow abort abc12345-6789-0def-ghij-klmnopqrstuv
```

---

## :warning: Behavior

!!! warning "Irreversible"
    Aborting a workflow cannot be undone.

| Action | Description |
|--------|-------------|
| :material-stop: Stop tasks | Running tasks are terminated |
| :material-sync: Status change | `Running` → `Aborting` → `Aborted` |
| :material-folder: Intermediate files | **Not cleaned up** |

---

## :keyboard: From Dashboard

1. Navigate to workflow with ++up++ / ++down++
2. Press ++a++
3. Confirm action

---

## :books: See Also

- [:material-view-dashboard: Dashboard](dashboard.md)
- [:material-magnify: Query](query.md)
