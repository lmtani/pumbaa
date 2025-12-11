# Pumbaa 🐗

A CLI tool for interacting with [Cromwell](https://cromwell.readthedocs.io/) workflow engine and WDL files.

## Installation

### Quick Install (Linux/macOS)

```bash
curl -sSL https://raw.githubusercontent.com/lmtani/pumbaa/main/install.sh | bash
```

### Manual Download

Download the latest binary from [GitHub Releases](https://github.com/lmtani/pumbaa/releases) for your platform.

## Usage

### Dashboard (Interactive TUI)

Browse and manage workflows with an interactive terminal interface:

```bash
pumbaa dashboard
# or
pumbaa dash
```

**Key bindings:**
| Key | Action |
|-----|--------|
| `↑/↓` | Navigate workflows |
| `Enter` | Open workflow in debug view |
| `a` | Abort running workflow |
| `s` | Cycle status filter |
| `/` | Filter by name |
| `r` | Refresh list |
| `q` | Quit |

### Debug (Workflow Inspector)

Explore workflow execution with an interactive tree view:

```bash
# From Cromwell server
pumbaa workflow debug --id <workflow-id>

# From local metadata file
pumbaa workflow debug --file metadata.json
```

**Key bindings:**
| Key | Action |
|-----|--------|
| `↑/↓` | Navigate tree |
| `←/→` | Collapse/expand nodes |
| `Tab` | Switch panels |
| `1` | View task inputs |
| `2` | View task outputs |
| `3` | View task command |
| `4` | View logs |
| `t` | View timeline |
| `y` | Copy Docker image |
| `?` | Help |

### Submit Workflow

```bash
pumbaa workflow submit \
  --workflow main.wdl \
  --inputs inputs.json \
  --options options.json
```

### Query Workflows

```bash
# List recent workflows
pumbaa workflow query

# Filter by status
pumbaa workflow query --status Running --status Failed

# Filter by name
pumbaa workflow query --name MyWorkflow
```

### Get Metadata

```bash
pumbaa workflow metadata <workflow-id>
pumbaa workflow metadata <workflow-id> --verbose
```

### Abort Workflow

```bash
pumbaa workflow abort <workflow-id>
```

### Bundle WDL Dependencies

Package a WDL workflow with all its imports into a single ZIP file:

```bash
pumbaa bundle --workflow main.wdl --output <name>
# will create name.wdl and name.zip in the specified output path
```

## Configuration

Set the Cromwell server URL:

```bash
# Via flag
pumbaa --host http://cromwell:8000 dashboard
```
