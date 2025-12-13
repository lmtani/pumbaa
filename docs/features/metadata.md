# Metadata

Get detailed workflow metadata.

## Usage

```bash
pumbaa workflow metadata <workflow-id> [FLAGS]
```

## Flags

| Flag | Alias | Description |
|------|-------|-------------|
| `--verbose` | `-v` | Show detailed call information |

## Examples

### Basic

```bash
pumbaa workflow metadata abc12345
```

### Verbose

```bash
pumbaa workflow metadata abc12345 --verbose
```

## Output

Shows:
- Workflow ID and name
- Status and timing
- Inputs and outputs
- Labels
- Call details (with `--verbose`)

## Use Cases

- Debugging failed workflows
- Extracting outputs
- Checking execution times
- Verifying inputs

## See Also

- [Debug View](debug.md) - Interactive metadata exploration
- [Query](query.md) - List workflows
