#!/bin/bash
set -e

# Development mode: skip download if env var is set
if [ -n "$HELM_CEL_PLUGIN_NO_INSTALL_HOOK" ]; then
    echo "Development mode: not downloading versioned release."
    exit 0
fi

# Get version from plugin.yaml (assumes version: "x.y.z" is present)
VERSION=$(grep '^version:' plugin.yaml | cut -d '"' -f 2)

# Detect OS
OS=""
case "$(uname -s)" in
    Darwin)
        OS="Darwin"
        ;;
    Linux)
        OS="Linux"
        ;;
    MINGW*|MSYS*|CYGWIN*|Windows_NT)
        OS="Windows"
        ;;
    *)
        echo "Unsupported OS: $(uname -s)"
        exit 1
        ;;
esac

# Detect ARCH
ARCH=""
case "$(uname -m)" in
    x86_64)
        ARCH="amd64"
        ;;
    aarch64|arm64)
        ARCH="arm64"
        ;;
    armv6*)
        ARCH="armv6"
        ;;
    armv7*)
        ARCH="armv7"
        ;;
    *)
        echo "Failed to detect target architecture: $(uname -m)"
        exit 1
        ;;
esac

ARCHIVE="helm-cel_${VERSION}_${OS}_${ARCH}"
if [ "$OS" = "Windows" ]; then
    ARCHIVE="${ARCHIVE}.zip"
else
    ARCHIVE="${ARCHIVE}.tar.gz"
fi

URL="https://github.com/idsulik/helm-cel/releases/download/v${VERSION}/${ARCHIVE}"
echo "Downloading $URL"

# Clean and create bin directory
rm -rf "$HELM_PLUGIN_DIR/bin"
mkdir -p "$HELM_PLUGIN_DIR/bin"
chmod 755 "$HELM_PLUGIN_DIR/bin"

# Download and extract
if [ "$OS" = "Windows" ]; then
    curl -sSL -o "$ARCHIVE" "$URL"
    unzip -o "$ARCHIVE" -d "$HELM_PLUGIN_DIR/bin/"
    rm "$ARCHIVE"
else
    curl -sSL "$URL" | tar xzf - -C "$HELM_PLUGIN_DIR/bin/"
fi

# Make binary executable (not needed for Windows)
if [ "$OS" != "Windows" ]; then
    chmod +x "$HELM_PLUGIN_DIR/bin/helm-cel"
fi

echo "Helm CEL plugin v$VERSION is installed successfully!"
