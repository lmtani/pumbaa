# Dashboard

Interactive TUI for monitoring Cromwell workflows.

## Usage

```bash
pumbaa dashboard
```

## Interface Layout



## Keys

| Key | Action |
|-----|--------|
| `↑/↓` or `k/j` | Navigate |
| `Enter` | Open debug view |
| `a` | Abort workflow |
| `s` | Filter by status |
| `/` | Filter by name |
| `l` | Filter by label |
| `r` | Refresh |
| `Ctrl+X` | Clear filters |
| `q` | Quit |

## Features

- Filter by status (All/Running/Failed/Succeeded)
- Filter by name or labels
- Press `Enter` to debug selected workflow
- Press `a` to abort (with confirmation)

## Workflow Columns

- **ID** - First 8 chars of workflow ID
- **Name** - From WDL workflow definition
- **Status** - Color-coded (Running/Succeeded/Failed/etc)
- **Submitted** - Submission timestamp
- **Labels** - Submitted labels

## See Also

- [Debug View](debug.md)
- [Query](query.md)
