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

- **Linux (x86_64)**: `pumbaa-linux-amd64`
- **Linux (ARM64)**: `pumbaa-linux-arm64`
- **macOS (Intel)**: `pumbaa-darwin-amd64`
- **macOS (Apple Silicon)**: `pumbaa-darwin-arm64`

### Step 2: Install

=== "Linux/macOS"

    ```bash
    # Download (replace URL with your platform)
    wget https://github.com/lmtani/pumbaa/releases/latest/download/pumbaa-linux-amd64
    
    # Make executable
    chmod +x pumbaa-linux-amd64
    
    # Move to PATH
    sudo mv pumbaa-linux-amd64 /usr/local/bin/pumbaa
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
