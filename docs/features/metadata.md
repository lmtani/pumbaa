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
pumbaa workflow metadata <workflow-id> [FLAGS]
```

## :material-flag: Flags

| Flag | Alias | Description |
|------|:-----:|-------------|
| `--verbose` | `-v` | Show detailed call information |

## :material-lightbulb: Examples

=== "Basic"

    ```bash
    pumbaa workflow metadata abc12345
    ```

=== "Verbose"

    ```bash
    pumbaa workflow metadata abc12345 --verbose
    ```

## :material-file-document: Output

| Section | Description |
|---------|-------------|
| Workflow ID | Full UUID |
| Name | WDL workflow name |
| Status | Current state |
| Timing | Start/end timestamps |
| Inputs | Submitted inputs |
| Outputs | Generated outputs |
| Labels | User labels |
| Calls | Task execution details (with `-v`) |

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
