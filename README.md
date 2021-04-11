# :pig2: Cromwell CLI :rocket:

[![codecov](https://codecov.io/gh/lmtani/cromwell-cli/branch/master/graph/badge.svg?token=IZHS203UA7)](https://codecov.io/gh/lmtani/cromwell-cli)

Command line interface for Cromwell Server. Check these other repositories if you don't need Bearer token:

- https://github.com/broadinstitute/cromshell
- https://github.com/stjudecloud/oliver

## Quickstart

```bash
# Install
curl https://raw.githubusercontent.com/lmtani/cromwell-cli/master/install.sh | bash

# Commands
cromwell-cli -h
# NAME:
#    cromwell-cli - Command line interface for Cromwell Server

# USAGE:
#    cromwell-cli [global options] command [command options] [arguments...]

# COMMANDS:
#    version, v   Cromwell-CLI version
#    query, q     Query workflows
#    submit, s    Submit a workflow and its inputs to Cromwell
#    inputs, i    Recover inputs from the specified workflow (JSON)
#    kill, k      Kill a running job
#    metadata, m  Inspect workflow details (table)
#    outputs, o   Query workflow outputs (JSON)
#    navigate, n  Navigate through metadata data
#    gcp, g       Use commands specific for Google backend
#    help, h      Shows a list of commands or help for one command

# GLOBAL OPTIONS:
#    --iap value   Uses your defauld Google Credentials to obtains an access token to this audience.
#    --host value  Url for your Cromwell Server (default: "http://127.0.0.1:8000")
#    --help, -h    show help (default: false)

# Submit a job
cromwell-cli s -w sample/wf.wdl -i sample/wf.inputs.json

# Query jobs history
cromwell-cli q

# Kill a running job
cromwell-cli k -o <operation>

# Check metadata
cromwell-cli m -o <operation>

# Check outputs
cromwell-cli o -o <operation>

# Navigate on Workflow metadata
cromwell-cli n -o <operation>

# Wait until job stop running
cromwell-cli w -o <operation>

# Check for Google Cloud Platform resource usage.
cromwell-cli gcp resources -o <operation>
```

> **Obs:** You need to point to [Cromwell](https://github.com/broadinstitute/cromwell/releases/tag/53.1) server in order to make all comands work. E.g.: `java -jar /path/to/cromwell.jar server`

### Example: Cromwell behind Google Indentity Aware Proxy

```bash
GOOGLE_APPLICATION_CREDENTIALS=/path/to/your/google/service-account.json
HOST="https://your-cromwell.dev"
AUDIENCE="Expected audience"
cromwell-cli --host "${HOST}" --iap "${AUDIENCE}" query
```
