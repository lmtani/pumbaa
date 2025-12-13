# Configuration

Pumbaa can be configured through command-line flags, environment variables, or a configuration file.

## Cromwell Server

The most important configuration is your Cromwell server URL.

### Environment Variable (Recommended)

```bash
export CROMWELL_HOST=http://cromwell.example.com:8000
```

Add to your shell profile (`~/.bashrc`, `~/.zshrc`, etc.) for persistence:

```bash
echo 'export CROMWELL_HOST=http://cromwell.example.com:8000' >> ~/.bashrc
```

### Command-Line Flag

```bash
pumbaa --host http://cromwell.example.com:8000 dashboard
```

!!! note "Default value"
    The default host is `http://localhost:8000`.


### Configuration Priority

Settings are applied in this order (later overrides earlier):

1. Default values
2. Environment variables
3. Command-line flags

## Available Configuration

## Authentication

Pumbaa assumes a direct connection to a reachable Cromwell server; it does not perform authentication itself.

If your Cromwell instance runs inside Kubernetes, expose it locally and point Pumbaa at localhost. Example using port-forward:

```bash
# forward the Cromwell service to local port 8000
kubectl -n <namespace> port-forward svc/cromwell 8000:8000

# then in another terminal
export CROMWELL_HOST=http://localhost:8000
pumbaa dashboard
```

You can also pass the host directly with `--host`:

```bash
pumbaa --host http://localhost:8000 dashboard
```

## Next Steps

- [Quick Start](quick-start.md) - Run your first commands
- [Dashboard](../features/dashboard.md) - Learn about the interactive dashboard
