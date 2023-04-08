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

You can download the binary from the [releases page]() for your platform or install it by running the following command:

```bash
curl https://raw.githubusercontent.com/lmtani/cromwell-cli/main/install.sh | bash
```

> Note: The script will download the latest release for your platform and add it to `/usr/local/bin/`. If you want to install it in a different location, you can use a environment variable to specify it. For example:
>
> `curl https://raw.githubusercontent.com/lmtani/cromwell-cli/main/install.sh | PREFIX=/home/taniguti/bin bash`

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

## Examples

### Local server deploy

You need to open a new terminal after starting the server. Press CTRL+c when you want to shut it down.
[![asciicast](https://asciinema.org/a/9oHGhq7t2CGpBMl3M0vicA67Q.svg)](https://asciinema.org/a/9oHGhq7t2CGpBMl3M0vicA67Q)

### Submit workflow

[![asciicast](https://asciinema.org/a/rSGGiYwAOITWNx4gX4Qtq8h8F.svg)](https://asciinema.org/a/rSGGiYwAOITWNx4gX4Qtq8h8F)

### Query workflows
[![asciicast](https://asciinema.org/a/JTQR8Va7bnHhYIZ5uxSWfZBse.svg)](https://asciinema.org/a/JTQR8Va7bnHhYIZ5uxSWfZBse)

### Navigate through workflow metadata

[![asciicast](https://asciinema.org/a/yxDZp4H2DYAWStjS2nPvsAIqM.svg)](https://asciinema.org/a/yxDZp4H2DYAWStjS2nPvsAIqM)


### Cromwell behind Google Identity Aware Proxy

This is a very specific use case, but it's here. If you are using a Cromwell server behind Google Identity Aware Proxy (IAP), you can use the `--iap` flag to make requests to it. You will need to provide the expected audience of the token, which is the URL of the server. For example:

You will also need to set the `GOOGLE_APPLICATION_CREDENTIALS` environment variable to the path of your Google service account JSON file.

```bash
GOOGLE_APPLICATION_CREDENTIALS=/path/to/your/google/service-account.json
HOST="https://your-cromwell.dev"
AUDIENCE="Expected audience"
cromwell-cli --host "${HOST}" --iap "${AUDIENCE}" query
```
