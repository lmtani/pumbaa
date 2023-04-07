# Cromwell CLI

[![codecov](https://codecov.io/gh/lmtani/cromwell-cli/branch/main/graph/badge.svg?token=IZHS203UA7)](https://codecov.io/gh/lmtani/cromwell-cli)
 [![DeepSource](https://deepsource.io/gh/lmtani/cromwell-cli.svg/?label=active+issues&show_trend=true&token=AqgzwJfwaA6RBPpVTGK11it0)](https://deepsource.io/gh/lmtani/cromwell-cli/?ref=repository-badge)

---

This program was created to:

- Facilitate the installation and configuration of a Cromwell Server with a local backend using Docker.
- Provide the functionality to reuse already processed jobs by default, via the Call Cache mechanism.
- Provide ways to interact with the server, such as submitting, querying, and inspecting jobs.
- Familiarize myself with the Go language.

The [Broad Institute has its own CLI](https://github.com/broadinstitute/cromshell), be sure to check it out.

But don't forget to give us a star if ours has been helpful 😉

## Quickstart

```bash
# Install
curl https://raw.githubusercontent.com/lmtani/cromwell-cli/main/install.sh | bash

cromwell-cli --help
```

## Features

- [x] Start Cromwell Server locally
- [x] List workflows by name
- [x] Submit a workflow
- [x] Abort a workflow
- [x] Navigate through workflow metadata
- [x] Get metadata
- [x] Get outputs
- [x] Get inputs
- [x] Make requests to a remote Cromwell Server protected by IAP (Google Identity Aware Proxy)
- [x] For Google Cloud backend jobs: estimate resource usage
- [ ] Have a cool name

### Example: Cromwell behind Google Identity Aware Proxy

```bash
GOOGLE_APPLICATION_CREDENTIALS=/path/to/your/google/service-account.json
HOST="https://your-cromwell.dev"
AUDIENCE="Expected audience"
cromwell-cli --host "${HOST}" --iap "${AUDIENCE}" query
```
