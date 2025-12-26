# Dashboard

Interactive TUI for monitoring Cromwell workflows.

<div class="grid cards" markdown>

-   :material-filter: **Smart Filtering**

    Filter by status, name, or labels

-   :material-keyboard: **Keyboard-first**

    Navigate efficiently without touching the mouse

</div>

## :material-rocket-launch: Quick Start

```bash
pumbaa dashboard
```

## :material-keyboard: Controls

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


## :material-star: Features

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


## :material-table: Workflow Columns

| Column | Description |
|--------|-------------|
| **ID** | First 8 chars of workflow UUID |
| **Name** | From WDL workflow definition |
| **Status** | Color-coded (Running/Succeeded/Failed) |
| **Submitted** | Submission timestamp |
| **Labels** | User-submitted labels |

## :material-book-open-variant: See Also

- [:material-bug: Debug View](debug.md)
- [:material-magnify: Query](query.md)
