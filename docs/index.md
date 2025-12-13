# Pumbaa 🐗

A powerful CLI tool for interacting with [Cromwell](https://cromwell.readthedocs.io/) workflow engine and managing WDL (Workflow Description Language) files.

## What is Pumbaa?

Pumbaa provides an intuitive command-line interface to:

- 🎯 **Manage workflows** with an interactive terminal dashboard (TUI)
- 🔍 **Debug workflows** with detailed execution views and preemption analysis
- 📦 **Bundle WDL files** with all dependencies into portable packages
- 📊 **View metadata**, logs, timing in a user-friendly format

## Key Features

### Interactive Dashboard

Browse and manage all your workflows in a beautiful terminal UI with real-time updates:

```bash
pumbaa dashboard
```

![Dashboard Screenshot](assets/dashboard.png)

### Debug View

Explore workflow execution in detail with an interactive tree view showing tasks, calls, and timing:

```bash
pumbaa workflow debug --id <workflow-id>
```

![Debug View Screenshot](assets/debug.png)

### WDL Bundling

Package your WDL workflow with all imports into a single distributable file:

```bash
pumbaa bundle --workflow main.wdl --output my-workflow
```

## Why Pumbaa?

- **💻 Terminal-first** - Beautiful TUI interfaces for power users
- **📦 Portable** - Single binary with no dependencies
- **🎨 User-friendly** - Intuitive commands and a practical TUI for inspecting and debugging workflows

## Next Steps

- [Installation Guide](getting-started/installation.md) - Detailed installation instructions
- [Quick Start](getting-started/quick-start.md) - Get up and running in 5 minutes
- [Configuration](getting-started/configuration.md) - Configure Pumbaa for your environment
- [Features](features/dashboard.md) - Explore all features in detail

## Need Help?

- [GitHub Issues](https://github.com/lmtani/pumbaa/issues) - Report bugs or request features
- [Discussions](https://github.com/lmtani/pumbaa/discussions) - Ask questions and share ideas
