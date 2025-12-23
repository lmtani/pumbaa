# Metadata

Get detailed workflow metadata.

---

## :rocket: Usage

```bash
pumbaa workflow metadata <workflow-id> [FLAGS]
```

---

## :flags: Flags

| Flag | Alias | Description |
|------|:-----:|-------------|
| `--verbose` | `-v` | Show detailed call information |

---

## :bulb: Examples

=== "Basic"

    ```bash
    pumbaa workflow metadata abc12345
    ```

=== "Verbose"

    ```bash
    pumbaa workflow metadata abc12345 --verbose
    ```

---

## :page_facing_up: Output

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

---

## :wrench: Use Cases

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

---

## :books: See Also

- [:material-bug: Debug View](debug.md) — Interactive exploration
- [:material-magnify: Query](query.md) — List workflows
