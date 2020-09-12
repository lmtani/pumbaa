# Cromwell CLI

Command line interface for Cromwell server.

## Quickstart

```bash
# Download
wget https://github.com/lmtani/cromwell-cli/releases/download/v0.1/cromwell-cli-linux-amd64

# Put in a directory of your PATH (may require sudo)
mv cromwell-cli-linux-amd64 /usr/bin/cromwell-cli

# Submit a job
cromwell-cli s -w sample/wf.wdl -i sample/wf.inputs.json

# Query jobs history
cromwell-cli q

# Kill a running job
cromwell-cli k -o <operation>
```

> **Obs:** You need to point to [Cromwell](https://github.com/broadinstitute/cromwell/releases/tag/53.1) server in order to make all comands work. E.g.: `java -jar /path/to/cromwell.jar server`

## Go ecosystem

- [x] Command line [urfave/cli/v2](https://github.com/urfave/cli)
- [x] Logging  [logrus](https://github.com/uber-go/zap)
- [x] Http request  [net/http](https://golang.org/pkg/net/http/)
- [x] Pretty format terminal tables [olekukonko/tablewriter](https://github.com/olekukonko/tablewriter)

## Cromwell server interactions

- [x] Submit a job (with dependencies)
- [x] Kill a job
- [x] Query job status
- [x] Get jobs by name
- [ ] Query job outputs
- [ ] Add config for host url
- [ ] Make binary available for MacOS
