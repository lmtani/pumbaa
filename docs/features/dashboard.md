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
| ++a++ | Abort workflow (Running/Submitted only) |
| ++s++ | Filter by status |
| ++slash++ | Filter by name |
| ++l++ | Filter by label |
| ++u++ | Go to workflow by UUID |
| ++ctrl+x++ | Clear filters |
| ++shift+l++ | Manage labels |
| ++c++ | Compare runs (mark base, then target) |
| ++e++ | Error details |
| ++r++ | Refresh |
| ++w++ | Toggle auto-refresh |
| ++question++ | Help overlay |
| ++esc++ | Back / quit · ++ctrl+c++ quits immediately |


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

-   :material-file-compare: **Compare Runs**
    
    Press ++c++ on two workflows to diff them

-   :material-refresh-auto: **Auto-refresh**
    
    Press ++w++ to keep the list updating

</div>

## :material-file-compare: Comparing Two Runs

To understand why two executions of the same pipeline behaved differently:

1. Press ++c++ on the first workflow to mark it as the **base**
2. Press ++c++ on a second workflow to open the **diff modal**

The diff shows differences in **inputs**, **options**, and **per-task metrics** (duration, resources), resolving call-cache hits to their original executions so cached tasks show real numbers.

!!! tip "CLI equivalent"
    The same comparison is available non-interactively: `pumbaa workflow diff <id-a> <id-b>` (see [Diff](diff.md)).


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
