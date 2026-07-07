# Inputs & Outputs

Retrieve the submitted inputs or generated outputs of a workflow.

## :material-rocket-launch: Quick Start

=== "Inputs"

    ```bash
    pumbaa workflow inputs <workflow-id>
    ```

    Aliases: `pumbaa wf i`, `pumbaa wf in`

=== "Outputs"

    ```bash
    pumbaa workflow outputs <workflow-id>
    ```

    Aliases: `pumbaa wf o`, `pumbaa wf out`

## :material-flag: Flags

| Flag | Alias | Description |
|------|:-----:|-------------|
| `--json` | `-j` | Output as JSON instead of a table |

## :material-lightbulb: Examples

=== "Inspect inputs"

    ```bash
    pumbaa workflow inputs abc-123
    ```

    Human-readable key-value listing (the workflow name prefix is stripped for readability).

=== "Pipe outputs to a script"

    ```bash
    pumbaa workflow outputs abc-123 --json | jq -r '.["pipeline.final_vcf"]'
    ```

=== "Reuse inputs for a new submission"

    ```bash
    pumbaa workflow inputs abc-123 --json > inputs.json
    # edit inputs.json, then:
    pumbaa workflow submit --workflow pipeline.wdl --inputs inputs.json
    ```

## :material-book-open-variant: See Also

- [:material-upload: Submit](submit.md) — Resubmit with modified inputs
- [:material-file-document: Metadata](metadata.md) — Status, timing, and calls
- [:material-file-compare: Diff](diff.md) — Compare inputs between two runs
