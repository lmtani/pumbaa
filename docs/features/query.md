# Query Workflows

List and filter workflows via CLI.

## Usage

```bash
pumbaa workflow query [FLAGS]
```

## Flags

| Flag | Alias | Description |
|------|-------|-------------|
| `--name` | `-n` | Filter by workflow name |
| `--status` | `-s` | Filter by status (repeatable) |
| `--limit` | `-l` | Max results (default: 20) |

## Examples

### List All

```bash
pumbaa workflow query
```

### Filter by Status

```bash
pumbaa workflow query --status Running
pumbaa workflow query --status Failed --status Aborting
```

### Filter by Name

```bash
pumbaa workflow query --name variant-calling
```

### Combined

```bash
pumbaa workflow query \
  --name pipeline \
  --status Succeeded \
  --limit 10
```

## Status Values

- `Submitted`
- `Running`
- `Succeeded`
- `Failed`
- `Aborting`
- `Aborted`

## Output

Displays table with:
- ID (first 8 chars)
- Name
- Status (color-coded)
- Submission time

## See Also

- [Dashboard](dashboard.md) - Interactive query
- [Metadata](metadata.md) - Detailed workflow info
