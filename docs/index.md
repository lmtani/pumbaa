# Pumbaa :boar:

**Modern CLI for Cromwell workflow management and WDL files.**

[:material-github: GitHub](https://github.com/lmtani/pumbaa){ .md-button }
[:material-download: Installation](getting-started/installation.md){ .md-button .md-button--primary }

---

## :dart: What is Pumbaa?

Pumbaa is a command-line tool that simplifies interaction with [Cromwell](https://cromwell.readthedocs.io/) — an execution engine for bioinformatics workflows.

!!! tip "Who is it for?"
    Bioinformaticians and developers running WDL pipelines who need practical tools for monitoring, debugging, and efficiency analysis.

---


## :rocket: Quick Start

=== "1. Install"

    ```bash
    curl -sSL https://raw.githubusercontent.com/lmtani/pumbaa/main/install.sh | bash
    ```

=== "2. Configure"

    ```bash
    pumbaa config init
    ```

=== "3. Use"

    ```bash
    pumbaa dashboard
    ```

---


## :sparkles: Features

<div class="grid cards" markdown>

-   :material-view-dashboard: **Interactive Dashboard**
    
    View and manage workflows in a terminal UI (TUI) with real-time updates.
    
    ```bash
    pumbaa dashboard
    ```

-   :material-robot: **AI Chat**
    
    Query workflows and read GCS files using natural language.
    
    ```bash
    pumbaa chat
    ```

-   :material-chart-timeline: **Efficiency Analysis**
    
    Compare actual execution time vs. allocated resources. Identify tasks with CPU/memory over-provisioning.

-   :material-package-variant: **WDL Bundling**
    
    Package your workflow with all dependencies into a single distributable file.
    
    ```bash
    pumbaa bundle --workflow main.wdl
    ```

</div>

---

## :zap: Highlights

| Feature | Description |
|:---:|---|
| :material-console: **Terminal-first** | Rich TUI interfaces for power users |
| :material-package: **Portable** | Single binary, no dependencies |
| :material-speedometer: **Efficient** | Resource utilization analysis for cost optimization |
| :material-language-go: **Native Go** | Fast, compiled, cross-platform |

---

## :books: Next Steps

- [:material-download: Installation](getting-started/installation.md) — Detailed instructions
- [:material-play: Quick Start](getting-started/quick-start.md) — Get started
- [:material-cog: Configuration](getting-started/configuration.md) — Options and providers
- [:material-star: Features](features/dashboard.md) — All features in detail

---

## :handshake: Support

- [:material-github: Issues](https://github.com/lmtani/pumbaa/issues) — Bugs and feature requests
- [:material-forum: Discussions](https://github.com/lmtani/pumbaa/discussions) — Questions and ideas
