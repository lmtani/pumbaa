# Cromwell CLI [![codecov](https://codecov.io/gh/lmtani/cromwell-cli/branch/main/graph/badge.svg?token=IZHS203UA7)](https://codecov.io/gh/lmtani/cromwell-cli)


Command line interface for [Cromwell Server](https://cromwell.readthedocs.io/en/stable/).

## Quickstart

```bash
# Install
curl https://raw.githubusercontent.com/lmtani/cromwell-cli/main/install.sh | bash

cromwell-cli --help
```

> **Obs:** You need to point to [Cromwell](https://github.com/broadinstitute/cromwell) server in order to make all commands work. E.g.: running `java -jar /path/to/cromwell.jar server` in your localhost.

### Example: Cromwell behind Google Identity Aware Proxy

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