# Submit Workflows

Submit WDL workflows to Cromwell.

<div class="grid cards" markdown>

-   :material-upload: **One Command**

    Submit workflows with a single CLI command

-   :material-label: **Labels Support**

    Organize workflows with custom labels

-   :material-package: **Dependencies**

    Include imported WDL files via ZIP bundles

</div>

## :material-rocket-launch: Quick Start

```bash
pumbaa workflow submit --workflow FILE [OPTIONS]
```

## :material-flag: Flags

| Flag | Alias | Required | Description |
|------|:-----:|:--------:|-------------|
| `--workflow` | `-w` | :material-check: | WDL workflow file |
| `--inputs` | `-i` | | Inputs JSON file |
| `--options` | `-o` | | Options JSON file |
| `--dependencies` | `-d` | | Dependencies ZIP file |
| `--label` | `-l` | | Labels (`key=value`) |

## :material-lightbulb: Examples

=== "Basic"

    ```bash
    pumbaa workflow submit --workflow hello.wdl
    ```

=== "With Inputs"

    ```bash
    pumbaa workflow submit \
      --workflow pipeline.wdl \
      --inputs inputs.json
    ```

=== "Complete"

    ```bash
    pumbaa workflow submit \
      --workflow analysis.wdl \
      --inputs inputs.json \
      --options options.json \
      --dependencies deps.zip \
      --label sample=S001 \
      --label env=prod
    ```

## :material-file-document: Input File

JSON format matching WDL inputs:

```json
{
  "workflow.sample_name": "SAMPLE-001",
  "workflow.fastq": "gs://bucket/sample.fastq.gz",
  "workflow.reference": "gs://bucket/ref.fa"
}
```

## :material-cog: Options File

Configure workflow execution:

```json
{
  "final_workflow_outputs_dir": "gs://bucket/outputs",
  "delete_intermediate_output_files": true,
  "write_to_cache": true,
  "read_from_cache": true
}
```

| Option | Description |
|--------|-------------|
| `final_workflow_outputs_dir` | Output location |
| `delete_intermediate_output_files` | Cleanup intermediates |
| `write_to_cache` / `read_from_cache` | Call caching |

## :material-package-variant: Dependencies

For workflows with imports, create ZIP with imported files:

```bash
pumbaa bundle create --workflow workflow.wdl --output deps.zip
```

Then submit:

```bash
pumbaa workflow submit --workflow workflow.wdl --dependencies deps.zip
```

## :material-label: Labels

Organize workflows with labels:

```bash
pumbaa workflow submit \
  --workflow pipeline.wdl \
  --label sample=S001 \
  --label pipeline=variant-calling \
  --label user=$USER
```

!!! tip "Filter in Dashboard"
    Press ++l++ in dashboard to filter by labels.

## :material-check-circle: Response

```json
{
  "id": "abc12345-6789-0def-ghij-klmnopqrstuv",
  "status": "Submitted"
}
```

Use the ID to monitor:

```bash
pumbaa dashboard                    # Interactive
pumbaa workflow debug <id>          # Debug view  
pumbaa workflow query --id <id>     # CLI query
```

## :material-book-open-variant: See Also

- [:material-package: Bundle Creation](bundle.md)
- [:material-view-dashboard: Dashboard](dashboard.md)
- [:material-bug: Debug View](debug.md)
