# Quick Start

Get up and running with Pumbaa in 5 minutes.

## Prerequisites

- Pumbaa installed ([Installation Guide](installation.md))
- Access to a Cromwell server
- (Optional) A WDL workflow file

## Step 1: Configure Cromwell Server

The easiest way is using the config wizard:

```bash
pumbaa config init
```

Or set your Cromwell server URL directly:

```bash
export CROMWELL_HOST=http://localhost:8000
```

!!! tip
    Use `pumbaa config set cromwell_host <url>` to persist configuration.

## Step 2: Verify Connection

Test the connection to your Cromwell server:

```bash
pumbaa workflow query --limit 5
```

You should see a list of recent workflows (or an empty list if none exist).

## Step 3: Explore the Dashboard

Launch the interactive dashboard:

```bash
pumbaa dashboard
```

### Dashboard Navigation

- **↑/↓**: Navigate through workflows
- **Enter**: Open workflow in debug view
- **s**: Filter by status (All/Running/Failed/Succeeded)
- **/**: Filter by workflow name
- **a**: Abort selected workflow
- **r**: Refresh list
- **q**: Quit

![Dashboard](../assets/dashboard.png)

## Step 4: Submit a Workflow

If you have a WDL file, submit it:

```bash
pumbaa workflow submit \
  --workflow myworkflow.wdl \
  --inputs inputs.json
```

Example `inputs.json`:

```json
{
  "myworkflow.input_file": "gs://bucket/data.txt",
  "myworkflow.sample_name": "sample-001"
}
```

## Step 5: Monitor Workflow

### Query Specific Workflow

```bash
pumbaa workflow query --status Running
```

### View Metadata

```bash
pumbaa workflow metadata <workflow-id>
```

### Debug View

Open detailed execution view:

```bash
pumbaa workflow debug --id <workflow-id>
```

Navigate the debug view:

- **↑/↓** or **j/k**: Navigate through tasks
- **←/→** or **h/l**: Collapse/expand nodes
- **Enter**: Toggle expand
- **Tab**: Switch between tree and details panel
- **c**: View command
- **i**: View inputs
- **o**: View outputs
- **L**: View log paths
- **t**: View task duration timeline
- **q**: Return to dashboard

## Common Workflows

### Abort a Workflow

```bash
pumbaa workflow abort <workflow-id>
```

Or use the dashboard (press `a` on selected workflow).

### Bundle a WDL

Package your WDL with all imports:

```bash
pumbaa bundle --workflow main.wdl --output my-bundle
```

This creates:
- `my-bundle.wdl` - Main workflow file
- `my-bundle.zip` - All dependencies bundled


## Troubleshooting

### "Connection refused"

Verify Cromwell is running:

```bash
curl $CROMWELL_HOST/api/workflows/v1/query
```

### "No workflows found"

Submit a test workflow:

```bash
# Use the example workflow
pumbaa workflow submit --workflow <simple-workflow>.wdl
```

### Dashboard not updating

Press `r` to manually refresh, or check your connection.

## Next Steps

Now that you're familiar with the basics:

- [Dashboard Guide](../features/dashboard.md) - Master the interactive dashboard
- [Debug View](../features/debug.md) - Advanced debugging techniques
- [Submit Workflows](../features/submit.md) - Detailed submission options
- [Bundle WDL](../features/bundle.md) - Package workflows for distribution

## Getting Help

- Run `pumbaa --help` for command reference
- Run `pumbaa <command> --help` for specific command help
- Visit [GitHub Issues](https://github.com/lmtani/pumbaa/issues) for support
