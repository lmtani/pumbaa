# :pig2: Cromwell CLI :rocket:

Command line interface for Cromwell Server. Check these other repositories if you don't need Bearer token:

- https://github.com/broadinstitute/cromshell
- https://github.com/stjudecloud/oliver

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

# Navigate on Workflow metadata
cromwell-cli n -o <operation>

# View monitoring scripts log. Pipe to "less -S" if it has lot of lines
cat <monitoring.log> | grep -v "#" | cromwell-cli gce monitoring -r <cpu|mem|disk>
```

> **Obs:** You need to point to [Cromwell](https://github.com/broadinstitute/cromwell/releases/tag/53.1) server in order to make all comands work. E.g.: `java -jar /path/to/cromwell.jar server`

### Example: Cromwell behind Google Indentity Aware Proxy

```bash
TOKEN=$(gcloud auth print-identity-token --impersonate-service-account <your@service-account.iam.gserviceaccount.com> --audiences <oauth-client-id.googleusercontent.com> --include-email)
HOST="https://your-cromwell.dev"
cromwell-cli --host "${HOST}" --token "${TOKEN}" query
```
