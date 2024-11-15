#!/bin/bash

set -e

# Determine OS and architecture
OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m)

# Normalize OS and architecture names
case $OS in
    darwin) OS="Darwin" ;;
    linux) OS="Linux" ;;
    windows) OS="Windows" ;;
esac

case $ARCH in
    x86_64) ARCH="x86_64" ;;
    aarch64) ARCH="arm64" ;;
    arm64) ARCH="arm64" ;;
esac

# Get the latest version if not specified
if [ -z "${VERSION}" ]; then
    VERSION=$(curl -s https://api.github.com/repos/idsulik/helm-cel/releases/latest | grep '"tag_name":' | cut -d'"' -f4)
fi
VERSION=${VERSION#v}  # Remove 'v' prefix if present

# Construct file name and URL
ARCHIVE="helm-cel_${VERSION}_${OS}_${ARCH}"
if [ "$OS" = "Windows" ]; then
    ARCHIVE="${ARCHIVE}.zip"
else
    ARCHIVE="${ARCHIVE}.tar.gz"
fi

URL="https://github.com/idsulik/helm-cel/releases/download/v${VERSION}/${ARCHIVE}"
echo "Downloading $URL"

# Create bin directory
mkdir -p "$HELM_PLUGIN_DIR/bin"

# Download and extract
if [ "$OS" = "Windows" ]; then
    curl -sSL -o "${ARCHIVE}" "${URL}"
    unzip -o "${ARCHIVE}" -d bin/
    rm "${ARCHIVE}"
else
    curl -sSL "${URL}" | tar xzf - -C bin/
fi

# Make binary executable (not needed for Windows)
if [ "$OS" != "Windows" ]; then
    chmod +x "$HELM_PLUGIN_DIR/bin/helm-cel"
fi

echo "Helm CEL plugin is installed successfully!"