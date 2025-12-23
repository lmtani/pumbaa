# Query Workflows

List and filter workflows via CLI.

---

## :rocket: Usage

```bash
pumbaa workflow query [FLAGS]
```

---

## :flags: Flags

| Flag | Alias | Description |
|------|:-----:|-------------|
| `--name` | `-n` | Filter by workflow name |
| `--status` | `-s` | Filter by status (repeatable) |
| `--limit` | `-l` | Max results (default: 20) |

---

## :bulb: Examples

=== "List All"

    ```bash
    pumbaa workflow query
    ```

=== "By Status"

    ```bash
    pumbaa workflow query --status Running
    pumbaa workflow query --status Failed --status Aborting
    ```

=== "By Name"

    ```bash
    pumbaa workflow query --name variant-calling
    ```

=== "Combined"

    ```bash
    pumbaa workflow query \
      --name pipeline \
      --status Succeeded \
      --limit 10
    ```

---

## :traffic_light: Status Values

| Status | Description |
|:------:|-------------|
| `Submitted` | Pending execution |
| `Running` | Currently executing |
| `Succeeded` | Completed successfully |
| `Failed` | Execution failed |
| `Aborting` | Being aborted |
| `Aborted` | Aborted by user |

---

## :page_facing_up: Output

Displays table with:

- **ID** — First 8 chars
- **Name** — Workflow name
- **Status** — Color-coded
- **Submitted** — Timestamp

---

## :books: See Also

- [:material-view-dashboard: Dashboard](dashboard.md) — Interactive query
- [:material-file-document: Metadata](metadata.md) — Detailed workflow info
