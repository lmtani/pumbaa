# Metadata

Get detailed workflow metadata.

<div class="grid cards" markdown>

-   :material-file-search: **Complete Info**

    Inputs, outputs, timing, labels, and more

-   :material-bug: **Debug Failures**

    Inspect failed task details

-   :material-timer: **Execution Timing**

    Review start/end timestamps

</div>

## :material-rocket-launch: Quick Start

```bash
pumbaa workflow metadata <workflow-id>
```

Aliases: `pumbaa wf m`, `pumbaa wf meta`

## :material-lightbulb: Example

```bash
pumbaa workflow metadata abc12345-6789-0def-ghij-klmnopqrstuv
```

## :material-file-document: Output

| Section | Description |
|---------|-------------|
| Workflow ID | Full UUID |
| Name | WDL workflow name |
| Status | Current state |
| Timing | Start/end timestamps and duration |
| Labels | User labels |
| Calls | Per-task status, timing, and attempts |
| Failures | Error messages for failed workflows |

!!! tip "Inputs and outputs"
    Submitted inputs and generated outputs have their own commands: see [Inputs & Outputs](inputs-outputs.md).

## :material-wrench: Use Cases

<div class="grid cards" markdown>

-   :material-bug: **Debug Failures**
    
    Inspect failed task details

-   :material-download: **Extract Outputs**
    
    Find output file paths

-   :material-timer: **Check Timing**
    
    Review execution duration

-   :material-check: **Verify Inputs**
    
    Confirm submitted parameters

</div>

## :material-book-open-variant: See Also

- [:material-bug: Debug View](debug.md) — Interactive exploration
- [:material-magnify: Query](query.md) — List workflows
