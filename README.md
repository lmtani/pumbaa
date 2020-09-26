# Cromwell CLI

Command line interface for Cromwell server.

## Quickstart

```bash
# Install
curl https://raw.githubusercontent.com/lmtani/cromwell-cli/master/install.sh | bash

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
```

> **Obs:** You need to point to [Cromwell](https://github.com/broadinstitute/cromwell/releases/tag/53.1) server in order to make all comands work. E.g.: `java -jar /path/to/cromwell.jar server`

### Example: Cromwell behind Google Indentity Aware Proxy

```bash
TOKEN=$(gcloud auth print-identity-token --impersonate-service-account <your@service-account.iam.gserviceaccount.com> --audiences <oauth-client-id.googleusercontent.com> --include-email)
HOST="https://your-cromwell.dev"
cromwell-cli --host "${HOST}" --token "${TOKEN}" query
```

## Go ecosystem

- [x] Command line [urfave/cli/v2](https://github.com/urfave/cli)
- [x] Logging  [Zap](https://github.com/uber-go/zap)
- [x] Http request  [net/http](https://golang.org/pkg/net/http/)
- [x] Pretty format terminal tables [olekukonko/tablewriter](https://github.com/olekukonko/tablewriter)

## Cromwell server interactions

- [x] Submit a job (with dependencies)
- [x] Kill a job
- [x] Query job status
- [x] Get jobs by name
- [x] Allow to pass an Bearer token from the environment
- [x] Make binary available for MacOS and Windows
- [x] Add config for host url
- [x] Query job outputs

