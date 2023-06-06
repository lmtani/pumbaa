# Pumbaa

[![Go Report Card](https://goreportcard.com/badge/github.com/lmtani/pumbaa)](https://goreportcard.com/report/github.com/lmtani/pumbaa)
[![codecov](https://codecov.io/gh/lmtani/pumbaa/branch/main/graph/badge.svg?token=IZHS203UA7)](https://codecov.io/gh/lmtani/pumbaa)
[![DeepSource](https://deepsource.io/gh/lmtani/pumbaa.svg/?label=active+issues&show_trend=true&token=AqgzwJfwaA6RBPpVTGK11it0)](https://deepsource.io/gh/lmtani/pumbaa/?ref=repository-badge)


A command line interface for Cromwell Server.

This program was created to:

- Facilitate the installation and configuration of a Cromwell Server with a local backend using Docker.
- Provide the functionality to reuse already processed jobs by default, via the Call Cache mechanism.
- Provide ways to interact with the server, such as submitting, querying, and inspecting jobs.
- Familiarize myself with the Go language.

The [Broad Institute has its own CLI](https://github.com/broadinstitute/cromshell), be sure to check it out.

But don't forget to give us a star if ours has been helpful 😉

## Quickstart

You can download the binary from the [releases page](https://github.com/lmtani/pumbaa/releases) for your platform or install it by running the following command:

```bash
curl https://raw.githubusercontent.com/lmtani/pumbaa/main/install.sh | bash
```

It's allowed to install the binary in any location, but you will need to set variable PREFIX when running the install script.

```bash
curl https://raw.githubusercontent.com/lmtani/pumbaa/main/install.sh | PREFIX=/home/taniguti/bin bash
```

This way you don't need to provide privileged access.

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
- [x] Have a cool name

## Examples

_Obs:_ the examples below are from a previous release, but the commands are the same. I will update them soon.

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

This is a very specific use case, but it's here. If you are using a Cromwell server behind Google Identity Aware Proxy (IAP), you can use the `--iap` flag to make requests to it. You will need to provide the expected audience of the token, which is the client_id of your oauth. For example:

You will also need to set the `GOOGLE_APPLICATION_CREDENTIALS` environment variable to the path of your Google service account JSON file.

```bash
GOOGLE_APPLICATION_CREDENTIALS=/path/to/your/google/service-account.json
HOST="https://your-cromwell.dev"
AUDIENCE="Expected audience"
pumbaa --host "${HOST}" --iap "${AUDIENCE}" query
```
