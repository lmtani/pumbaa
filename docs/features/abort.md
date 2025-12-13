# Abort Workflows

Stop running workflows.

## Usage

```bash
pumbaa workflow abort <workflow-id>
```

## Examples

```bash
pumbaa workflow abort abc12345-6789-0def-ghij-klmnopqrstuv
```

## Behavior

- Stops running tasks
- Sets status to `Aborting` then `Aborted`
- Doesn't clean up intermediate files
- Irreversible operation

## From Dashboard

1. Navigate to workflow
2. Press `a`
3. Confirm action

## See Also

- [Dashboard](dashboard.md)
- [Query](query.md)
