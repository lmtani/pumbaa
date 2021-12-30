# Cromwell CLI [![codecov](https://codecov.io/gh/lmtani/cromwell-cli/branch/main/graph/badge.svg?token=IZHS203UA7)](https://codecov.io/gh/lmtani/cromwell-cli)


Command line interface for [Cromwell Server](https://cromwell.readthedocs.io/en/stable/).

## Quickstart

```bash
# Install
curl https://raw.githubusercontent.com/lmtani/cromwell-cli/main/install.sh | bash

# ⚙️ Submit a job
cromwell-cli s -w sample/wf.wdl -i sample/wf.inputs.json
# Operation= a-new-uuid , Status=Submitted

# ⚙️ Query jobs history
cromwell-cli q
# +-----------+------+-------------------+----------+---------+
# | OPERATION | NAME |       START       | DURATION | STATUS  |
# +-----------+------+-------------------+----------+---------+
# | aaa       | wf   | 2021-03-22 13h06m | 0s       | Running |
# +-----------+------+-------------------+----------+---------+
# - Found 1 workflows

# ⚙️ Check metadata
cromwell-cli m -o <operation>
# +-------------------+---------+----------+--------+
# |       TASK        | ATTEMPT | ELAPSED  | STATUS |
# +-------------------+---------+----------+--------+
# | RunHelloWorkflows | 1       | 7.515s   | Done   |
# | RunHelloWorkflows | 1       | 7.514s   | Done   |
# | SayGoodbye        | 1       | 720h0m0s | Done   |
# | SayHello          | 1       | 720h0m0s | Done   |
# | SayHelloCache     | 1       | 720h0m0s | Done   |
# +-------------------+---------+----------+--------+

# ⚙️ Check outputs
cromwell-cli o -o <operation>
# {
#    "output_path": "/path/to/output.txt"
# }

# ⚙️ Kill a running job
cromwell-cli k -o <operation>

# ⚙️ Navigate on Workflow metadata
cromwell-cli n -o <operation>

# ⚙️ Wait until job stop running
cromwell-cli w -o <operation>

# ⚙️ Check for Google Cloud Platform resource usage.
cromwell-cli gcp resources -o <operation>
# +---------------+---------------+------------+---------+
# |   RESOURCE    | NORMALIZED TO | PREEMPTIVE | NORMAL  |
# +---------------+---------------+------------+---------+
# | CPUs          | 1 hour        | 1440.00    | 720.00  |
# | Memory (GB)   | 1 hour        | 2880.00    | 1440.00 |
# | HDD disk (GB) | 1 month       | 20.00      | -       |
# | SSD disk (GB) | 1 month       | 20.00      | 20.00   |
# +---------------+---------------+------------+---------+
# - Tasks with cache hit: 1
# - Total time with running VMs: 2160h
```

> **Obs:** You need to point to [Cromwell](https://github.com/broadinstitute/cromwell/releases/tag/53.1) server in order to make all comands work. E.g.: `java -jar /path/to/cromwell.jar server`

### Example: Cromwell behind Google Indentity Aware Proxy

```bash
GOOGLE_APPLICATION_CREDENTIALS=/path/to/your/google/service-account.json
HOST="https://your-cromwell.dev"
AUDIENCE="Expected audience"
cromwell-cli --host "${HOST}" --iap "${AUDIENCE}" query
```

### Others

Check these other repositories if you don't need Google authentication:

- https://github.com/broadinstitute/cromshell
- https://github.com/stjudecloud/oliver