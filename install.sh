#!/bin/bash
#
# Pumbaa Installer Script
# Usage: curl -sSL https://raw.githubusercontent.com/lmtani/pumbaa/main/install.sh | bash
#

set -e

REPO="lmtani/pumbaa"
BINARY_NAME="pumbaa"
INSTALL_DIR="${INSTALL_DIR:-/usr/local/bin}"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

error() {
    echo -e "${RED}[ERROR]${NC} $1"
    exit 1
}

# Detect OS
detect_os() {
    case "$(uname -s)" in
        Linux*)     OS="linux";;
        Darwin*)    OS="darwin";;
        MINGW*|MSYS*|CYGWIN*) OS="windows";;
        *)          error "Unsupported operating system: $(uname -s)";;
    esac
}

# Detect architecture
detect_arch() {
    case "$(uname -m)" in
        x86_64|amd64)   ARCH="amd64";;
        arm64|aarch64)  ARCH="arm64";;
        *)              error "Unsupported architecture: $(uname -m)";;
    esac
}

# Get latest release version from GitHub
get_latest_version() {
    LATEST_VERSION=$(curl -sL "https://api.github.com/repos/${REPO}/releases/latest" | grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/')
    if [ -z "$LATEST_VERSION" ]; then
        error "Failed to get latest version from GitHub"
    fi
}

# Download and install
install() {
    detect_os
    detect_arch
    get_latest_version

    info "Installing ${BINARY_NAME} ${LATEST_VERSION} for ${OS}/${ARCH}..."

    # Build download URL
    EXTENSION=""
    if [ "$OS" = "windows" ]; then
        EXTENSION=".exe"
    fi
    
    FILENAME="${BINARY_NAME}-${OS}-${ARCH}${EXTENSION}"
    DOWNLOAD_URL="https://github.com/${REPO}/releases/download/${LATEST_VERSION}/${FILENAME}"

    # Create temp directory
    TMP_DIR=$(mktemp -d)
    trap "rm -rf ${TMP_DIR}" EXIT

    info "Downloading from ${DOWNLOAD_URL}..."
    if ! curl -sL -o "${TMP_DIR}/${BINARY_NAME}" "${DOWNLOAD_URL}"; then
        error "Failed to download ${BINARY_NAME}"
    fi

    # Make executable
    chmod +x "${TMP_DIR}/${BINARY_NAME}"

    # Install to destination
    if [ -w "$INSTALL_DIR" ]; then
        mv "${TMP_DIR}/${BINARY_NAME}" "${INSTALL_DIR}/${BINARY_NAME}"
    else
        info "Requesting sudo to install to ${INSTALL_DIR}..."
        sudo mv "${TMP_DIR}/${BINARY_NAME}" "${INSTALL_DIR}/${BINARY_NAME}"
    fi

    info "Successfully installed ${BINARY_NAME} to ${INSTALL_DIR}/${BINARY_NAME}"
    
    # Verify installation
    if command -v ${BINARY_NAME} &> /dev/null; then
        info "Version: $(${BINARY_NAME} --version)"
    else
        warn "${INSTALL_DIR} may not be in your PATH"
        warn "Add it with: export PATH=\"\$PATH:${INSTALL_DIR}\""
    fi

    echo ""
    info "Run '${BINARY_NAME} --help' to get started!"
}

install
