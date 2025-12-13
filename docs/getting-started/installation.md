# Installation

There are several ways to install Pumbaa depending on your platform and preferences.

## Quick Install (Recommended)

The easiest way to install Pumbaa on Linux or macOS:

```bash
curl -sSL https://raw.githubusercontent.com/lmtani/pumbaa/main/install.sh | bash
```

This script will:

1. Detect your operating system and architecture
2. Download the appropriate binary from GitHub Releases
3. Install it to `/usr/local/bin/pumbaa`
4. Make it executable

!!! tip "Verify Installation"
    After installation, verify it works:
    ```bash
    pumbaa --version
    ```

## Manual Download

For more control over the installation:

### Step 1: Download

Visit [GitHub Releases](https://github.com/lmtani/pumbaa/releases/latest) and download the appropriate binary for your platform:

Choose the release asset matching your OS and CPU architecture. Common assets on Releases are shown below:

| Platform | Architecture | Release asset |
|---|---:|---|
| Linux | x86_64 | `pumbaa_Linux_x86_64.tar.gz` |
| Linux | ARM64 | `pumbaa_Linux_arm64.tar.gz` |
| macOS | x86_64 | `pumbaa_Darwin_x86_64.tar.gz` |
| macOS | ARM64 | `pumbaa_Darwin_arm64.tar.gz` |

Download the matching asset from the Releases page (replace the filename in the examples below with the one you downloaded).

### Step 2: Install

=== "Linux/macOS"

    ```bash
    # Download (replace URL with your platform)
    # Example (tar.gz asset)
    wget https://github.com/lmtani/pumbaa/releases/latest/download/pumbaa_Linux_x86_64.tar.gz

    # Extract (creates `pumbaa` binary)
    tar -xzf pumbaa_Linux_x86_64.tar.gz

    # Make executable and install
    chmod +x pumbaa
    sudo mv pumbaa /usr/local/bin/pumbaa
    ```

## Troubleshooting

### Permission Denied

If you get "permission denied" when running pumbaa:

```bash
chmod +x /path/to/pumbaa
```

### Command Not Found

Ensure the installation directory is in your PATH:

```bash
# Check current PATH
echo $PATH

# Add to PATH (add to ~/.bashrc or ~/.zshrc for persistence)
export PATH=$PATH:/usr/local/bin
```

### macOS Security Warning

On macOS, you might see a security warning. To allow the app:

1. System Preferences → Security & Privacy → General
2. Click "Allow Anyway" next to the pumbaa message
3. Or run: `xattr -d com.apple.quarantine /usr/local/bin/pumbaa`

## Next Steps

- [Configuration](configuration.md) - Set up Pumbaa for your environment
- [Quick Start](quick-start.md) - Run your first commands
