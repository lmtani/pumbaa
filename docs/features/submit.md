# Submit Workflows

Submit WDL workflows to Cromwell.

## Usage

```bash
pumbaa workflow submit --workflow FILE [OPTIONS]
```

## Flags

| Flag | Alias | Required | Description |
|------|-------|----------|-------------|
| `--workflow` | `-w` | Yes | WDL workflow file |
| `--inputs` | `-i` | No | Inputs JSON file |
| `--options` | `-o` | No | Options JSON file |
| `--dependencies` | `-d` | No | Dependencies ZIP file |
| `--label` | `-l` | No | Labels (format: `key=value`) |

## Examples

### Basic

```bash
pumbaa workflow submit --workflow hello.wdl
```

### With Inputs

```bash
pumbaa workflow submit \
  --workflow pipeline.wdl \
  --inputs inputs.json
```

### Complete

```bash
pumbaa workflow submit \
  --workflow analysis.wdl \
  --inputs inputs.json \
  --options options.json \
  --dependencies deps.zip \
  --label sample=S001 --label env=prod
```

## Input File

JSON format matching WDL inputs:

```json
{
  "workflow.sample_name": "SAMPLE-001",
  "workflow.fastq": "gs://bucket/sample.fastq.gz",
  "workflow.reference": "gs://bucket/ref.fa"
}
```

## Options File

Configure workflow execution:

```json
{
  "final_workflow_outputs_dir": "gs://bucket/outputs",
  "delete_intermediate_output_files": true,
  "write_to_cache": true,
  "read_from_cache": true
}
```

Common options:
- `final_workflow_outputs_dir` - Output location
- `delete_intermediate_output_files` - Cleanup
- `write_to_cache`/`read_from_cache` - Call caching

## Dependencies

For workflows with imports, create ZIP with imported files:

```bash
# Using pumbaa
pumbaa bundle create --workflow workflow.wdl --output deps.zip
```

Submit with:
```bash
pumbaa workflow submit --workflow workflow.wdl --dependencies deps.zip
```

## Labels

Organize workflows with labels:

```bash
pumbaa workflow submit \
  --workflow pipeline.wdl \
  --label sample=S001 \
  --label pipeline=variant-calling \
  --label user=$USER
```

Filter by labels in dashboard (press `l`).

## Response

Success:
```json
{
  "id": "abc12345-6789-0def-ghij-klmnopqrstuv",
  "status": "Submitted"
}
```

Use the ID to monitor:
```bash
pumbaa dashboard                        # Interactive
pumbaa workflow debug <id>             # Debug view
pumbaa workflow query --id <id>        # CLI query
```

## See Also

- [Bundle Creation](bundle.md)
- [Dashboard](dashboard.md)
- [Debug View](debug.md)
