# Dashboard

Interactive TUI for monitoring Cromwell workflows.

---

## :rocket: Usage

```bash
pumbaa dashboard
```

---

## :keyboard: Controls

| Key | Action |
|:---:|--------|
| ++up++ / ++down++ | Navigate |
| ++enter++ | Open debug view |
| ++a++ | Abort workflow |
| ++s++ | Filter by status |
| ++slash++ | Filter by name |
| ++l++ | Filter by label |
| ++r++ | Refresh |
| ++ctrl+x++ | Clear filters |
| ++q++ | Quit |

---

## :sparkles: Features

<div class="grid cards" markdown>

-   :material-filter: **Status Filter**
    
    Filter by All / Running / Failed / Succeeded

-   :material-text-search: **Name Filter**
    
    Press ++slash++ to search by workflow name

-   :material-label: **Label Filter**
    
    Press ++l++ to filter by workflow labels

-   :material-cancel: **Abort**
    
    Press ++a++ to abort with confirmation

</div>

---

## :page_facing_up: Workflow Columns

| Column | Description |
|--------|-------------|
| **ID** | First 8 chars of workflow UUID |
| **Name** | From WDL workflow definition |
| **Status** | Color-coded (Running/Succeeded/Failed) |
| **Submitted** | Submission timestamp |
| **Labels** | User-submitted labels |

---

## :books: See Also

- [:material-bug: Debug View](debug.md)
- [:material-magnify: Query](query.md)
