# Installation

Install Pumbaa — a single binary with no dependencies.

---

## :zap: Quick Install (Recommended)

=== ":material-linux: Linux / :material-apple: macOS"

    ```bash
    curl -sSL https://raw.githubusercontent.com/lmtani/pumbaa/main/install.sh | bash
    ```

This script will:

1. :desktop_computer: Detect your OS and architecture
2. :arrow_down: Download the appropriate binary from GitHub Releases
3. :package: Install it to `/usr/local/bin/pumbaa`
4. :white_check_mark: Make it executable

!!! success "Verify Installation"
    ```bash
    pumbaa --version
    ```

---

## :package: Manual Download

If you prefer to install manually:

### Step 1: Download from GitHub Releases

[:material-github: Go to Releases](https://github.com/lmtani/pumbaa/releases/latest){ .md-button .md-button--primary }

| Platform | Architecture | Asset |
|:---:|:---:|---|
| :material-linux: Linux | x86_64 | `pumbaa_Linux_x86_64.tar.gz` |
| :material-linux: Linux | ARM64 | `pumbaa_Linux_arm64.tar.gz` |
| :material-apple: macOS | x86_64 | `pumbaa_Darwin_x86_64.tar.gz` |
| :material-apple: macOS | ARM64 | `pumbaa_Darwin_arm64.tar.gz` |

### Step 2: Extract and Install

```bash
# Extract
tar -xzf pumbaa_*.tar.gz

# Install
chmod +x pumbaa
sudo mv pumbaa /usr/local/bin/
```

---

## :bug: Troubleshooting

??? warning "Permission Denied"
    ```bash
    chmod +x /path/to/pumbaa
    ```

??? warning "Command Not Found"
    Add to your PATH:
    ```bash
    export PATH=$PATH:/usr/local/bin
    ```

??? warning "macOS Security Warning"
    ```bash
    xattr -d com.apple.quarantine /usr/local/bin/pumbaa
    ```

---

## :books: Next Steps

- [:material-cog: Configuration](configuration.md) — Set up Pumbaa
- [:material-play: Quick Start](quick-start.md) — Run your first commands
